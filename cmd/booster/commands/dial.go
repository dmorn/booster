package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/danielmorandini/booster/booster"
	"github.com/spf13/cobra"
)

var dialCmd = &cobra.Command{
	Use:   "dial",
	Short: "starts a booster node",
	Long:  `starts a booster proxy and node. Both are tcp servers, their listening port will be logged`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		dest := strings.Join(args, " ")
		ctx := context.TODO()
		conn, err := b.DialContext(ctx, "tcp", dest)
		if err != nil {
			fmt.Println(err)
			return
		}
		conn.Conn.Close()
	},
}
