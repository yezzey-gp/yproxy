package client

import (
	"fmt"
	"net"

	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type YClient struct {
	Conn net.Conn
}

func NewYClient(c net.Conn) *YClient {
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
