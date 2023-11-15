package proc

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/yezzey-gp/yproxy/pkg/client"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type ProtoReader struct {
	c net.Conn
}

func NewProtoReader(ycl *client.YClient) *ProtoReader {
	return &ProtoReader{
		c: ycl.Conn,
	}
}

type MessageType byte

type RequestEncryption byte

const (
	MessageTypeCat   = MessageType(42)
	MessageTypePut   = MessageType(43)
	DecryptMessage   = RequestEncryption(1)
	NoDecryptMessage = RequestEncryption(0)
)

func (m MessageType) String() string {
	switch m {
	case MessageTypeCat:
		return "CAT"
	}
	return "UNKNOWN"
}

const maxMsgLen = 1 << 20

func (r *ProtoReader) ReadPacket() (MessageType, []byte, error) {
	msgLenBuf := make([]byte, 8)
	_, err := io.ReadFull(r.c, msgLenBuf)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read params: %w", err)
	}

	dataLen := binary.BigEndian.Uint64(msgLenBuf)

	if dataLen > maxMsgLen {
		return 0, nil, fmt.Errorf("message too big")
	}

	if dataLen <= 8 {
		return 0, nil, fmt.Errorf("message empty")
	}

	dataLen -= 8

	ylogger.Zero.Debug().Uint64("size", dataLen).Msg("requested packet")

	data := make([]byte, dataLen)
	_, err = io.ReadFull(r.c, data)
	if err != nil {
		return 0, nil, err
	}

	msgType := MessageType(data[0])
	return msgType, data, nil
}
