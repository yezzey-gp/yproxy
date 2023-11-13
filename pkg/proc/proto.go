package proc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type ProtoReader struct {
	c net.Conn
}

func NewProtoReader(c net.Conn) *ProtoReader {
	return &ProtoReader{c}
}

type MessageType byte

type RequestEncryption byte

const (
	MessageTypeCat   = MessageType(42)
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

func GetCatName(b []byte) string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String()
}

func ConstructMessage(name string, decrypt bool) []byte {

	bt := []byte{
		byte(MessageTypeCat),
		0,
		0,
		0,
	}

	if decrypt {
		bt[1] = byte(DecryptMessage)
	} else {
		bt[1] = byte(NoDecryptMessage)
	}

	bt = append(bt, []byte(name)...)
	bt = append(bt, 0)
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}
