package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielmorandini/booster-network/node"
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect [id]",
	Short: "disconnect two nodes",
	Long:  `disconnect aks (by default) the local node to perform the necessary steps required to disconnect completely a node from itself.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := strings.Join(args, " ")
		b := node.NewBoosterDefault()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := b.Disconnect(ctx, "tcp", boosterAddr, id); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("disconnected from: %v\n", id)
	},
}
