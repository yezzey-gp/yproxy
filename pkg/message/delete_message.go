package message

import (
	"bytes"
	"encoding/binary"
)

type DeleteMessage struct { //seg port
	Name    string
	Port    int
	Segnum  int
	Confirm bool
}

var _ ProtoMessage = &DeleteMessage{}

func NewDeleteMessage(name string, port int, seg int, confirm bool) *DeleteMessage {
	return &DeleteMessage{
		Name:    name,
		Port:    port,
		Segnum:  seg,
		Confirm: confirm,
	}
}

func (c *DeleteMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeDelete),
		0,
		0,
		0,
	}

	if c.Confirm {
		bt[1] = 1
	}

	bt = append(bt, []byte(c.Name)...)
	bt = append(bt, 0)

	p := make([]byte, 8)
	binary.BigEndian.PutUint64(p, uint64(c.Port))
	bt = append(bt, p...)

	p = make([]byte, 8)
	binary.BigEndian.PutUint64(p, uint64(c.Segnum))
	bt = append(bt, p...)

	ln := len(bt) + 8
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *DeleteMessage) Decode(body []byte) {
	if body[1] == 1 {
		c.Confirm = true
	}
	c.Name = c.GetDeleteName(body[4:])
	c.Port = int(binary.BigEndian.Uint64(body[len(body)-16 : len(body)-8]))
	c.Segnum = int(binary.BigEndian.Uint64(body[len(body)-8:]))
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
