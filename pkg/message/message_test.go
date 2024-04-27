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

		msg2 := message.PutMessage{}

		msg2.Decode(body[8:])

		assert.Equal(msg.Name, msg2.Name)
		assert.Equal(msg.Encrypt, msg2.Encrypt)
	}
}

func TestPatchMsg(t *testing.T) {
	assert := assert.New(t)

	type tcase struct {
		name    string
		encrypt bool
		off     uint64
		err     error
	}

	for _, tt := range []tcase{
		{
			"nam1",
			true,
			1235,
			nil,
		},
	} {

		msg := message.NewPatchMessage(tt.name, tt.off, tt.encrypt)
		body := msg.Encode()

		msg2 := message.PatchMessage{}

		msg2.Decode(body[8:])

		assert.Equal(msg.Name, msg2.Name)
		assert.Equal(msg.Encrypt, msg2.Encrypt)
		assert.Equal(msg.Offset, msg2.Offset)
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

func TestCopyMsg(t *testing.T) {
	assert := assert.New(t)

	msg := message.NewCopyMessage("myname/mynextname", "myoldcfg/path", true, true)
	body := msg.Encode()

	assert.Equal(body[8], byte(message.MessageTypeCopy))

	msg2 := message.CopyMessage{}
	msg2.Decode(body[8:])

	assert.Equal("myname/mynextname", msg2.Name)
	assert.Equal("myoldcfg/path", msg2.OldCfgPath)
	assert.True(msg2.Decrypt)
	assert.True(msg2.Encrypt)
}
