package message_test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yezzey-gp/yproxy/pkg/message"
)

func TestCatMsg(t *testing.T) {
	assert := assert.New(t)

	type tcase struct {
		name    string
		decrypt bool
		err     error
	}

	for _, tt := range []tcase{
		{
			"nam1",
			true,
			nil,
		},
	} {

		msg := message.NewCatMessage(tt.name, tt.decrypt)
		body := msg.Encode()

		msg2 := message.CatMessage{}

		msg2.Decode(body[8:])

		assert.Equal(msg.Name, msg2.Name)
		assert.Equal(msg.Decrypt, msg2.Decrypt)
	}
}

func TestPutMsg(t *testing.T) {
	assert := assert.New(t)

	type tcase struct {
		name    string
		encrypt bool
		err     error
	}

	for _, tt := range []tcase{
		{
			"nam1",
			true,
			nil,
		},
	} {

		msg := message.NewPutMessage(tt.name, tt.encrypt)
		body := msg.Encode()

		msg2 := message.CatMessage{}

		msg2.Decode(body[8:])

		assert.Equal(msg.Name, msg2.Name)
		assert.Equal(msg.Encrypt, msg2.Decrypt)
	}
}

func TestCopyDataMsg(t *testing.T) {
	assert := assert.New(t)

	type tcase struct {
		body []byte
		err  error
	}

	for _, tt := range []tcase{
		{
			[]byte(
				"hiuefheiufheuif",
			),
			nil,
		},
	} {

		msg := message.NewCopyDataMessage()
		msg.Data = tt.body
		msg.Sz = uint64(len(tt.body))
		body := msg.Encode()

		msg2 := message.CopyDataMessage{}

		msg2.Decode(body[8:])

		sz := binary.BigEndian.Uint64(body[:8])

		assert.Equal(int(sz), len(body))

		assert.Equal(msg.Data, msg2.Data)
		assert.Equal(msg.Sz, msg2.Sz)
	}
}

func TestListMsg(t *testing.T) {
	assert := assert.New(t)

	type tcase struct {
		prefix string
	}

	for _, tt := range []tcase{
		{
			"nam1",
		},
	} {

		msg := message.NewListMessage(tt.prefix)
		body := msg.Encode()

		msg2 := message.ListMessage{}

		msg2.Decode(body[8:])

		assert.Equal(msg.Prefix, msg2.Prefix)
	}
}
