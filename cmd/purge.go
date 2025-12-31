// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bdmorin/gristle/gristapi"
	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge document history",
	Long:  `Purge old document history, keeping only the most recent states.`,
}

var purgeDocCmd = &cobra.Command{
	Use:   "doc <doc-id> [num-states]",
	Short: "Purge document history",
	Long:  `Purge document history, keeping only the specified number of most recent states (default: 3)`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		docID := args[0]
		nbStates := 3 // default

		if len(args) == 2 {
			var err error
			nbStates, err = strconv.Atoi(args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid number of states: %s\n", args[1])
				os.Exit(1)
			}
		}

		gristapi.PurgeDoc(docID, nbStates)
	},
}

func init() {
	rootCmd.AddCommand(purgeCmd)
	purgeCmd.AddCommand(purgeDocCmd)
}
