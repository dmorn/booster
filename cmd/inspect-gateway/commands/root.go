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
	targetAddr  string
	boosterAddr string
)

var rootCmd = &cobra.Command{
	Use:   "booster",
	Short: "inspect-gateway is an http client that collects data from a booster node and sends it as json packets to a target API",
	Long: `inspect-gateway tries to communicate with a booster node, defaulting on a local one, making an inspect request. Collects
	the data and POSTs it to the target API using JSON encoding`,
}

func Execute() {
	// parse flags
	startCmd.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")
	startCmd.Flags().StringVarP(&targetAddr, "taddr", "t", ":4000", "target API address")

	// add commands
	rootCmd.AddCommand(versionCmd, startCmd)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
