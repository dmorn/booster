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
	verbose     bool
	pport       int
	bport       int
	apiEndpoint string
)

var rootCmd = &cobra.Command{
	Use:   "inspect-gateway",
	Short: "inspect-gateway is an http client that collects data from a booster node and sends it as json packets to a target API",
	Long:  `inspect-gateway tries to communicate with a booster node, defaulting on a local one, making an inspect request. Collects the data and POSTs it to the target API using JSON encoding`,
}

func Execute() {
	// parse flags
	startCmd.Flags().StringVar(&apiEndpoint, "api", ":4000", "API endpoint address")
	startCmd.Flags().IntVar(&pport, "pport", 1080, "proxy listening port")
	startCmd.Flags().IntVar(&bport, "bport", 4884, "booster listening port")

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// add commands
	rootCmd.AddCommand(startCmd)

	// execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
