// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Grist connection settings",
	Long: `Interactively configure your Grist API token and URL.
Settings are saved to ~/.gristle`,
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.Config()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
