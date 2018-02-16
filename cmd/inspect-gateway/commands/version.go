package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "prints inspect-gateway version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s, BuildTime: %s\n\n", Version, BuildTime)
	},
}
