package main

import (
	"context"
	"fmt"
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
			b := node.NewBoosterDefault()

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
			b := node.NewBoosterDefault()
			ctx := context.Background()

			id, err := b.Connect(ctx, "tcp", boosterAddr, dest)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Printf("connected to (%v): %v\n", dest, id)
		},
	}

	var cmdInspect = &cobra.Command{
		Use:   "inspect",
		Short: "inspect the remote nodes connected to the target node",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			b := node.NewBoosterDefault()
			ctx := context.Background()
			stream := make(chan *node.Node)
			errc := make(chan error)

			err := b.InspectStream(ctx, "tcp", boosterAddr, stream, errc)
			if err != nil {
				fmt.Println(err)
				return
			}

			for {
				select {
				case err := <-errc:
					fmt.Println(err)
					return
				case node := <-stream:
					b.Println(node)
				}
			}
		},
	}

	var cmdDisconnect = &cobra.Command{
		Use:   "disconnect id",
		Short: "disconnect a previously connected remote booster node",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := strings.Join(args, " ")
			b := node.NewBoosterDefault()
			ctx := context.Background()

			if err := b.Disconnect(ctx, "tcp", boosterAddr, id); err != nil {
				fmt.Println(err)
				return
			}

			fmt.Printf("disconnected from: %v\n", id)
		},
	}

	cmdConnect.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")
	cmdInspect.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")
	cmdDisconnect.Flags().StringVarP(&boosterAddr, "baddr", "b", ":4884", "booster address")

	var rootCmd = &cobra.Command{Use: "booster"}
	rootCmd.AddCommand(cmdStart, cmdConnect, cmdDisconnect, cmdInspect)

	rootCmd.Execute()
}
