// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create resources",
	Long:  `Create organizations and other resources.`,
}

var createOrgCmd = &cobra.Command{
	Use:   "org <name> <domain>",
	Short: "Create a new organization",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.CreateOrg(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.AddCommand(createOrgCmd)
}
