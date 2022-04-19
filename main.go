package main

import (
	"net/http"
	_ "net/http/pprof"
	"strconv"

	"fmt"

	"github.com/oraoto/go-bore/bore"
	"github.com/spf13/cobra"
)


var (
	serverSecret string
	serverMinPort int
	remotePort int
	remoteHost string
	localPort int
	localSecret string
)

func main() {
	rootCmd := &cobra.Command{
		Use: "go-bore",
		Short: "a TCP tunnel",
	}

	serverCommand := &cobra.Command{
		Use: "server",
		Short: "Run the remote proxy server",
		RunE: func(cmds *cobra.Command, args []string) error {
			return bore.NewServer(serverMinPort, serverSecret).Listen()
		},
	}
	serverCommand.Flags().IntVar(&serverMinPort, "min-port", 1023, "Minimum TCP port number to accept")
	serverCommand.Flags().StringVarP(&serverSecret, "secret", "s", "", "Optional secret for authentication")
	rootCmd.AddCommand(serverCommand)

	localCommand := &cobra.Command{
		Use: "local [local_port]",
		Short: "Starts a local proxy to the remote server",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			localPort, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			return bore.NewClient(remoteHost, remotePort, localPort, localSecret).Start()
		},
	}
	localCommand.Flags().IntVarP(&remotePort, "port", "p", 0, "Optional port on the remote server to select")
	localCommand.Flags().StringVarP(&localSecret, "secret", "s", "", "Optional secret for authentication")
	localCommand.Flags().StringVarP(&remoteHost, "to", "t", "", "Address of the remote server to expose local ports to")
	localCommand.Flags().IntVar(&localPort, "local_port", 0, "The local port to listen on")
	localCommand.MarkFlagRequired("to")
	rootCmd.AddCommand(localCommand)

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Print(err)
	}
}
