package commands

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster-network/booster"
	"github.com/danielmorandini/booster-network/node"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "inspects the node's activity",
	Long:  `inspect listents (by default) on the local node for each node activity update, and logs it.`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		b := booster.NewBoosterDefault()
		stream := make(chan *node.Node)
		errc := make(chan error)
		ctx := context.Background()

		err := b.InspectStream(ctx, "tcp", boosterAddr, stream, errc)
		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			select {
			case err := <-errc:
				fmt.Println(err)
				return
			case node := <-stream:
				b.Println(node)
			}
		}
	},
}
