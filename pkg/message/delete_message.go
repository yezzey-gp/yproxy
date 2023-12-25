package message

import (
	"bytes"
	"encoding/binary"
)

type DeleteMessage struct {
	Name string
}

var _ ProtoMessage = &DeleteMessage{}

func NewDeleteMessage(name string) *DeleteMessage {
	return &DeleteMessage{
		Name: name,
	}
}

func (c *DeleteMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeDelete),
		0,
		0,
		0,
	}

	bt = append(bt, []byte(c.Name)...)
	bt = append(bt, 0)
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *DeleteMessage) Decode(body []byte) {
	c.Name = c.GetDeleteName(body[4:])
}

func (c *DeleteMessage) GetDeleteName(b []byte) string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String()
}
