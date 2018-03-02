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
	boosterAddr string
)

var rootCmd = &cobra.Command{
	Use:   "booster",
	Short: "booster is a peer-to-peer network interface balancer",
	Long:  `booster creates a network of peer nodes, each of them with an active Internet connection, balancing the network usage between their interfaces`,
}

func Execute() {
	// parse flags
	startCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	startCmd.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	dialCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	dialCmd.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	// add commands
	rootCmd.AddCommand(versionCmd, startCmd, dialCmd)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
