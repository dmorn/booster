package commands

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/log"
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect [host:port -- optional] [node_id]",
	Short: "disconnect two nodes",
	Long:  `disconnect aks (by default) the local node to perform the necessary steps required to disconnect completely a node from itself.`,
	Args:  cobra.RangeArgs(1, 2),
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

		fmt.Printf("disconnected from: %v\n", id)
	},
}
