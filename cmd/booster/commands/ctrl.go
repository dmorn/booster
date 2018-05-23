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

var ctrlCmd = &cobra.Command{
	Use:   "ctrl [x] [o]",
	Short: "ctrl performs operation o on node x.",
	Long: `ctrl performs operation o on node x.

x: node address (format host:port) that will be contacted (optional, default localhost:4884)
o: operation to be performed. Available options:
	- stop
	- restart

	Example:
	bin/booster ctrl restart
	2018/05/21 15:51:04.017960 booster: -> ctrl: localhost:4884
	2018/05/21 15:51:04.019280 booster: <- hello: ::1 1080-4884
	ctrl operation performed (localhost:4884)
	`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		addr := "localhost:4884"
		rawOp := ""
		if len(args) == 2 {
			addr = args[0]
			rawOp = args[1]
		} else {
			rawOp = args[0]
		}

		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		op, err := protocol.OperationFromString(rawOp)
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := b.Ctrl(context.Background(), "tcp", addr, op); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("ctrl operation performed (%v)\n", addr)
	},
}
