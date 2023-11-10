package main

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := config.LoadInstanceCfg(cfgPath)
		if err != nil {
			return err
		}

		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("tcp", instanceCnf.SocketPath)

		if err != nil {
			return err
		}

		defer con.Close()

		_, err = con.Write(ConstructMessage(Args[1]))
		if err != nil {
			return err
		}

		reply := make([]byte, 1024)

		_, err = con.Read(reply)

		if err != nil {
			return err
		}

		fmt.Println("reply:", string(reply))
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
