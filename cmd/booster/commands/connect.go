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
	Use:   "connect [host:port -- optional] [host:port]",
	Short: "connect two nodes together",
	Long:  `connect asks (by default) the local node to perform the necessary steps required to connect an external node to itself. Returns the added node identifier if successfull. You can use the 'inspect' command to monitor node activity.`,
	Args:  cobra.RangeArgs(1, 2),
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
