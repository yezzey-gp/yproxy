package main

import (
	"net"

	"github.com/yezzey-gp/yproxy/pkg/proc"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

var port int = 1337

var sockPath string = "/tmp/yezzey.sock"

func main() {

	logger := ylogger.NewZeroLogger("proxy.log")

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		logger.Error().Err(err).Msg("failed to start socket listener")
		return
	}
	defer listener.Close()

	for {
		clConn, err := listener.Accept()
		if err != nil {
			logger.Error().Err(err).Msg("failed to accept connection")
		}
		go proc.ProcConn(clConn)
	}
}
