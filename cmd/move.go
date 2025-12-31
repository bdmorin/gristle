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

var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move resources",
	Long:  `Move documents between workspaces.`,
}

var moveDocCmd = &cobra.Command{
	Use:   "doc <doc-id> <workspace-id>",
	Short: "Move a document to a different workspace",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		wsID, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid workspace ID: %s\n", args[1])
			os.Exit(1)
		}
		gristapi.MoveDoc(args[0], wsID)
	},
}

var moveDocsCmd = &cobra.Command{
	Use:   "docs <from-workspace-id> <to-workspace-id>",
	Short: "Move all documents from one workspace to another",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fromID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid from workspace ID: %s\n", args[0])
			os.Exit(1)
		}
		toID, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid to workspace ID: %s\n", args[1])
			os.Exit(1)
		}
		gristapi.MoveAllDocs(fromID, toID)
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)
	moveCmd.AddCommand(moveDocCmd)
	moveCmd.AddCommand(moveDocsCmd)
}
