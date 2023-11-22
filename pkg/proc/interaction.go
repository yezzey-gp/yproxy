package proc

import (
	"io"

	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/crypt"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

func ProcConn(s storage.StorageReader, cr crypt.Crypter, ycl *client.YClient) error {
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

	default:

		_ = ycl.ReplyError(nil, "wrong request type")

		return ycl.Conn.Close()
	}

	return nil
}
