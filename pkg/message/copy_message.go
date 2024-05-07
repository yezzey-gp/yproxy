package message

import (
	"encoding/binary"
	"fmt"

	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

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

func (message *CopyMessage) Encode() []byte {
	encodedMessage := []byte{
		byte(MessageTypeCopy),
		byte(NoDecryptMessage),
		byte(NoEncryptMessage),
		0,
	}

	if message.Decrypt {
		encodedMessage[1] = byte(DecryptMessage)
	}

	if message.Encrypt {
		encodedMessage[2] = byte(EncryptMessage)
	}

	byteName := []byte(message.Name)
	byteLen := make([]byte, 8)
	binary.BigEndian.PutUint64(byteLen, uint64(len(byteName)))
	encodedMessage = append(encodedMessage, byteLen...)
	encodedMessage = append(encodedMessage, byteName...)

	byteOldCfg := []byte(message.OldCfgPath)
	binary.BigEndian.PutUint64(byteLen, uint64(len(byteOldCfg)))
	encodedMessage = append(encodedMessage, byteLen...)
	encodedMessage = append(encodedMessage, byteOldCfg...)

	binary.BigEndian.PutUint64(byteLen, uint64(len(encodedMessage)+8))
	fmt.Printf("send: %v\n", MessageType(encodedMessage[0]))
	ylogger.Zero.Debug().Str("object-path", MessageType(encodedMessage[0]).String()).Msg("decrypt object")
	return append(byteLen, encodedMessage...)
}

func (encodedMessage *CopyMessage) Decode(data []byte) {
	if data[1] == byte(DecryptMessage) {
		encodedMessage.Decrypt = true
	}
	if data[2] == byte(EncryptMessage) {
		encodedMessage.Encrypt = true
	}

	nameLen := binary.BigEndian.Uint64(data[4:12])
	encodedMessage.Name = string(data[12 : 12+nameLen])
	oldConfLen := binary.BigEndian.Uint64(data[12+nameLen : 12+nameLen+8])
	encodedMessage.OldCfgPath = string(data[12+nameLen+8 : 12+nameLen+8+oldConfLen])
}
