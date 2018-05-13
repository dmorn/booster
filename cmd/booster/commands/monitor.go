/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
	monitorNode      bool
	monitorBandwidth bool
)

var stream = func(addr string, features []protocol.Message, f func(i interface{})) {
	b, err := booster.New(pport, bport)
	if err != nil {
		fmt.Println(err)
		return
	}

	stream, err := b.Monitor(context.Background(), "tcp", addr, features)
	if err != nil {
		fmt.Println(err)
		return
	}

	for n := range stream {
		f(n)
	}
}

var monitorCmd = &cobra.Command{
	Use:   "monitor [host:port -- optional]",
	Short: "monitors node and bandwidth activity",
	Long:  `monitor listents (by default) on the local node for each activity update, and logs it.`,
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

var monitorNodeCmd = &cobra.Command{
	Use:   "node [host:port -- optional]",
	Short: "monitors the node's activity",
	Long:  `monitor listents (by default) on the local node for each node activity update, and logs it.`,
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

var monitorNetCmd = &cobra.Command{
	Use:   "net [download|upload] [host:port -- optional]",
	Short: "monitors the network's activity",
	Long:  `monitor listents (by default) on the local node for each net activity update, and logs it.`,
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
				log.Error.Printf("booster: monitor net: unrecognised payload: %+v", i)
				return
			}

			if pb.Type == target {
				log.Println(pb)
			}
		})
	},
}
