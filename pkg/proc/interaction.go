package proc

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/crypt"
	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

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
	fmt.Printf("recieved: %v\n", tp)
	fmt.Printf("type: %s\n", string(body))

	ylogger.Zero.Debug().Str("msg-type", tp.String()).Msg("recieved client request")

	switch tp {
	case message.MessageTypeCat:
		// omit first byte
		msg := message.CatMessage{}
		msg.Decode(body)

		yr := NewYRetryReader(NewRestartReader(s, msg.Name))

		var contentReader io.Reader
		contentReader = yr
		defer yr.Close()

		if msg.Decrypt {
			ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("decrypt object")
			contentReader, err = cr.Decrypt(yr)
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
		fmt.Printf("metas count %d\n", len(objectMetas))
		fmt.Printf("meta ok: %v\n", objectMetas)
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

	case message.MessageTypeCopy:
		fmt.Printf("start copy\n")
		msg := message.CopyMessage{}
		msg.Decode(body)

		//get config for old bucket
		instanceCnf, err := config.ReadInstanceConfig(msg.OldCfgPath)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not read old config: %s", err), "failed to compelete request")
			return nil
		}
		oldStorage := storage.NewStorage(
			&instanceCnf.StorageCnf,
		)
		fmt.Printf("ok new conf: %v\n", instanceCnf)
		fmt.Printf("ok old st: %v\n", oldStorage)

		//list objects
		objectMetas, err := s.ListPath(msg.Name)
		if err != nil {
			fmt.Printf("list fail %v\n", err)
			_ = ycl.ReplyError(fmt.Errorf("could not list objects: %s", err), "failed to compelete request")
			return nil
		}
		fmt.Printf("metas count %d\n", len(objectMetas))
		fmt.Printf("meta ok: %v\n", objectMetas)

		var failed []*storage.S3ObjectMeta
		for len(objectMetas) > 0 {
			fmt.Printf("while %d\n", len(objectMetas))
			for i := 0; i < len(objectMetas); i++ {
				path := strings.TrimPrefix(objectMetas[i].Path, instanceCnf.StorageCnf.StoragePrefix) //wrong prefix
				fmt.Printf("files: %v\n", path)
				//get reader
				yr := NewYRetryReader(NewRestartReader(s, path))

				var fromReader io.Reader
				fromReader = yr
				defer yr.Close()

				if msg.Decrypt {
					ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("decrypt object")
					fromReader, err = cr.Decrypt(yr)
					if err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to decrypt object")
						failed = append(failed, objectMetas[i])
						continue
					}
				}
				fmt.Printf("decrypt ok:\n")

				//reencrypt
				r, w := io.Pipe()
				mas := make([]byte, objectMetas[i].Size)

				fmt.Printf("pype ok:\n")

				var ww io.WriteCloser = w
				if msg.Encrypt {
					var err error
					ww, err = cr.Encrypt(w)
					if err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to encrypt object")
						failed = append(failed, objectMetas[i])
						fmt.Printf("encrypt fail %v\n", err)
						continue
					}
				}

				fmt.Printf("encrypt ok:\n")
				if n, err := fromReader.Read(mas); err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to read copy data")
					failed = append(failed, objectMetas[i])
					fmt.Printf("read fail %v\n", err)
					continue

				} else if n != int(objectMetas[i].Size) {
					ylogger.Zero.Error().Err(fmt.Errorf("unfull read")).Msg("failed to read copy data")
					failed = append(failed, objectMetas[i])
					fmt.Printf("encrypt fail size\n")
					continue
				}
				fmt.Printf("read ok:\n")

				if n, err := ww.Write(mas); err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to write copy data")
					failed = append(failed, objectMetas[i])
					fmt.Printf("write fail %v\n", err)
					continue

				} else if n != int(objectMetas[i].Size) {
					ylogger.Zero.Error().Err(fmt.Errorf("unfull write")).Msg("failed to write copy data")
					failed = append(failed, objectMetas[i])
					fmt.Printf("write fail size\n")
					continue
				}
				fmt.Printf("write ok:\n")

				defer w.Close() //TODO проверить ошибку
				if err := ww.Close(); err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to close writer")
					failed = append(failed, objectMetas[i])
					continue
				}
				fmt.Printf("close ok:\n")

				//write file
				err = s.PutFileToDest(msg.Name+"_copy", r) //TODO path
				if err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to upload file")
					failed = append(failed, objectMetas[i])
					continue
				}
				fmt.Printf("put file ok:\n")
			}
			objectMetas = failed
			fmt.Printf("next files: %d\n", len(objectMetas))
			failed = make([]*storage.S3ObjectMeta, 0)
		}
		fmt.Printf("finish \n")

	default:
		ylogger.Zero.Error().Any("type", tp).Msg("what tip is it")
		_ = ycl.ReplyError(nil, "wrong request type")

		return nil
	}

	return nil
}
