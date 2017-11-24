package main

import (
	"context"
	"log"
	"strings"

	"github.com/danielmorandini/booster-network/node"
	"github.com/spf13/cobra"
)

func main() {
	var pport int
	var bport int

	var boosterAddr string

	var cmdStart = &cobra.Command{
		Use:   "start",
		Short: "starts a booster node",
		Long:  ``,
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			b := node.BOOSTER()

			if err := b.Start(pport, bport); err != nil {
				log.Fatal(err)
			}
		},
	}

	cmdStart.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	cmdStart.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	var cmdConnect = &cobra.Command{
		Use:   "connect host:port",
		Short: "pair with a remote booster node",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dest := strings.Join(args, " ")
			b := new(node.Booster)
			ctx := context.Background()

			if err := b.Pair(ctx, "tcp", boosterAddr, dest); err != nil {
				log.Fatal(err)
			}
		},
	}

	var cmdDisconnect = &cobra.Command{
		Use:   "disconnect host:port",
		Short: "disconnect a previously connected remote booster node",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dest := strings.Join(args, " ")
			b := new(node.Booster)
			ctx := context.Background()

			if err := b.Unpair(ctx, "tcp", boosterAddr, dest); err != nil {
				log.Fatal(err)
			}
		},
	}

	cmdConnect.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")
	cmdDisconnect.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")

	var rootCmd = &cobra.Command{Use: "booster"}
	rootCmd.AddCommand(cmdStart, cmdConnect, cmdDisconnect)

	rootCmd.Execute()
}
