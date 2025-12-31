// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import resources",
	Long:  `Import users and other resources.`,
}

var importUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Import users interactively",
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.ImportUsers()
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importUsersCmd)
}
