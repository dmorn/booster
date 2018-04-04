package commands

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/protocol"
	"github.com/spf13/cobra"
)

var (
	inspectNode      bool
	inspectBandwidth bool
)

var stream = func(addr string, features []protocol.Message, f func(i interface{})) {
	b, err := booster.New(pport, bport)
	if err != nil {
		fmt.Println(err)
		return
	}

	stream, err := b.Inspect(context.Background(), "tcp", addr, features)
	if err != nil {
		fmt.Println(err)
		return
	}

	for n := range stream {
		f(n)
	}
}

var inspectCmd = &cobra.Command{
	Use:   "inspect [host:port -- optional]",
	Short: "inspects node and bandwidth activity",
	Long:  `inspect listents (by default) on the local node for each activity update, and logs it.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		if len(args) == 1 {
			addr = args[0]

		}

		features := []protocol.Message{protocol.MessageNode, protocol.MessageBandwidth}
		stream(addr, features, func(i interface{}) {
			log.Println(i)
		})
	},
}

var inspectNodeCmd = &cobra.Command{
	Use:   "node [host:port -- optional]",
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

		features := []protocol.Message{protocol.MessageNode}
		stream(addr, features, func(i interface{}) {
			log.Println(i)
		})
	},
}

var inspectNetCmd = &cobra.Command{
	Use:   "net [download|upload] [host:port -- optional]",
	Short: "inspects the network's activity",
	Long:  `inspect listents (by default) on the local node for each net activity update, and logs it.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		target := args[0]
		if len(args) == 2 {
			addr = args[1]
		}

		features := []protocol.Message{protocol.MessageBandwidth}
		stream(addr, features, func(i interface{}) {
			pb, ok := i.(*protocol.PayloadBandwidth)
			if !ok {
				log.Error.Printf("booster: inspect net: unrecognised payload: %+v", i)
			}

			if pb.Type == target {
				log.Println(pb)
			}
		})
	},
}
