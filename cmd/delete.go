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

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Long:  `Delete organizations, workspaces, documents, or users.`,
}

var deleteOrgCmd = &cobra.Command{
	Use:   "org <org-id> <org-name>",
	Short: "Delete an organization",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		orgID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid org ID: %s\n", args[0])
			os.Exit(1)
		}
		gristtools.DeleteOrg(orgID, args[1])
	},
}

var deleteWorkspaceCmd = &cobra.Command{
	Use:   "workspace <workspace-id>",
	Short: "Delete a workspace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wsID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid workspace ID: %s\n", args[0])
			os.Exit(1)
		}
		gristtools.DeleteWorkspace(wsID)
	},
}

var deleteDocCmd = &cobra.Command{
	Use:   "doc <doc-id>",
	Short: "Delete a document",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DeleteDoc(args[0])
	},
}

var deleteUserCmd = &cobra.Command{
	Use:   "user <user-id>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		userID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid user ID: %s\n", args[0])
			os.Exit(1)
		}
		gristtools.DeleteUser(userID)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deleteOrgCmd)
	deleteCmd.AddCommand(deleteWorkspaceCmd)
	deleteCmd.AddCommand(deleteDocCmd)
	deleteCmd.AddCommand(deleteUserCmd)
}
