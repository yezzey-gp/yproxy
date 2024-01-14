package message

import (
	"bytes"
	"encoding/binary"
)

type PatchMessage struct {
	Encrypt bool
	Offset  uint64
	Name    string
}

var _ ProtoMessage = &PatchMessage{}

func NewPatchMessage(name string, offset uint64, encrypt bool) *PatchMessage {
	return &PatchMessage{
		Encrypt: encrypt,
		Offset:  offset,
		Name:    name,
	}
}

func (c *PatchMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypePatch),
		0,
		0,
		0,
	}

	if c.Encrypt {
		bt[1] = byte(EncryptMessage)
	} else {
		bt[1] = byte(NoEncryptMessage)
	}
	offset := make([]byte, 8)
	binary.BigEndian.PutUint64(offset, uint64(c.Offset))

	bt = append(bt, offset...)
	bt = append(bt, []byte(c.Name)...)
	bt = append(bt, 0)
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *PatchMessage) GetPatchName(b []byte) string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String()
}

func (c *PatchMessage) Decode(body []byte) {
	if body[1] == byte(EncryptMessage) {
		c.Encrypt = true
	}

	c.Offset = binary.BigEndian.Uint64(body[4:12])
	c.Name = c.GetPatchName(body[12:])
}
