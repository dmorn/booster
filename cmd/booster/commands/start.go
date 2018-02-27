package commands

import (
	"fmt"

	"github.com/danielmorandini/booster/booster"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a booster node",
	Long:  `starts a booster proxy and node. Both are tcp servers, their listening port will be logged`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		b, err := booster.New(pport, bport)
		if err != nil {
			fmt.Println(err)
			return
		}

		if err = b.Run(); err != nil {
			fmt.Println(err)
		}
	},
}
