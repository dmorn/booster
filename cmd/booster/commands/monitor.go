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
	"sync"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "monitor monitors the activity of a booster node.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}
	},
}

var monitorProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "proxy will monitor proxy updates.",
	Long: `proxy will monitor proxy updates. Logs a stream of json encoded data.

	Example:
	bin/booster monitor proxy
	`,
	Args: cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		stream(target, protocol.MessageProxyUpdate)
	},
}

var monitorNetCmd = &cobra.Command{
	Use:   "net [download|upload]",
	Short: "net will monitor network stats.",
	Long: `net will monitor network stats. Logs a stream of json encoded data.
	
	Example:
	bin/booster monitor net download
	`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stream(target, protocol.MessageNetworkStatus)
	},
}

var stream = func(addr string, feature protocol.Message) {
	b, err := booster.New(pport, bport)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	wg.Add(1)
	if err := b.Monitor(ctx, "tcp", "localhost:4884", booster.Inspection{
		Feature: feature,
		Run: func(m packet.Module) error {
			log.Info.Printf("%s", m.Payload())
			return nil
		},
		PostRun: func(err error) {
			log.Info.Println("monitor stopping..")
			if err != nil {
				log.Error.Println(err)
			}

			wg.Done()
		},
	}); err != nil {
		fmt.Println(err)
	}

	wg.Wait()
}

