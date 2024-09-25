package proc

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/backups"
	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/crypt"
	"github.com/yezzey-gp/yproxy/pkg/database"
	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/object"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

func ProcessPutExtended(
	s storage.StorageInteractor,
	pr *ProtoReader,
	name string,
	encrypt bool, settings []message.PutSettings, cr crypt.Crypter, ycl client.YproxyClient) error {

	ycl.SetExternalFilePath(name)

	var w io.WriteCloser
	r, w := io.Pipe()

	defer r.Close()
	defer w.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		var ww io.WriteCloser = w
		if encrypt {
			if cr == nil {
				_ = ycl.ReplyError(fmt.Errorf("failed to encrypt, crypter not configured"), "connection aborted")
				ycl.Close()
				return
			}

			var err error
			ww, err = cr.Encrypt(w)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to encrypt")

				ycl.Close()
				return
			}
		} else {
			ylogger.Zero.Debug().Str("path", name).Msg("omit encryption for upload chunks")
		}

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

	/* Should go after reader dispatch! */
	err := s.PutFileToDest(name, r, settings)

	if err != nil {
		_ = ycl.ReplyError(err, "failed to upload")

		return err
	}

	wg.Wait()

	_, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode())

	if err != nil {
		_ = ycl.ReplyError(err, "failed to upload")

		return err
	}

	return nil
}

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
			if cr == nil {
				_ = ycl.ReplyError(err, "failed to decrypt object, decrypter not configured")
				ycl.Close()
				return nil
			}
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

		if err := ProcessPutExtended(s, pr, msg.Name, msg.Encrypt, nil, cr, ycl); err != nil {
			return err
		}

	case message.MessageTypePutV2:

		msg := message.PutMessageV2{}
		msg.Decode(body)

		if err := ProcessPutExtended(s, pr, msg.Name, msg.Encrypt, msg.Settings, cr, ycl); err != nil {
			return err
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

			return err
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
		oldStorage, err := storage.NewStorage(&instanceCnf.StorageCnf)
		if err != nil {
			return err
		}
		fmt.Printf("ok new conf: %v\n", instanceCnf)

		//list objects
		objectMetas, err := oldStorage.ListPath(msg.Name)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not list objects: %s", err), "failed to compelete request")
			return nil
		}

		var failed []*object.ObjectInfo
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
						ylogger.Zero.Error().Str("path", path).Err(err).Msg("failed to copy data")
						failed = append(failed, objectMetas[i])
						return
					}

					if err := writerToNewBucket.Close(); err != nil {
						ylogger.Zero.Error().Str("path", path).Err(err).Msg("failed to close writer")
						failed = append(failed, objectMetas[i])
						return
					}
				}()

				//write file
				err = s.PutFileToDest(path, readerEncrypt, nil)
				if err != nil {
					ylogger.Zero.Error().Err(err).Msg("failed to upload file")
					failed = append(failed, objectMetas[i])
					continue
				}
			}
			objectMetas = failed
			fmt.Printf("failed files count: %d\n", len(objectMetas))
			failed = make([]*object.ObjectInfo, 0)
		}

		if len(objectMetas) > 0 {
			fmt.Printf("failed files count: %d\n", len(objectMetas))
			fmt.Printf("failed files: %v\n", objectMetas)
			ylogger.Zero.Error().Int("failed files count", len(objectMetas)).Msg("failed to upload some files")
			ylogger.Zero.Error().Any("failed files", objectMetas).Msg("failed to upload some files")

			_ = ycl.ReplyError(err, "failed to copy some files")
			return err
		}

		if _, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode()); err != nil {
			_ = ycl.ReplyError(err, "failed to upload")
			return err
		}
		fmt.Println("Copy finished successfully")
		ylogger.Zero.Info().Msg("Copy finished successfully")

	case message.MessageTypeDelete:
		//recieve message
		msg := message.DeleteMessage{}
		msg.Decode(body)

		ycl.SetExternalFilePath(msg.Name)

		dbInterractor := &database.DatabaseHandler{}
		backupHandler := &backups.WalgBackupInterractor{}

		var dh DeleteHandler
		dh = &BasicDeleteHandler{
			StorageInterractor: s,
			DbInterractor:      dbInterractor,
			BackupInterractor:  backupHandler,
		}

		if msg.Garbage {
			err = dh.HandleDeleteGarbage(msg)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to finish operation")
				return err
			}
		} else {
			err = dh.HandleDeleteFile(msg)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to finish operation")
				return err
			}
		}

		if _, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode()); err != nil {
			_ = ycl.ReplyError(err, "failed to upload")
			return err
		}
		ylogger.Zero.Info().Msg("Deleted garbage successfully")
		if !msg.Confirm {
			ylogger.Zero.Warn().Msg("It was a dry-run, nothing was deleted")
		}
	case message.MessageTypeGool:
		return ProcMotion(s, cr, ycl)

	default:
		ylogger.Zero.Error().Any("type", tp).Msg("unknown message type")
		_ = ycl.ReplyError(nil, "wrong request type")

		return nil
	}

	return nil
}

func ProcMotion(s storage.StorageInteractor, cr crypt.Crypter, ycl client.YproxyClient) error {

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

	msg := message.GoolMessage{}
	msg.Decode(body)

	ylogger.Zero.Info().Msg("recieved client gool succ")

	_, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode())
	if err != nil {
		_ = ycl.ReplyError(err, "failed to gool")
	}
	return nil
}
