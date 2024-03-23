package proc_test

import (
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock "github.com/yezzey-gp/yproxy/pkg/mock/proc"
	"github.com/yezzey-gp/yproxy/pkg/proc"
)

func TestYproxyRetryReaderEmpty(t *testing.T) {

	ctrl := gomock.NewController(t)

	rr := mock.NewMockRestartReader(ctrl)

	yr := proc.NewYRetryReader(rr)

	buf := []byte{1, 233, 45}

	rr.EXPECT().Restart(int64(0)).Return(nil)
	rr.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
	rr.EXPECT().Close().Times(1)

	_, err := yr.Read(buf)

	assert.Equal(t, io.EOF, err)

	assert.Nil(t, yr.Close())
}
