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
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect [x] [y]",
	Short: "disconnect removes y from the network of x",
	Long: `disconnect removes y from the network of x.

x: node address (format host:port) that was boosted by y (optional, default localhost:4884)
y: node identifier of the node that will be disconnected

	Example:
	bin/booster disconnect 4d8ef492e992c2394bb0a80611bc04f72cb2d417
	2018/05/21 15:35:48.455137 booster: -> disconnect: 4d8ef492e992c2394bb0a80611bc04f72cb2d417
	2018/05/21 15:35:48.456494 booster: <- hello: ::1 1080-4884
	disconnected from: localhost:4884
	`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		id := ""
		if len(args) == 2 {
			addr = args[0]
			id = args[1]
		} else {
			id = args[0]
		}

		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = b.Disconnect(context.Background(), "tcp", addr, id)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("disconnected from: %v\n", addr)
	},
}
