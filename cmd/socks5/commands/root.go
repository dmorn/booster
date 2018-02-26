package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildTime string
)

var (
	pport int
)

var rootCmd = &cobra.Command{
	Use:   "socks5",
	Short: "socks5 is a SOCKS5 proxy server",
}

func Execute() {
	// parse flags
	startCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")

	// add commands
	rootCmd.AddCommand(startCmd, versionCmd)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
