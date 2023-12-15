package proc

import (
	"bytes"
	"encoding/binary"
)

type PutMessage struct {
	ProtoMessage
	Encrypt bool
	Name    string
}

func NewPutMessage(name string, encrypt bool) *PutMessage {
	return &PutMessage{
		Name:    name,
		Encrypt: encrypt,
	}
}

func (c *PutMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeCat),
		0,
		0,
		0,
	}

	if c.Encrypt {
		bt[1] = byte(EncryptMessage)
	} else {
		bt[1] = byte(NoEncryptMessage)
	}

	bt = append(bt, []byte(c.Name)...)
	bt = append(bt, 0)
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *PutMessage) GetPutName(b []byte) string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String()
}

func (c *PutMessage) Decode(body []byte) error {
	if body[1] == byte(EncryptMessage) {
		c.Encrypt = true
	}
	c.Name = c.GetPutName(body[4:])
	return nil
}
