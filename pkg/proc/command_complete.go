package proc

import "encoding/binary"

type CommandCompleteMessage struct {
}

var _ ProtoMessage = &CommandCompleteMessage{}

func NewCommandCompleteMessage() *CommandCompleteMessage {
	return &CommandCompleteMessage{}
}

func (cc *CommandCompleteMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeCommandComplete),
		0,
		0,
		0,
	}

	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *CommandCompleteMessage) Decode(body []byte) {
}
