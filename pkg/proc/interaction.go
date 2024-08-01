package proc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
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

		ycl.SetExternalFilePath(msg.Name)

		var w io.WriteCloser

		r, w := io.Pipe()

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {

			var ww io.WriteCloser = w
			if msg.Encrypt {
				if cr == nil {
					_ = ycl.ReplyError(err, "failed to encrypt, crypter not configured")
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
				ylogger.Zero.Debug().Str("path", msg.Name).Msg("omit encryption for chunk")
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

		var failed []*storage.ObjectInfo
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
			failed = make([]*storage.ObjectInfo, 0)
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

	case message.MessageTypeDelete:
		//recieve message
		msg := message.DeleteMessage{}
		msg.Decode(body)

		ycl.SetExternalFilePath(msg.Name)

		//get firsr backup lsn
		firstBackupLSN, err := getFirstLSN(msg.Segnum)
		if err != nil {
			ylogger.Zero.Error().AnErr("err", err).Msg("failed to get first lsn") //return or just assume there are no backups?
		}
		ylogger.Zero.Info().Uint64("lsn", firstBackupLSN).Msg("first backup LSN")

		//list files in storage
		ylogger.Zero.Info().Str("path", msg.Name).Msg("going to list path")
		objectMetas, err := s.ListPath(msg.Name)
		if err != nil {
			_ = ycl.ReplyError(fmt.Errorf("could not list objects: %s", err), "failed to complete request")
			return nil
		}
		ylogger.Zero.Info().Int("amount", len(objectMetas)).Msg("objects count")

		vi, ei, err := getVirtualExpireIndexes(msg.Port)
		if err != nil {
			ylogger.Zero.Error().AnErr("err", err).Msg("failed to get indexes")
			_ = ycl.ReplyError(fmt.Errorf("could not get virtual and expire indexes: %s", err), "failed to compelete request")
			return nil
		}
		ylogger.Zero.Info().Msg("recieved virtual index and expire index")
		ylogger.Zero.Debug().Int("virtual", len(vi)).Msg("vi count")
		ylogger.Zero.Debug().Int("expire", len(ei)).Msg("ei count")

		var failed []*storage.S3ObjectMeta
		retryCount := 0
		deletedFilesCount := 0
		for len(objectMetas) > 0 && retryCount < 10 {
			retryCount++
			for i := 0; i < len(objectMetas); i++ {
				p1 := strings.Split(objectMetas[i].Path, "/") //TODO fix this
				p2 := p1[len(p1)-1]
				p3 := strings.Split(p2, "_")
				if len(p3) >= 4 {
					p2 = fmt.Sprintf("%s_%s_%s_%s_", p3[0], p3[1], p3[2], p3[3])
				}
				lsn, ok := ei[p2]
				ylogger.Zero.Debug().Uint64("lsn", lsn).Uint64("backup lsn", firstBackupLSN).Msg("comparing lsn")
				if !vi[p2] && (lsn < firstBackupLSN || !ok) {
					ylogger.Zero.Debug().Str("file", objectMetas[i].Path).
						Bool("file in expire indexi", ok).
						Bool("lsn is less than in first backup", lsn < firstBackupLSN).
						Msg("file will be deleted")
					deletedFilesCount++
					if !msg.Confirm { //do not delete files if no confirmation flag provided
						continue
					}
					filePathParts := strings.Split(objectMetas[i].Path, "/")
					err = s.MoveObject(objectMetas[i].Path, fmt.Sprintf("segments_005/seg%d/basebackups_005/yezzey/trash/%s", msg.Segnum, filePathParts[len(filePathParts)-1]))
					if err != nil {
						deletedFilesCount--
						ylogger.Zero.Warn().AnErr("err", err).Str("file", objectMetas[i].Path).Msg("failed to move file")
						failed = append(failed, objectMetas[i])
					}
				}
			}
			objectMetas = failed
			failed = make([]*storage.S3ObjectMeta, 0)
		}

		ylogger.Zero.Info().Int("amount", deletedFilesCount).Msg("deleted files count")

		if len(objectMetas) > 0 {
			ylogger.Zero.Error().Int("failed files count", len(objectMetas)).Msg("some files were not moved")
			ylogger.Zero.Error().Any("failed files", objectMetas).Msg("failed to move some files")

			_ = ycl.ReplyError(err, "failed to move some files")
			return nil
		}

		if _, err = ycl.GetRW().Write(message.NewReadyForQueryMessage().Encode()); err != nil {
			_ = ycl.ReplyError(err, "failed to upload")
			return nil
		}
		ylogger.Zero.Info().Msg("Deleted garbage successfully")
		if !msg.Confirm {
			ylogger.Zero.Warn().Msg("It was a dry-run, nothing was deleted")
		}

	default:
		ylogger.Zero.Error().Any("type", tp).Msg("unknown message type")
		_ = ycl.ReplyError(nil, "wrong request type")

		return nil
	}

	return nil
}

// get lsn of the oldest backup
func getFirstLSN(seg int) (uint64, error) {
	cmd := exec.Command("/usr/bin/wal-g", "st", "ls", fmt.Sprintf("segments_005/seg%d/basebackups_005/", seg), "--config=/etc/wal-g/wal-g.yaml")
	ylogger.Zero.Debug().Any("flags", cmd.Args).Msg("Command args")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		ylogger.Zero.Debug().AnErr("error", err).Msg("Failed to run st ls")
		return 0, err
	}
	p1 := strings.Split(out.String(), "\n")

	minLSN := BackupLSN{Lsn: ^uint64(0)}
	for _, line := range p1 {
		if !strings.Contains(line, ".json") {
			continue
		}
		p2 := strings.Split(line, " ")
		p3 := p2[len(p2)-1]

		ylogger.Zero.Debug().Str("file: %s", fmt.Sprintf("segments_005/seg%d/basebackups_005/%s", seg, p3)).Msg("check lsn in file")
		cmd2 := exec.Command("/usr/bin/wal-g", "st", "cat", fmt.Sprintf("segments_005/seg%d/basebackups_005/%s", seg, p3), "--config=/etc/wal-g/wal-g.yaml")

		var out2 bytes.Buffer
		cmd2.Stdout = &out2

		err = cmd2.Run()
		if err != nil {
			ylogger.Zero.Debug().AnErr("error", err).Msg("Failed to run st cat")
			return 0, err
		}
		lsn := BackupLSN{}
		err = json.Unmarshal(out2.Bytes(), &lsn)

		if lsn.Lsn < minLSN.Lsn {
			minLSN.Lsn = lsn.Lsn
		}

	}

	return minLSN.Lsn, err
}

func connectToDatabase(port int, database string) (*pgx.Conn, error) {
	config, err := pgx.ParseEnvLibpq()
	if err != nil {
		return nil, errors.Wrap(err, "Connect: unable to read environment variables")
	}

	config.Port = uint16(port)
	config.Database = database

	config.RuntimeParams["gp_role"] = "utility"
	conn, err := pgx.Connect(config)
	if err != nil {
		config.RuntimeParams["gp_session_role"] = "utility"
		conn, err = pgx.Connect(config)
		if err != nil {
			fmt.Printf("error in connection %v", err) // delete this
			return nil, err
		}
	}
	return conn, nil
}

func getVirtualExpireIndexes(port int) (map[string]bool, map[string]uint64, error) { //TODO несколько баз
	db, err := getDatabase(port)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get ao/aocs tables %v", err) //fix
	}
	ylogger.Zero.Debug().Str("database name", db.name).Msg("recieved database")
	conn, err := connectToDatabase(port, db.name)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close() //error
	ylogger.Zero.Debug().Msg("connected to database")

	rows, err := conn.Query(`SELECT reloid, relfileoid, expire_lsn, fqnmd5 FROM yezzey.yezzey_expire_index WHERE expire_lsn != '0/0';`)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get ao/aocs tables %v", err) //fix
	}
	defer rows.Close()
	ylogger.Zero.Debug().Msg("executed select")

	c := make(map[string]uint64, 0)
	for rows.Next() {
		row := Ei{}
		if err := rows.Scan(&row.reloid, &row.relfileoid, &row.expireLsn, &row.fqnmd5); err != nil {
			return nil, nil, fmt.Errorf("unable to parse query output %v", err)
		}

		lsn, err := pgx.ParseLSN(row.expireLsn)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse query output %v", err)
		}

		ylogger.Zero.Debug().Str("file", fmt.Sprintf("%d_%d_%s_%d_", db.tablespace, db.oid, row.fqnmd5, row.relfileoid)).Msg("added file to ei")
		c[fmt.Sprintf("%d_%d_%s_%d_", db.tablespace, db.oid, row.fqnmd5, row.relfileoid)] = lsn
	}
	ylogger.Zero.Debug().Msg("read 1")

	rows2, err := conn.Query(`SELECT x_path FROM yezzey.yezzey_virtual_index;`)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get ao/aocs tables %v", err) //fix
	}
	defer rows2.Close()
	ylogger.Zero.Debug().Msg("read 2")

	c2 := make(map[string]bool, 0)
	for rows2.Next() {
		xpath := ""
		if err := rows2.Scan(&xpath); err != nil {
			return nil, nil, fmt.Errorf("unable to parse query output %v", err)
		}
		p1 := strings.Split(xpath, "/")
		p2 := p1[len(p1)-1]
		p3 := strings.Split(p2, "_")
		if len(p3) >= 4 {
			p2 = fmt.Sprintf("%s_%s_%s_%s_", p3[0], p3[1], p3[2], p3[3])
		}
		c2[p2] = true
		ylogger.Zero.Debug().Str("file", p2).Msg("added")
	}
	ylogger.Zero.Debug().Msg("read 3")

	return c2, c, err
}

func getDatabase(port int) (DB, error) {
	conn, err := connectToDatabase(port, "postgres")
	if err != nil {
		return DB{}, err
	}
	defer conn.Close() //error
	ylogger.Zero.Debug().Msg("connected to db")
	rows, err := conn.Query(`SELECT dattablespace, oid, datname FROM pg_database WHERE datallowconn;`)
	if err != nil {
		return DB{}, err
	}
	defer rows.Close()
	ylogger.Zero.Debug().Msg("recieved db list")

	for rows.Next() {
		row := DB{}
		ylogger.Zero.Debug().Msg("cycle 1")
		if err := rows.Scan(&row.tablespace, &row.oid, &row.name); err != nil {
			return DB{}, err
		}
		ylogger.Zero.Debug().Msg("cycle 2")
		ylogger.Zero.Debug().Str("db", row.name).Int("db", int(row.oid)).Int("db", int(row.tablespace)).Msg("database")
		if row.name == "postgres" {
			continue
		}

		ylogger.Zero.Debug().Str("db", row.name).Msg("check database")
		connDb, err := connectToDatabase(port, row.name)
		if err != nil {
			return DB{}, err
		}
		defer connDb.Close() //error
		ylogger.Zero.Debug().Msg("cycle 3")

		rowsdb, err := connDb.Query(`SELECT exists(SELECT * FROM information_schema.schemata WHERE schema_name='yezzey');`)
		if err != nil {
			return DB{}, err
		}
		defer rowsdb.Close()
		ylogger.Zero.Debug().Msg("cycle 4")
		var ans bool
		rowsdb.Next()
		err = rowsdb.Scan(&ans)
		if err != nil {
			ylogger.Zero.Error().AnErr("error", err).Msg("error during yezzey check")
			return DB{}, err
		}
		ylogger.Zero.Debug().Bool("result", ans).Msg("find yezzey schema")
		if ans {
			ylogger.Zero.Debug().Str("db", row.name).Msg("found yezzey schema in database")
			ylogger.Zero.Debug().Int("db", int(row.oid)).Int("db", int(row.tablespace)).Msg("found yezzey schema in database")
			return row, nil
		}

		ylogger.Zero.Debug().Str("db", row.name).Msg("no yezzey schema in database")
	}
	return DB{}, fmt.Errorf("no yezzey schema across databases")
}

type Ei struct {
	reloid     pgtype.OID
	relfileoid pgtype.OID
	expireLsn  string
	fqnmd5     string
}

type DB struct {
	name       string
	tablespace pgtype.OID
	oid        pgtype.OID
}

type BackupLSN struct {
	Lsn uint64 `json:"FinishLSN"`
}
