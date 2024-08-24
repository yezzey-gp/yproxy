package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yezzey-gp/yproxy/pkg/message"
	"github.com/yezzey-gp/yproxy/pkg/storage"
)

func TestResolveSettings(t *testing.T) {

	assert := assert.New(t)

	type tcase struct {
		name     string
		defaultV string
		exp      string
		settings []message.PutSetting
	}

	for _, tt := range []tcase{
		{
			"ababa",
			"aboba",
			"aboba",
			nil,
		},
		{
			"ababa",
			"aboba",
			"aboba",
			[]message.PutSetting{
				{
					Name:  "djewikdeowp",
					Value: "jdoiwejoidew",
				},
			},
		},

		{
			"ababa",
			"aboba",
			"valval",
			[]message.PutSetting{
				{
					Name:  "ababa",
					Value: "valval",
				},
			},
		},
	} {

		assert.Equal(tt.exp, storage.ResolveStorageSetting(tt.settings, tt.name, tt.defaultV))
	}
}
