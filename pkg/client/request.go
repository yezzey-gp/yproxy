package client

import "net"

type YClient struct {
	conn net.Conn
}

func ReplyError(err error) error {
	return nil
}
