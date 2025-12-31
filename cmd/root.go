// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/bdmorin/gristle/gristtools"
	"github.com/bdmorin/gristle/tui"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
	jsonOutput   bool
	Version      = "dev" // Set via ldflags during build
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "gristle",
	Short: "Gristle - The meaty CLI for Grist",
	Long: `Gristle is a command-line tool for interacting with Grist.
It provides commands to manage organizations, workspaces, documents, and more.

Run with no arguments to launch the interactive TUI.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Special case: no subcommand launches TUI
		if len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
			// If just running "gristle" or "gristle --help", handle normally
			if len(os.Args) == 1 {
				if err := tui.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				return
			}
		}
		// Otherwise show help
		_ = cmd.Help()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Set output format globally before any command runs
		if jsonOutput || outputFormat == "json" {
			gristtools.SetOutput("json")
		} else {
			gristtools.SetOutput("table")
		}
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON (shorthand for -o json)")
}
