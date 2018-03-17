package commands

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [host:port -- optional]",
	Short: "inspects the node's activity",
	Long:  `inspect listents (by default) on the local node for each node activity update, and logs it.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		if len(args) == 1 {
			addr = args[0]

		}
		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		stream, err := b.Inspect(context.Background(), "tcp", addr)
		if err != nil {
			fmt.Println(err)
			return
		}

		for n := range stream {
			log.Println(n)
		}
	},
}
