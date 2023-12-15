package proc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yezzey-gp/yproxy/pkg/proc"
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

		msg := proc.NewCatMessage(tt.name, tt.decrypt)
		body := msg.Encode()

		msg2 := proc.CatMessage{}

		err := msg2.Decode(body[8:])

		assert.NoError(err)
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

		msg := proc.NewPutMessage(tt.name, tt.encrypt)
		body := msg.Encode()

		msg2 := proc.CatMessage{}

		err := msg2.Decode(body[8:])

		assert.NoError(err)
		assert.Equal(msg.Name, msg2.Name)
		assert.Equal(msg.Encrypt, msg2.Decrypt)
	}
}
