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
	Use:   "ctrl [host:port -- optional] [stop|restart]",
	Short: "perform control operation on node",
	Long:  `ctrl tells the local node (by default) to perform an operation.`,
	Args:  cobra.RangeArgs(1, 2),
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
