package commands

import (
	"fmt"
	"net"

	"github.com/danielmorandini/booster/socks5"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a socks5 proxy server",
	Long:  ``,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		p := socks5.New(new(net.Dialer))

		if err := p.Run(pport); err != nil {
			fmt.Println(err)
		}
	},
}
