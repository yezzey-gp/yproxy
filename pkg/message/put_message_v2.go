package message

import (
	"bytes"
	"encoding/binary"
)

type PutSetting struct {
	Name  string
	Value string
}

type PutMessageV2 struct {
	Encrypt bool
	Name    string

	Settings []PutSetting
}

var _ ProtoMessage = &PutMessageV2{}

func NewPutMessageV2(name string, encrypt bool, settings []PutSetting) *PutMessageV2 {
	return &PutMessageV2{
		Name:     name,
		Encrypt:  encrypt,
		Settings: settings,
	}
}

func (c *PutMessageV2) Encode() []byte {
	bt := []byte{
		byte(MessageTypePutV2),
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

	slen := make([]byte, 8)
	binary.BigEndian.PutUint64(slen, uint64(len(c.Settings)))
	bt = append(bt, slen...)

	for _, s := range c.Settings {

		bt = append(bt, []byte(s.Name)...)
		bt = append(bt, 0)

		bt = append(bt, []byte(s.Value)...)
		bt = append(bt, 0)
	}

	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *PutMessageV2) GetCstring(b []byte) (string, uint64) {
	offset := uint64(0)
	buff := bytes.NewBufferString("")

	for i := 0; i < len(b); i++ {
		offset++
		if b[i] == 0 {
			break
		}
		buff.WriteByte(b[i])
	}

	return buff.String(), offset
}

func (c *PutMessageV2) Decode(body []byte) {
	if body[1] == byte(EncryptMessage) {
		c.Encrypt = true
	}
	var off uint64
	c.Name, off = c.GetCstring(body[4:])

	settLen := binary.BigEndian.Uint64(body[4+off : 4+off+8])

	totalOff := 4 + off + 8

	c.Settings = make([]PutSetting, settLen)

	for i := 0; i < int(settLen); i++ {

		var currOff uint64

		c.Settings[i].Name, currOff = c.GetCstring(body[totalOff:])
		totalOff += currOff

		c.Settings[i].Value, currOff = c.GetCstring(body[totalOff:])
		totalOff += currOff
	}
}
