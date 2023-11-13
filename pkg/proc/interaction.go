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
		name := GetCatName(body[4:])
		ylogger.Zero.Debug().Str("object-path", name).Msg("cat object")
		r, err := s.CatFileFromStorage(name)
		if err != nil {
			_ = ycl.ReplyError(err, "failed to compelete request")

			return err
		}
		if body[1] == byte(DecryptMessage) {
			ylogger.Zero.Debug().Str("object-path", name).Msg("decrypt object ")
			r, err = cr.Decrypt(r)
			if err != nil {
				_ = ycl.ReplyError(err, "failed to compelete request")

				return err
			}
		}
		io.Copy(ycl.Conn, r)

	default:

		_ = ycl.ReplyError(nil, "wrong request type")

		return ycl.Conn.Close()
	}

	return nil
}
