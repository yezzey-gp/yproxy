package main

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {

		instanceCnf := config.InstanceConfig()

		con, err := net.Dial("tcp", instanceCnf.SocketPath)

		if err != nil {
			fmt.Errorf("encounter error: %w", err)
			os.Exit(1)
		}

		defer con.Close()

		_, err = con.Write([]byte(constructMessage(os.Args[1])))
		if err != nil {
			fmt.Errorf("encounter error: %w", err)
			os.Exit(1)
		}

		reply := make([]byte, 1024)

		_, err = con.Read(reply)

		if err != nil {
			fmt.Errorf("encounter error: %w", err)
			os.Exit(1)
		}

		fmt.Println("reply:", string(reply))
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		ylogger.Zero.Fatal().Err(err).Msg("")
	}
}
