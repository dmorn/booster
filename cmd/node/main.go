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
			b := node.Booster()

			go func() {
				log.Fatal(b.Proxy.ListenAndServe(pport))
			}()
			log.Fatal(b.ListenAndServe(bport))
		},
	}

	cmdStart.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	cmdStart.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	var cmdRegister = &cobra.Command{
		Use:   "register host:port",
		Short: "register a remote booster node",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dest := strings.Join(args, " ")
			b := new(node.Booster)
			ctx := context.Background()

			if err := b.Register(ctx, "tcp", boosterAddr, dest); err != nil {
				log.Fatal(err)
			}
		},
	}

	cmdRegister.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")

	var rootCmd = &cobra.Command{Use: "booster"}
	rootCmd.AddCommand(cmdStart, cmdRegister)

	rootCmd.Execute()
}
