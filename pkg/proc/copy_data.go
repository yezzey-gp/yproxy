package proc

import "encoding/binary"

type CopyDataMessage struct {
	Sz   int
	Data []byte
}

func NewCopyDataMessage() *CopyDataMessage {
	return &CopyDataMessage{}
}

func (cc *CopyDataMessage) Encode() []byte {
	bt := make([]byte, 4+cc.Sz)

	bt[0] = byte(MessageTypeCopyData)

	// sizeof(sz) + data
	ln := len(bt) + 8 + 8 + cc.Sz

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))

	binary.BigEndian.PutUint64(bt[4:], uint64(cc.Sz))

	// check data len more than cc.sz?
	copy(bt[4+8:], cc.Data[:cc.Sz])

	return append(bs, bt...)
}
