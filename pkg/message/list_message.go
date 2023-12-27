package message

import (
	"bytes"
	"encoding/binary"
)

type ListMessage struct {
	Name string
}

var _ ProtoMessage = &ListMessage{}

func NewListMessage(name string) *ListMessage {
	return &ListMessage{
		Name: name,
	}
}

func (c *ListMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeList),
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

func (c *ListMessage) Decode(body []byte) {
	c.Name = c.GetListName(body[4:])
}

func (c *ListMessage) GetListName(b []byte) string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String()
}
