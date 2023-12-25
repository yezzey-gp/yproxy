package message

import "encoding/binary"

type ReadyForQueryMessage struct {
	ProtoMessage
}

func NewReadyForQueryMessage() *ReadyForQueryMessage {
	return &ReadyForQueryMessage{}
}

func (cc *ReadyForQueryMessage) Encode() []byte {
	bt := []byte{
		byte(MessageTypeReadyForQuery),
		0,
		0,
		0,
	}

	ln := len(bt) + 8

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(ln))
	return append(bs, bt...)
}

func (c *ReadyForQueryMessage) Decode(body []byte) error {
	return nil
}
