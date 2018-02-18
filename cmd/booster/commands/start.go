package commands

import (
	"fmt"

	"github.com/danielmorandini/booster-network/booster"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a booster node",
	Long:  `starts a booster proxy and node. Both are tcp servers, their listening port will be logged`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		b := booster.NewBoosterDefault()
		if err := b.Start(pport, bport); err != nil {
			fmt.Println(err)
		}
	},
}
