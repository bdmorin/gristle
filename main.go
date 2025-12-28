// SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg
//
// SPDX-License-Identifier: MIT

// Main program
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/bdmorin/gristle/gristapi"
	"github.com/bdmorin/gristle/gristtools"
	mcpserver "github.com/bdmorin/gristle/mcp"
	"github.com/bdmorin/gristle/tui"
)

var version = "Undefined"

func main() {
	// Define the options
	optionOutput := flag.String("o", "table", "Output format (table, json)")
	optionJSON := flag.Bool("json", false, "Output as JSON (shorthand for -o json)")

	flag.Parse()

	// Set output format
	if *optionJSON || *optionOutput == "json" {
		gristtools.SetOutput("json")
	} else {
		gristtools.SetOutput("table")
	}

	args := flag.Args()

	// No args = launch interactive TUI
	if len(args) < 1 {
		if err := tui.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// CLI mode for scripting
	switch arg1 := args[0]; arg1 {
	case "mcp", "serve":
		// Start MCP server for AI assistant integration
		if err := mcpserver.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
	case "config":
		gristtools.Config()
	case "version":
		gristtools.Version(version)
	case "get":
		handleGet(args)
	case "move":
		handleMove(args)
	case "purge":
		handlePurge(args)
	case "delete":
		handleDelete(args)
	case "import":
		handleImport(args)
	case "create":
		handleCreate(args)
	default:
		gristtools.Help()
		flag.PrintDefaults()
	}
}

func handleGet(args []string) {
	if len(args) < 2 {
		gristtools.Help()
		return
	}

	switch args[1] {
	case "org":
		switch len(args) {
		case 2:
			gristtools.DisplayOrgs()
		case 3:
			gristtools.DisplayOrg(args[2])
		case 4:
			switch args[3] {
			case "access":
				gristtools.DisplayOrgAccess(args[2])
			case "usage":
				gristtools.GetOrgUsageSummary(args[2])
			default:
				gristtools.Help()
			}
		default:
			gristtools.Help()
		}

	case "doc":
		switch len(args) {
		case 3:
			gristtools.DisplayDoc(args[2])
		case 4:
			docId := args[2]
			switch args[3] {
			case "access":
				gristtools.DisplayDocAccess(docId)
			case "grist":
				gristtools.ExportDocGrist(docId)
			case "excel":
				gristtools.ExportDocExcel(docId)
			case "webhooks":
				gristtools.DisplayDocWebhooks(docId)
			default:
				fmt.Println("Options: access, grist, excel, webhooks")
			}
		case 5:
			if args[3] == "table" {
				gristapi.GetTableContent(args[2], args[4])
			} else {
				gristtools.Help()
			}
		default:
			gristtools.Help()
		}

	case "workspace":
		switch len(args) {
		case 3:
			if wsId, err := strconv.Atoi(args[2]); err == nil {
				gristtools.DisplayWorkspace(wsId)
			}
		case 4:
			if args[3] == "access" {
				if wsId, err := strconv.Atoi(args[2]); err == nil {
					gristtools.DisplayWorkspaceAccess(wsId)
				}
			}
		default:
			gristtools.Help()
		}

	case "users":
		gristtools.DisplayUserMatrix()

	default:
		gristtools.Help()
	}
}

func handleMove(args []string) {
	if len(args) < 3 {
		gristtools.Help()
		return
	}

	switch args[1] {
	case "doc":
		if len(args) >= 5 && args[3] == "workspace" {
			if wsId, err := strconv.Atoi(args[4]); err == nil {
				gristapi.MoveDoc(args[2], wsId)
			}
		}
	case "docs":
		if len(args) >= 6 {
			fromId, err1 := strconv.Atoi(args[3])
			toId, err2 := strconv.Atoi(args[5])
			if err1 == nil && err2 == nil {
				gristapi.MoveAllDocs(fromId, toId)
			}
		}
	default:
		gristtools.Help()
	}
}

func handlePurge(args []string) {
	if len(args) < 3 {
		gristtools.Help()
		return
	}

	if args[1] == "doc" {
		docId := args[2]
		nbHisto := 3
		if len(args) == 4 {
			if nb, err := strconv.Atoi(args[3]); err == nil {
				nbHisto = nb
			}
		}
		gristapi.PurgeDoc(docId, nbHisto)
	} else {
		gristtools.Help()
	}
}

func handleDelete(args []string) {
	if len(args) < 3 {
		gristtools.Help()
		return
	}

	switch args[1] {
	case "org":
		if len(args) == 4 {
			if orgId, err := strconv.Atoi(args[2]); err == nil {
				gristtools.DeleteOrg(orgId, args[3])
			}
		}
	case "workspace":
		if wsId, err := strconv.Atoi(args[2]); err == nil {
			gristtools.DeleteWorkspace(wsId)
		}
	case "user":
		if userId, err := strconv.Atoi(args[2]); err == nil {
			gristtools.DeleteUser(userId)
		}
	case "doc":
		gristtools.DeleteDoc(args[2])
	default:
		gristtools.Help()
	}
}

func handleImport(args []string) {
	if len(args) > 1 && args[1] == "users" {
		gristtools.ImportUsers()
	} else {
		gristtools.Help()
	}
}

func handleCreate(args []string) {
	if len(args) >= 4 && args[1] == "org" {
		gristtools.CreateOrg(args[2], args[3])
	} else {
		gristtools.Help()
	}
}
