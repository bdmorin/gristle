// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	mcpserver "github.com/bdmorin/gristle/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:     "mcp",
	Aliases: []string{"serve"},
	Short:   "Start MCP server for AI assistant integration",
	Long: `Starts the Model Context Protocol (MCP) server on stdio.
This allows AI assistants to interact with your Grist instance.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := mcpserver.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
