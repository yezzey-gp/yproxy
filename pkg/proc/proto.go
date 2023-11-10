package proc

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type ProtoReader struct {
	c net.Conn
}

func NewProtoReader(c net.Conn) *ProtoReader {
	return &ProtoReader{c}
}

type MessageType int

const maxMsgLen = 1 << 20

func (r *ProtoReader) ReadPacket() (MessageType, []byte, error) {
	msgLenBuf := make([]byte, 4)
	_, err := io.ReadFull(r.c, msgLenBuf)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read params: %w", err)
	}

	dataLen := binary.BigEndian.Uint64(msgLenBuf)

	if dataLen > maxMsgLen {
		return 0, nil, fmt.Errorf("message too big")
	}

	data := make([]byte, dataLen)
	_, err = io.ReadFull(r.c, data)
	if err != nil {
		return 0, nil, err
	}

	msgType := MessageType(data[0])
	return msgType, data, nil
}

func GetCatName(b []byte) string {

	return ""
}
