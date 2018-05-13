/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildTime string
)

var (
	pport   int
	bport   int
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "booster",
	Short: "booster is a peer-to-peer network interface balancer",
	Long:  `booster creates a network of peer nodes, each of them with an active Internet connection, balancing the network usage between their interfaces`,
}

func Execute() {
	// parse flags
	startCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	startCmd.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	// add commands
	monitorCmd.AddCommand(monitorNodeCmd, monitorNetCmd)
	rootCmd.AddCommand(versionCmd, startCmd, connectCmd, disconnectCmd, monitorCmd, ctrlCmd)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
