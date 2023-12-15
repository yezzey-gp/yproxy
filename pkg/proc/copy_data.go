package proc

import (
	"encoding/binary"
)

type CopyDataMessage struct {
	ProtoMessage
	Sz   uint64
	Data []byte
}

func NewCopyDataMessage() *CopyDataMessage {
	return &CopyDataMessage{}
}

func (cc *CopyDataMessage) Encode() []byte {
	bt := make([]byte, 4+8+cc.Sz)

	bt[0] = byte(MessageTypeCopyData)

	// sizeof(sz) + data
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))

	binary.BigEndian.PutUint64(bt[4:], uint64(cc.Sz))

	// check data len more than cc.sz?
	copy(bt[(4+8):], cc.Data[:cc.Sz])

	return append(bs, bt...)
}

func (cc *CopyDataMessage) Decode(data []byte) {
	msgLenBuf := data[4:12]
	cc.Sz = binary.BigEndian.Uint64(msgLenBuf)
	cc.Data = make([]byte, cc.Sz)
	copy(cc.Data, data[12:12+cc.Sz])
}
