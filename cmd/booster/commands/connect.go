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

var connectCmd = &cobra.Command{
	Use:   "connect [x] [y]",
	Short: "connect makes y boost x",
	Long: `connect makes y boost x.

x: node address (format host:port) that will be boosted (optional, default localhost:4884)
y: node address (format host:port) that will boost

	Example:
	bin/booster connect localhost:4885
	2018/05/21 15:35:35.078475 booster: -> connect: localhost:4885
	2018/05/21 15:35:35.090891 booster: <- hello: ::1 1080-4884
	connected to (localhost:4885): 4d8ef492e992c2394bb0a80611bc04f72cb2d417

The sha1 hash returned is the identifier of the newly connected node in the network.
	`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		laddr := "localhost:4884"
		raddr := ""
		if len(args) == 2 {
			laddr = args[0]
			raddr = args[1]
		} else {
			raddr = args[0]
		}

		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		id, err := b.Connect(context.Background(), "tcp", laddr, raddr)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("connected to (%v): %v\n", raddr, id)
	},
}
