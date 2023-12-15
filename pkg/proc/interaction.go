package proc

import (
	"io"
	"sync"

	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/crypt"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

func ProcConn(s storage.StorageInteractor, cr crypt.Crypter, ycl *client.YClient) error {
	pr := NewProtoReader(ycl)
	tp, body, err := pr.ReadPacket()
	if err != nil {

		_ = ycl.ReplyError(err, "failed to compelete request")

		return err
	}

	ylogger.Zero.Debug().Str("msg-type", tp.String()).Msg("recieved client request")

	switch tp {
	case MessageTypeCat:
		// omit first byte
		msg := CatMessage{}
		msg.Decode(body)
		ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("cat object")
		r, err := s.CatFileFromStorage(msg.Name)
		if err != nil {
			_ = ycl.ReplyError(err, "failed to compelete request")

			return err
		}
		if msg.Decrypt {
			ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("decrypt object ")
			r, err = cr.Decrypt(r)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to compelete request")

				return err
			}
		}
		io.Copy(ycl.Conn, r)

	case MessageTypePut:

		msg := PutMessage{}
		msg.Decode(body)

		var w io.WriteCloser

		r, w := io.Pipe()

		if msg.Encrypt {
			var err error
			w, err = cr.Encrypt(w)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to encrypt")

				return ycl.Conn.Close()
			}
		}

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			for {
				tp, body, err := pr.ReadPacket()
				if err != nil {
					_ = ycl.ReplyError(err, "failed to compelete request")

					_ = ycl.Conn.Close()
					return
				}

				ylogger.Zero.Debug().Str("msg-type", tp.String()).Msg("recieved client request")

				switch tp {
				case MessageTypeCopyData:
					msg := CopyDataMessage{}
					msg.Decode(body)
					w.Write(msg.Data)
				case MessageTypeCommandComplete:
					msg := CommandCompleteMessage{}
					msg.Decode(body)
					w.Close()

					ylogger.Zero.Debug().Msg("closing msg writer")
					return
				}
			}
		}()

		err := s.PutFileToDest(msg.Name, r)

		wg.Wait()

		if err != nil {
			_ = ycl.ReplyError(err, "failed to upload")

			return ycl.Conn.Close()
		}

	default:

		_ = ycl.ReplyError(nil, "wrong request type")

		return ycl.Conn.Close()
	}

	return nil
}
