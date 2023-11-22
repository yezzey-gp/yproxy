package proc

import (
	"bytes"
	"encoding/binary"
)

type CatMessage struct {
	ProtoMessage
	Decrypt bool
	Name    string
}

func NewCatMessage(name string, decrypt bool) *CatMessage {
	return &CatMessage{
		Name:    name,
		Decrypt: decrypt,
	}
}

func (c *CatMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeCat),
		0,
		0,
		0,
	}

	if c.Decrypt {
		bt[1] = byte(DecryptMessage)
	} else {
		bt[1] = byte(NoDecryptMessage)
	}

	bt = append(bt, []byte(c.Name)...)
	bt = append(bt, 0)
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *CatMessage) Decode(body []byte) error {
	c.Name = c.GetCatName(body[4:])
	if body[1] == byte(DecryptMessage) {
		c.Decrypt = true
	}

	return nil
}

func (c *CatMessage) GetCatName(b []byte) string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String()
}
