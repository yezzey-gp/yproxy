package main

import (
	"net"

	"github.com/spf13/cobra"
	"github.com/yezzey-gp/yproxy/pkg/proc"
	"github.com/yezzey-gp/yproxy/pkg/storage"

	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

var cfgPath string

var logLevel string

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := config.LoadInstanceConfig(cfgPath)
		if err != nil {
			return err
		}

		instanceCnf := config.InstanceConfig()

		logger := ylogger.NewZeroLogger(instanceCnf.LogPath)

		listener, err := net.Listen("unix", instanceCnf.SocketPath)
		if err != nil {
			logger.Error().Err(err).Msg("failed to start socket listener")
			return err
		}
		defer listener.Close()

		s := storage.NewStorage()

		for {
			clConn, err := listener.Accept()
			if err != nil {
				logger.Error().Err(err).Msg("failed to accept connection")
			}
			go proc.ProcConn(s, clConn)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "/etc/yproxy/yproxy.yaml", "path to yproxy config file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		ylogger.Zero.Fatal().Err(err).Msg("")
	}
}
