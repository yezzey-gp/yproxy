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
		n, err := io.Copy(ycl.Conn, contentReader)
		if err != nil {
			_ = ycl.ReplyError(err, "copy failed to compelete")
		}
		fmt.Printf("size: %d\n", n)

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
					fmt.Printf("decrypting\n")
					ylogger.Zero.Debug().Str("object-path", msg.Name).Msg("decrypt object")
					fromReader, err = cr.Decrypt(yr)
					if err != nil {
						fmt.Printf("decrypt fail: %v\n", err)
						ylogger.Zero.Error().Err(err).Msg("failed to decrypt object")
						failed = append(failed, objectMetas[i])
						continue
					}
				}
				fmt.Printf("decrypt ok:\n")

				//reencrypt
				r, w := io.Pipe()
				//mas := make([]byte, objectMetas[i].Size)

				go func() {
					defer w.Close() //TODO проверить ошибку

					fmt.Printf("pype ok:\n")

					var ww io.WriteCloser = w

					if msg.Encrypt {
						fmt.Printf("encrypting\n")
						var err error
						ww, err = cr.Encrypt(w)
						if err != nil {
							ylogger.Zero.Error().Err(err).Msg("failed to encrypt object")
							failed = append(failed, objectMetas[i])
							fmt.Printf("encrypt fail %v\n", err)
							return
						}
					}

					if _, err := io.Copy(ww, fromReader); err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to copy data")
						failed = append(failed, objectMetas[i])
						fmt.Printf("copy fail %v\n", err)
						return
					} // else if n != objectMetas[i].Size {

					// 	ylogger.Zero.Error().Err(fmt.Errorf("unfull copy")).Msg("failed to copy data")
					// 	failed = append(failed, objectMetas[i])
					// 	fmt.Printf("copy fail size meta: %d actual %d\n", objectMetas[i].Size, n)
					// 	return
					// }

					if err := ww.Close(); err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to close writer")
						failed = append(failed, objectMetas[i])
						return
					}
					fmt.Printf("close ok:\n")
				}()

				//wg.Wait()

				//write file
				err = s.PutFileToDest(path+"_copy", r) //TODO path
				if err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to upload file")
					failed = append(failed, objectMetas[i])
					continue
				}
				re, err := s.CatFileFromStorage(path+"_copy", 0)
				if err != nil {
					fmt.Printf("check fail 1 %v\n", err)
				}
				red, wr := io.Pipe()
				go func() {
					fmt.Printf("check start++++++++++++++++++++++++++++++++++\n")
					n, err := io.Copy(wr, re)
					if err != nil {
						fmt.Printf("check fail 2 %v\n", err)
					}
					if n != objectMetas[i].Size {
						fmt.Printf("check fail 3 size meta: %d actual %d\n", objectMetas[i].Size, n)
					}
					wr.Close()
				}()
				mas := make([]byte, objectMetas[i].Size)
				n, err := red.Read(mas)
				if err != nil {
					fmt.Printf("check fail 22 %v\n", err)
				}
				if n != int(objectMetas[i].Size) {
					fmt.Printf("check fail 23 size meta: %d actual %d\n", objectMetas[i].Size, n)
				}
				fmt.Printf("check success----------------------------------------------------------------\n")
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
