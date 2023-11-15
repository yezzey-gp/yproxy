package proc

import "encoding/binary"

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
