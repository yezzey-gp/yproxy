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

func ProcConn(s storage.StorageInteractor, cr crypt.Crypter, ycl client.YproxyClient) error {

	defer func() {
		_ = ycl.Close()
	}()

	pr := NewProtoReader(ycl)
	tp, body, err := pr.ReadPacket()
	if err != nil {
		_ = ycl.ReplyError(err, "failed to read request packet")
		return err
	}

	ylogger.Zero.Debug().Str("msg-type", tp.String()).Msg("recieved client request")

	ycl.SetOPType(byte(tp))

	switch tp {
	case message.MessageTypeCat:
		// omit first byte
		msg := message.CatMessage{}
		msg.Decode(body)

		ycl.SetExternalFilePath(msg.Name)

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

		if msg.StartOffset != 0 {
			io.CopyN(io.Discard, contentReader, int64(msg.StartOffset))
		}

		n, err := io.Copy(ycl.GetRW(), contentReader)
		if err != nil {
			_ = ycl.ReplyError(err, "copy failed to complete")
		}
		ylogger.Zero.Debug().Int64("copied bytes", n).Msg("decrypt object")

	case message.MessageTypePut:

		msg := message.PutMessage{}
		msg.Decode(body)

		ycl.SetExternalFilePath(msg.Name)

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

					ycl.Close()
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

		_, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode())

		if err != nil {
			_ = ycl.ReplyError(err, "failed to upload")

			return nil
		}

	case message.MessageTypeList:
		msg := message.ListMessage{}
		msg.Decode(body)

		ycl.SetExternalFilePath(msg.Prefix)

		objectMetas, err := s.ListPath(msg.Prefix)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not list objects: %s", err), "failed to compelete request")

			return nil
		}

		const chunkSize = 1000

		for i := 0; i < len(objectMetas); i += chunkSize {
			_, err = ycl.GetRW().Write(message.NewObjectMetaMessage(objectMetas[i:min(i+chunkSize, len(objectMetas))]).Encode())
			if err != nil {
				_ = ycl.ReplyError(err, "failed to upload")

				return nil
			}

		}

		_, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode())

		if err != nil {
			_ = ycl.ReplyError(err, "failed to upload")

			return nil
		}

	case message.MessageTypeCopy:
		msg := message.CopyMessage{}
		msg.Decode(body)

		ycl.SetExternalFilePath(msg.Name)

		//get config for old bucket
		instanceCnf, err := config.ReadInstanceConfig(msg.OldCfgPath)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not read old config: %s", err), "failed to compelete request")
			return nil
		}
		config.EmbedDefaults(&instanceCnf)
		oldStorage := storage.NewStorage(&instanceCnf.StorageCnf)
		fmt.Printf("ok new conf: %v\n", instanceCnf)

		//list objects
		objectMetas, err := oldStorage.ListPath(msg.Name)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not list objects: %s", err), "failed to compelete request")
			return nil
		}

		var failed []*storage.S3ObjectMeta
		retryCount := 0
		for len(objectMetas) > 0 && retryCount < 10 {
			retryCount++
			for i := 0; i < len(objectMetas); i++ {
				path := strings.TrimPrefix(objectMetas[i].Path, instanceCnf.StorageCnf.StoragePrefix)

				//get reader
				readerFromOldBucket := NewYRetryReader(NewRestartReader(oldStorage, path))
				var fromReader io.Reader
				fromReader = readerFromOldBucket
				defer readerFromOldBucket.Close()

				if msg.Decrypt {
					oldCr, err := crypt.NewCrypto(&instanceCnf.CryptoCnf)
					if err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to configure decrypter")
						failed = append(failed, objectMetas[i])
						continue
					}
					fromReader, err = oldCr.Decrypt(readerFromOldBucket)
					if err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to decrypt object")
						failed = append(failed, objectMetas[i])
						continue
					}
				}

				//reencrypt
				readerEncrypt, writerEncrypt := io.Pipe()

				go func() {
					defer func() {
						if err := writerEncrypt.Close(); err != nil {
							ylogger.Zero.Warn().Err(err).Msg("failed to close writer")
						}
					}()

					var writerToNewBucket io.WriteCloser = writerEncrypt

					if msg.Encrypt {
						var err error
						writerToNewBucket, err = cr.Encrypt(writerEncrypt)
						if err != nil {
							ylogger.Zero.Error().Err(err).Msg("failed to encrypt object")
							failed = append(failed, objectMetas[i])
							return
						}
					}

					if _, err := io.Copy(writerToNewBucket, fromReader); err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to copy data")
						failed = append(failed, objectMetas[i])
						return
					}

					if err := writerToNewBucket.Close(); err != nil {
						ylogger.Zero.Error().Err(err).Msg("failed to close writer")
						failed = append(failed, objectMetas[i])
						return
					}
				}()

				//write file
				err = s.PutFileToDest(path, readerEncrypt)
				if err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to upload file")
					failed = append(failed, objectMetas[i])
					continue
				}
			}
			objectMetas = failed
			fmt.Printf("failed files count: %d\n", len(objectMetas))
			failed = make([]*storage.S3ObjectMeta, 0)
		}

		if len(objectMetas) > 0 {
			fmt.Printf("failed files count: %d\n", len(objectMetas))
			fmt.Printf("failed files: %v\n", objectMetas)
			ylogger.Zero.Error().Int("failed files count", len(objectMetas)).Msg("failed to upload some files")
			ylogger.Zero.Error().Any("failed files", objectMetas).Msg("failed to upload some files")

			_ = ycl.ReplyError(err, "failed to copy some files")
			return nil
		}

		if _, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode()); err != nil {
			_ = ycl.ReplyError(err, "failed to upload")
			return nil
		}
		fmt.Println("Copy finished successfully")
		ylogger.Zero.Info().Msg("Copy finished successfully")

	default:
		ylogger.Zero.Error().Any("type", tp).Msg("what type is it")
		_ = ycl.ReplyError(nil, "wrong request type")

		return nil
	}

	return nil
}
