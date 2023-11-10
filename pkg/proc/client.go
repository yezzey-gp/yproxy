package proc

import (
	"io"
	"net"

	"github.com/yezzey-gp/yproxy/pkg/storage"
)

func ProcConn(s storage.StorageReader, c net.Conn) error {
	pr := NewProtoReader(c)
	tp, body, err := pr.ReadPacket()
	if err != nil {
		return err
	}
	switch tp {
	case MessageTypeCat:
		name := GetCatName(body)
		r, err := s.CatFileFromStorage(name)
		if err != nil {
			return err
		}
		io.Copy(c, r)
	}

	return nil
}
