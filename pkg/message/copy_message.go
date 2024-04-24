package message

import "encoding/binary"

type CopyMessage struct {
	Decrypt    bool
	Encrypt    bool
	Name       string
	OldCfgPath string
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
	bt := []byte{
		byte(MessageTypeCat),
		byte(NoDecryptMessage),
		byte(NoEncryptMessage),
		0,
	}

	if cc.Decrypt {
		bt[1] = byte(DecryptMessage)
	}

	if cc.Encrypt {
		bt[2] = byte(EncryptMessage)
	}

	byteName := []byte(cc.Name)
	binary.BigEndian.PutUint64(bt, uint64(len(byteName)))
	bt = append(bt, byteName...)

	byteOldCfg := []byte(cc.OldCfgPath)
	binary.BigEndian.PutUint64(bt, uint64(len(byteOldCfg)))
	bt = append(bt, byteOldCfg...)

	ln := len(bt) + 8
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (cc *CopyMessage) Decode(data []byte) {
	if data[1] == byte(DecryptMessage) {
		cc.Decrypt = true
	}
	if data[2] == byte(EncryptMessage) {
		cc.Encrypt = true
	}

	nameLen := binary.BigEndian.Uint64(data[4:12])
	cc.Name = string(data[12 : 12+nameLen])
	oldConfLen := binary.BigEndian.Uint64(data[12+nameLen : 12+nameLen+8])
	cc.OldCfgPath = string(data[12+nameLen+8 : 12+nameLen+8+oldConfLen])
}
