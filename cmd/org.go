// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage organizations",
	Long:  `Commands for listing, viewing, and managing Grist organizations.`,
}

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all organizations",
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayOrgs()
	},
}

var orgGetCmd = &cobra.Command{
	Use:   "get <org-id>",
	Short: "Get organization details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayOrg(args[0])
	},
}

var orgAccessCmd = &cobra.Command{
	Use:   "access <org-id>",
	Short: "Get organization member access",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayOrgAccess(args[0])
	},
}

var orgUsageCmd = &cobra.Command{
	Use:   "usage <org-id>",
	Short: "Get organization usage summary",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.GetOrgUsageSummary(args[0])
	},
}

func init() {
	rootCmd.AddCommand(orgCmd)
	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgGetCmd)
	orgCmd.AddCommand(orgAccessCmd)
	orgCmd.AddCommand(orgUsageCmd)
}
