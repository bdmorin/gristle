// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "User management",
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "Display user access matrix across all orgs/workspaces",
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayUserMatrix()
	},
}

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(usersListCmd)
}
