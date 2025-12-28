# CLAUDE.md - Project Context for AI Agents

## Project Overview

**gristctl** is a command-line utility for interacting with [Grist](https://www.getgrist.com/), a platform that combines relational database capabilities with spreadsheet flexibility. This tool enables automation and management of Grist documents, workspaces, organizations, and users.

### Key Components

- **CLI** (`main.go`) - Command-line interface using Cobra
- **TUI** (`tui/`) - Interactive terminal UI using Bubble Tea
- **MCP Server** (`mcp/`) - Model Context Protocol server for AI agent integration
- **Grist API Client** (`gristapi/`) - HTTP client for Grist REST API
- **Common utilities** (`common/`) - Shared configuration and helpers

### Technology Stack

- **Language**: Go 1.23+
- **CLI Framework**: Cobra
- **TUI Framework**: Bubble Tea (charmbracelet)
- **MCP Library**: mark3labs/mcp-go
- **Output Formats**: Table, JSON, CSV

## Testing Environment

Each agent session has access to a dedicated Grist playground for testing:

**Playground URL**: https://grist.hexxa.dev/o/docs/uFiFazkXAEwx/vibe-kanban-playground

Use this workspace to create test documents, tables, and data when developing or testing gristctl features. This is a safe sandbox environment for experimentation.

## Building and Testing

```bash
# Build the project
go build

# Run tests
go test ./...

# Run the CLI
./gristctl --help

# Run the MCP server
./gristctl mcp

# Run the TUI
./gristctl tui
```

## Configuration

gristctl reads configuration from `~/.gristctl`:

```ini
GRIST_TOKEN="your-api-token"
GRIST_URL="https://your-grist-instance.com"
```

## Current Development Focus

The project is expanding Grist API coverage and improving the MCP server. See `docs/research/` for MCP best practices research.

### Priority Areas (from research)

1. **Security**: Input validation, path traversal prevention, rate limiting
2. **Performance**: HTTP client reuse, JSON optimization
3. **Reliability**: Panic recovery, context cancellation
4. **Testing**: Unit tests, integration tests, fuzz testing
5. **Code Quality**: Handler extraction, interfaces, structured logging

## API Implementation Status

Many Grist API endpoints are tracked as tasks in vibe-kanban. Check the task list for current implementation status of:
- Records APIs
- Webhooks APIs
- Attachments APIs
- SCIM user management
- Service accounts
- SQL queries

## Code Style

- Follow standard Go conventions
- Use structured error handling
- Prefer table-driven tests
- Keep handlers focused and testable

## MCP Server Tools

The MCP server exposes these tools for AI agents:

| Tool | Description |
|------|-------------|
| `list_orgs` | List all accessible organizations |
| `list_workspaces` | List workspaces in an org (requires `org_id`) |
| `list_docs` | List documents in a workspace (requires `workspace_id`) |
| `get_doc` | Get document details + table list (requires `doc_id`) |
| `get_doc_tables` | Get tables with column info (requires `doc_id`) |
| `export_doc` | Export to Excel or Grist format (requires `doc_id`, `format`) |

## TUI Features

- Launches by default when running `./gristctl` with no args
- Hierarchical navigation: Orgs → Workspaces → Documents → Actions
- Keyboard: `↑/k` `↓/j` navigate, `Enter` select, `Esc` back, `q` quit
- Document actions: View Tables, Export (CSV/Excel/Grist), View Access, Delete

## File Structure

```
grist-ctl/
├── main.go              # Entry point, command routing
├── tui/
│   ├── tui.go           # Bubbletea model, views, navigation
│   ├── styles.go        # Lipgloss styling
│   └── keys.go          # Keybindings
├── mcp/
│   └── server.go        # MCP server with tools
├── gristapi/            # API client
├── gristtools/          # CLI display helpers
├── common/              # Utilities
├── docs/research/       # MCP best practices research
└── go.mod
```
