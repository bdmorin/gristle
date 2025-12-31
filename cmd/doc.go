// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/bdmorin/gristle/gristapi"
	"github.com/bdmorin/gristle/gristtools"
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Manage documents",
	Long:  `Commands for viewing, exporting, and managing Grist documents.`,
}

var docGetCmd = &cobra.Command{
	Use:   "get <doc-id>",
	Short: "Get document details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayDoc(args[0])
	},
}

var docAccessCmd = &cobra.Command{
	Use:   "access <doc-id>",
	Short: "Get document access permissions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayDocAccess(args[0])
	},
}

var docWebhooksCmd = &cobra.Command{
	Use:   "webhooks <doc-id>",
	Short: "List document webhooks",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		gristtools.DisplayDocWebhooks(args[0])
	},
}

var docExportCmd = &cobra.Command{
	Use:   "export <doc-id> <format>",
	Short: "Export document",
	Long:  `Export document in the specified format: excel or grist`,
	Args:  cobra.ExactArgs(2),
	ValidArgs: []string{"excel", "grist"},
	Run: func(cmd *cobra.Command, args []string) {
		docID := args[0]
		format := args[1]

		switch format {
		case "excel":
			gristtools.ExportDocExcel(docID)
		case "grist":
			gristtools.ExportDocGrist(docID)
		default:
			_ = cmd.Help()
		}
	},
}

var docTableCmd = &cobra.Command{
	Use:   "table <doc-id> <table-name>",
	Short: "Export table as CSV",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		gristapi.GetTableContent(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(docCmd)
	docCmd.AddCommand(docGetCmd)
	docCmd.AddCommand(docAccessCmd)
	docCmd.AddCommand(docWebhooksCmd)
	docCmd.AddCommand(docExportCmd)
	docCmd.AddCommand(docTableCmd)
}
