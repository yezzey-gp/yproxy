package core

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/crypt"
	"github.com/yezzey-gp/yproxy/pkg/proc"
	"github.com/yezzey-gp/yproxy/pkg/storage"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type Instance struct {
	crypter crypt.Crypter
}

func (i *Instance) Run(instanceCnf *config.Instance) error {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	go func() {
		defer os.Remove(instanceCnf.SocketPath)
		defer cancelCtx()

		for {
			s := <-sigs
			ylogger.Zero.Info().Str("signal", s.String()).Msg("received signal")

			switch s {
			case syscall.SIGUSR1:
				ylogger.ReloadLogger(instanceCnf.LogPath)
			case syscall.SIGUSR2:
				return
			case syscall.SIGHUP:
				// reread config file

			case syscall.SIGINT, syscall.SIGTERM:

				// make better
				return
			default:
				return
			}
		}
	}()

	listener, err := net.Listen("unix", instanceCnf.SocketPath)
	if err != nil {
		ylogger.Zero.Error().Err(err).Msg("failed to start socket listener")
		return err
	}
	defer listener.Close()

	s := storage.NewStorage(
		&instanceCnf.StorageCnf,
	)

	cr := crypt.NewCrypto(&instanceCnf.CryptoCnf)

	go func() {
		<-ctx.Done()
		os.Exit(0)
	}()

	for {
		clConn, err := listener.Accept()
		if err != nil {
			ylogger.Zero.Error().Err(err).Msg("failed to accept connection")
		}
		ylogger.Zero.Debug().Str("addr", clConn.LocalAddr().String()).Msg("accepted client connection")
		go proc.ProcConn(s, cr, clConn)
	}
}
