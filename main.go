package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/oraoto/go-bore/bore"
	"github.com/spf13/cobra"
	"fmt"
)


var (
	serverSecret string
	serverMinPort int
	localSecret string
	localPort int
	localTo string
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

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Print(err)
	}
}
