package proc

import (
	"io"
	"net"
)

func ProcConn(c net.Conn) {
	pr := NewProtoReader(c)
	tp, body, err := pr.ReadPacket()
	if err != nil {
		return
	}
	switch tp {
	case MessageTypeCat:
		name := GetCatName(body)
		io.Copy(c, CatFileFromStorage(name))
	}
}
