package message

import "encoding/binary"

type CopyMessage struct {
	Decrypt    bool
	Encrypt    bool
	Name       string
	OldCfgPath string
	Sz         uint64
	Data       []byte
}

var _ ProtoMessage = &CopyMessage{}

func NewCopyMessage(name, oldCfgPath string, encrypt, decrypt bool) *CopyMessage {
	return &CopyMessage{
		Name:       name,
		Encrypt:    encrypt,
		Decrypt:    decrypt,
		OldCfgPath: oldCfgPath,
	}
}

func (cc *CopyMessage) Encode() []byte {
	bt := make([]byte, 4+8+cc.Sz)

	bt[0] = byte(MessageTypeCopy)

	// sizeof(sz) + data
	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))

	binary.BigEndian.PutUint64(bt[4:], uint64(cc.Sz))

	// check data len more than cc.sz?
	copy(bt[(4+8):], cc.Data[:cc.Sz])

	return append(bs, bt...)
}

func (cc *CopyMessage) Decode(data []byte) {
	msgLenBuf := data[4:12]
	cc.Sz = binary.BigEndian.Uint64(msgLenBuf)
	cc.Data = make([]byte, cc.Sz)
	copy(cc.Data, data[12:12+cc.Sz])
}
