// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Manage workspaces",
	Long:    `Commands for viewing and managing Grist workspaces.`,
}

var workspaceGetCmd = &cobra.Command{
	Use:   "get <workspace-id>",
	Short: "Get workspace details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wsID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid workspace ID: %s\n", args[0])
			os.Exit(1)
		}
		gristtools.DisplayWorkspace(wsID)
	},
}

var workspaceAccessCmd = &cobra.Command{
	Use:   "access <workspace-id>",
	Short: "Get workspace access permissions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wsID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid workspace ID: %s\n", args[0])
			os.Exit(1)
		}
		gristtools.DisplayWorkspaceAccess(wsID)
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(workspaceGetCmd)
	workspaceCmd.AddCommand(workspaceAccessCmd)
}
