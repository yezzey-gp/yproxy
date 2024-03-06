package proc

import (
	"fmt"
	"io"
	"sync"

	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/crypt"
	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type YproxyLoggingReader struct {
	underlying io.ReadCloser
}

// Close implements io.ReadCloser.
func (y *YproxyLoggingReader) Close() error {
	return y.underlying.Close()
}

// Read implements io.ReadCloser.
func (y *YproxyLoggingReader) Read(p []byte) (int, error) {
	n, err := y.underlying.Read(p)
	if err != nil {
		ylogger.Zero.Error().Err(err).Msg("encounderv read error")
	}
	return n, err
}

func newLoggingReader(r io.ReadCloser) io.ReadCloser {
	return &YproxyLoggingReader{
		underlying: r,
	}
}

var _ io.ReadCloser = &YproxyLoggingReader{}

func ProcConn(s storage.StorageInteractor, cr crypt.Crypter, ycl *client.YClient) error {

	defer func() {
		_ = ycl.Conn.Close()
	}()

	pr := NewProtoReader(ycl)
	tp, body, err := pr.ReadPacket()
	if err != nil {
		_ = ycl.ReplyError(err, "failed to read request packet")
		return err
	}

	ylogger.Zero.Debug().Str("msg-type", tp.String()).Msg("recieved client request")

	switch tp {
	case message.MessageTypeCat:
		// omit first byte
		msg := message.CatMessage{}
		msg.Decode(body)
		ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("cat object")
		r, err := s.CatFileFromStorage(msg.Name)
		if err != nil {
			_ = ycl.ReplyError(err, "failed to read from external storage")

			return err
		}

		r = newLoggingReader(r)

		var contentReader io.Reader
		contentReader = r
		defer r.Close()

		if msg.Decrypt {
			ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("decrypt object ")
			contentReader, err = cr.Decrypt(r)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to decrypt object")

				return err
			}
		}
		_, err = io.Copy(ycl.Conn, contentReader)
		if err != nil {
			_ = ycl.ReplyError(err, "copy failed to compelete")
		}

	case message.MessageTypePut:

		msg := message.PutMessage{}
		msg.Decode(body)

		var w io.WriteCloser

		r, w := io.Pipe()

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {

			var ww io.WriteCloser = w
			if msg.Encrypt {
				var err error
				ww, err = cr.Encrypt(w)
				if err != nil {
					_ = ycl.ReplyError(err, "failed to encrypt")

					ycl.Conn.Close()
					return
				}
			}

			defer w.Close()
			defer wg.Done()

			for {
				tp, body, err := pr.ReadPacket()
				if err != nil {
					_ = ycl.ReplyError(err, "failed to read chunk of data")
					return
				}

				ylogger.Zero.Debug().Str("msg-type", tp.String()).Msg("recieved client request")

				switch tp {
				case message.MessageTypeCopyData:
					msg := message.CopyDataMessage{}
					msg.Decode(body)
					if n, err := ww.Write(msg.Data); err != nil {
						_ = ycl.ReplyError(err, "failed to write copy data")

						return
					} else if n != int(msg.Sz) {

						_ = ycl.ReplyError(fmt.Errorf("unfull write"), "failed to compelete request")

						return
					}
				case message.MessageTypeCommandComplete:
					msg := message.CommandCompleteMessage{}
					msg.Decode(body)

					if err := ww.Close(); err != nil {
						_ = ycl.ReplyError(err, "failed to close connection")
						return
					}

					ylogger.Zero.Debug().Msg("closing msg writer")
					return
				}
			}
		}()

		err := s.PutFileToDest(msg.Name, r)

		wg.Wait()

		if err != nil {
			_ = ycl.ReplyError(err, "failed to upload")

			return nil
		}

		_, err = ycl.Conn.Write(message.NewReadyForQueryMessage().Encode())

		if err != nil {
			_ = ycl.ReplyError(err, "failed to upload")

			return nil
		}

	case message.MessageTypeList:
		msg := message.ListMessage{}
		msg.Decode(body)

		objectMetas, err := s.ListPath(msg.Prefix)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not list objects: %s", err), "failed to compelete request")

			return nil
		}

		const chunkSize = 1000

		for i := 0; i < len(objectMetas); i += chunkSize {
			_, err = ycl.Conn.Write(message.NewObjectMetaMessage(objectMetas[i:min(i+chunkSize, len(objectMetas))]).Encode())
			if err != nil {
				_ = ycl.ReplyError(err, "failed to upload")

				return nil
			}

		}

		_, err = ycl.Conn.Write(message.NewReadyForQueryMessage().Encode())

		if err != nil {
			_ = ycl.ReplyError(err, "failed to upload")

			return nil
		}

	default:
		_ = ycl.ReplyError(nil, "wrong request type")

		return nil
	}

	return nil
}
