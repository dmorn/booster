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
	target string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: os.Args[0] + " is a peer-to-peer network interface balancer",
	Long: os.Args[0] + ` is a peer-to-peer network interface balancer.
	
Every running instance is composed by two parts, a booster node and a proxy. The former is responsible for managing the bahaviour of the node inside the booster network, while the latter is only a forwarding proxy.

	Usage example:
	Step 1: bin/booster start <- start a node
	Step 2: make your system use the local proxy spawned by booster
	Step 3: bin/booster connect another_node_host:port <- make another_node boost this node
	
	You're all set!	
	`,
}

func Execute() {
	// parse flags
	startCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	startCmd.Flags().IntVar(&bport, "bport", 4884, "booster listening port")
	monitorCmd.Flags().StringVarP(&target, "target", "t", "localhost:4884", "address to monitor")

	// add commands
	monitorCmd.AddCommand(monitorProxyCmd, monitorNetCmd)
	rootCmd.AddCommand(versionCmd, startCmd, connectCmd, disconnectCmd, monitorCmd, ctrlCmd)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
