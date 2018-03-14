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
	pport       int
	bport       int
	APIEndpoint string
)

var rootCmd = &cobra.Command{
	Use:   "booster",
	Short: "inspect-gateway is an http client that collects data from a booster node and sends it as json packets to a target API",
	Long: `inspect-gateway tries to communicate with a booster node, defaulting on a local one, making an inspect request. Collects
	the data and POSTs it to the target API using JSON encoding`,
}

func Execute() {
	// parse flags
	startCmd.Flags().StringVar(&APIEndpoint, "api", ":4000", "API endpoint address")
	startCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	startCmd.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	// add commands
	rootCmd.AddCommand(startCmd)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
