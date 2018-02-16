package commands

import (
	"fmt"

	"github.com/danielmorandini/booster-network/socks5"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a socks5 proxy server",
	Long:  ``,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		p := socks5.SOCKS5()
		if err := p.ListenAndServe(pport); err != nil {
			fmt.Println(err)
		}
	},
}
