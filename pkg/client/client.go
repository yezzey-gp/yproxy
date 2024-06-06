package client

import (
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type YproxyClient interface {
	ID() uint
	ReplyError(err error, msg string) error
	GetRW() io.ReadWriteCloser
	Close() error
}

type YClient struct {
	Conn net.Conn
}

// Close implements YproxyClient.
func (y *YClient) Close() error {
	return y.Conn.Close()
}

// GetPointer do the same thing like fmt.Sprintf("%p", &num) but fast
// GetPointer returns the memory address of the given value as an unsigned integer.
func GetPointer(value interface{}) uint {
	ptr := reflect.ValueOf(value).Pointer()
	uintPtr := uintptr(ptr)
	return uint(uintPtr)
}

// ID implements YproxyClient.
func (y *YClient) ID() uint {
	return GetPointer(y)
}

func NewYClient(c net.Conn) YproxyClient {
	return &YClient{
		Conn: c,
	}
}

func (y *YClient) ReplyError(err error, msg string) error {
	ylogger.Zero.Error().Err(err).Msg(msg)

	_, _ = y.Conn.Write([]byte(
		fmt.Sprintf("%s: %v", msg, err),
	))
	return nil
}

func (y *YClient) GetRW() io.ReadWriteCloser {
	return y.Conn
}
