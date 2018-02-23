package commands

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster/socks5"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a socks5 proxy server",
	Long:  ``,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()
		p := socks5.SOCKS5()
		if err := p.ListenAndServe(ctx, pport); err != nil {
			fmt.Println(err)
		}
	},
}
