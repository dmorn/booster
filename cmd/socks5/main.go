package main

import (
	"log"

	"github.com/danielmorandini/booster-network/socks5"
	"github.com/spf13/cobra"
)

func main() {
	var pport int

	var cmdStart = &cobra.Command{
		Use:   "start",
		Short: "starts a socks5 proxy server",
		Long:  ``,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			p := socks5.SOCKS5()
			log.Fatal(p.ListenAndServe(pport))
		},
	}

	cmdStart.Flags().IntVar(&pport, "pport", 1080, "socks5 listening port")

	var rootCmd = &cobra.Command{Use: "socks5"}
	rootCmd.AddCommand(cmdStart)

	rootCmd.Execute()
}
