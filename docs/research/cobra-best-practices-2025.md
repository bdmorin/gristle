# Cobra CLI Framework Best Practices - 2025

## Executive Summary

This document provides comprehensive research on modern Cobra CLI framework best practices, patterns, and design principles as of 2025. The research is based on official Cobra documentation, real-world implementations (kubectl, gh, hugo), and modern CLI design principles from the Command Line Interface Guidelines (CLIG).

**Key Findings:**
- Cobra remains the dominant CLI framework for Go (173,000+ projects including Kubernetes, Docker, Hugo, GitHub CLI)
- Modern best practices emphasize security, testability, and human-first design
- RunE error handling pattern is strongly preferred over Run
- Viper integration provides robust configuration management
- Shell completions are critical for modern CLI UX
- Testing strategies include unit tests, integration tests, and golden file testing

---

## Table of Contents

1. [Framework Overview](#framework-overview)
2. [Command Structure & Organization](#command-structure--organization)
3. [Flag Management](#flag-management)
4. [Configuration Management](#configuration-management)
5. [Error Handling](#error-handling)
6. [CLI UX Best Practices](#cli-ux-best-practices)
7. [Testing Strategies](#testing-strategies)
8. [Shell Completion](#shell-completion)
9. [Advanced Patterns](#advanced-patterns)
10. [Analysis of Current gristctl Implementation](#analysis-of-current-gristctl-implementation)
11. [Recommendations for gristctl](#recommendations-for-gristctl)
12. [References](#references)

---

## Framework Overview

### What is Cobra?

Cobra is the powerful, battle-tested CLI framework that transforms complex command-line development into simple, elegant Go code. As of 2025, it powers:
- Kubernetes (kubectl)
- Docker CLI
- GitHub CLI (gh)
- Hugo
- 173,000+ other projects

### Core Architecture

Cobra is built on a structure of **commands, arguments & flags**:
- **Commands** represent actions (verbs)
- **Args** are things (nouns)
- **Flags** are modifiers (adjectives)

**Pattern**: `APPNAME VERB NOUN --ADJECTIVE` or `APPNAME COMMAND ARG --FLAG`

Example: `gristctl doc export myDocId --format json`

### Key Features

- Sophisticated command tree architecture with unlimited nesting depth
- Persistent flag inheritance throughout command hierarchy
- Pre/Post run hooks for command lifecycle management
- Automatic help generation and command suggestions
- Built-in shell completion (bash, zsh, fish, PowerShell)
- Seamless Viper integration for configuration management
- POSIX-compliant flag parsing via pflag library

---

## Command Structure & Organization

### Project Structure Patterns

#### Simple Structure (Small Projects)
```
gristctl/
├── main.go              # Entry point
└── cmd/
    ├── root.go          # Root command
    ├── doc.go           # doc command and subcommands
    ├── org.go           # org command and subcommands
    └── workspace.go     # workspace command and subcommands
```

#### Modular Structure (Large Projects - Recommended at Scale)
```
gristctl/
├── main.go
└── internal/
    ├── cmd/
    │   └── root.go
    ├── doc/
    │   └── commands.go  # Returns *cobra.Command
    ├── org/
    │   └── commands.go
    └── workspace/
        └── commands.go
```

### Command Definition Best Practices

**kubectl Pattern**: For every command, create:
1. `NewCmd<CommandName>` function that returns `*cobra.Command`
2. `<CommandName>Config` struct with variables for flags and arguments
3. Three methods on the config struct:
   - `Complete()` - completes struct fields with values
   - `Validate()` - performs validation and returns errors
   - `Run()` - executes the command logic

**Example:**
```go
type DocExportConfig struct {
    DocID  string
    Format string
    Output string
}

func (c *DocExportConfig) Complete(cmd *cobra.Command, args []string) error {
    if len(args) > 0 {
        c.DocID = args[0]
    }
    return nil
}

func (c *DocExportConfig) Validate() error {
    if c.DocID == "" {
        return fmt.Errorf("doc-id is required")
    }
    validFormats := []string{"excel", "grist", "csv"}
    if !contains(validFormats, c.Format) {
        return fmt.Errorf("invalid format: %s (must be one of %v)", c.Format, validFormats)
    }
    return nil
}

func (c *DocExportConfig) Run() error {
    // Actual export logic here
    return exportDoc(c.DocID, c.Format, c.Output)
}

func NewDocExportCmd() *cobra.Command {
    config := &DocExportConfig{}

    cmd := &cobra.Command{
        Use:   "export <doc-id>",
        Short: "Export a Grist document",
        Long: `Export a Grist document to various formats.

Supported formats:
  excel - Export to Excel (.xlsx)
  grist - Export to Grist format
  csv   - Export to CSV`,
        Example: `  # Export to Excel
  gristctl doc export myDocId --format excel

  # Export to specific file
  gristctl doc export myDocId --format grist -o backup.grist`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := config.Complete(cmd, args); err != nil {
                return err
            }
            if err := config.Validate(); err != nil {
                return err
            }
            return config.Run()
        },
    }

    cmd.Flags().StringVarP(&config.Format, "format", "f", "excel", "Export format (excel, grist, csv)")
    cmd.Flags().StringVarP(&config.Output, "output", "o", "", "Output file path")

    return cmd
}
```

### Command Naming Conventions

**CRITICAL**: Use **camelCase** (not snake_case/kebab-case) for command names in Go code, otherwise you will encounter errors.

```go
// CORRECT
var orgListCmd = &cobra.Command{
    Use: "list",
}

// INCORRECT - will cause errors
var org_list_cmd = &cobra.Command{
    Use: "list",
}
```

### Command Hierarchy

Commands should follow logical groupings:

```
gristctl
├── org
│   ├── list
│   ├── get
│   ├── access
│   └── usage
├── workspace (alias: ws)
│   ├── get
│   └── access
├── doc
│   ├── get
│   ├── access
│   ├── webhooks
│   ├── export
│   └── table
├── users
│   └── list
└── create
    └── org
```

---

## Flag Management

### Types of Flags

1. **Local Flags** - Configure a single command
2. **Persistent Flags** - Flow down the tree to all subcommands

```go
// Local flag - only for this command
cmd.Flags().StringVarP(&format, "format", "f", "json", "Output format")

// Persistent flag - available to this command and all children
cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format")
```

### Flag Best Practices

1. **Use shorthands judiciously** - Only for frequently used flags
2. **Provide good defaults** - Make common cases easy
3. **Mark required flags** when appropriate
4. **Validate relationships in PreRunE**

```go
// Mark as required
cmd.MarkFlagRequired("api-key")

// Mutually exclusive flags
cmd.MarkFlagsMutuallyExclusive("json", "yaml", "table")

// Flags that must be used together
cmd.MarkFlagsRequiredTogether("username", "password")

// At least one required
cmd.MarkFlagsOneRequired("config", "api-key")
```

### Flag Groups

Cobra provides three types of flag group relationships:

1. **Required Together** - If any flag in the group is set, all must be set
2. **One Required** - At least one flag from the group must be provided
3. **Mutually Exclusive** - Only one flag from the group can be provided

```go
// Mutually exclusive output formats
cmd.MarkFlagsMutuallyExclusive("json", "yaml", "csv")

// Required together for authentication
cmd.MarkFlagsRequiredTogether("username", "password")

// At least one required
cmd.MarkFlagsOneRequired("org-id", "org-name")
```

### Flag Completion

Register custom completion functions for better UX:

```go
cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"json", "yaml", "csv", "table"}, cobra.ShellCompDirectiveDefault
})

cmd.RegisterFlagCompletionFunc("org-id", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    // Fetch org IDs from API
    orgs := fetchOrgs()
    ids := make([]string, len(orgs))
    for i, org := range orgs {
        ids[i] = org.ID
    }
    return ids, cobra.ShellCompDirectiveDefault
})
```

---

## Configuration Management

### Viper Integration

**Key Principle**: Always treat Viper as the single source of truth for configuration values.

Configuration precedence (highest to lowest):
1. Command-line flags
2. Environment variables
3. Configuration files
4. Defaults

### Basic Integration Pattern

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
    Use: "gristctl",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return initConfig()
    },
}

func init() {
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gristctl)")
    rootCmd.PersistentFlags().String("api-key", "", "Grist API key")
    rootCmd.PersistentFlags().String("url", "https://docs.getgrist.com", "Grist server URL")

    // Bind flags to viper
    viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
    viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))

    // Environment variable binding
    viper.SetEnvPrefix("GRIST")
    viper.AutomaticEnv()
}

func initConfig() error {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            return err
        }
        viper.AddConfigPath(home)
        viper.SetConfigName(".gristctl")
        viper.SetConfigType("ini")
    }

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return err
        }
    }

    return nil
}

// ALWAYS use Viper to retrieve values, NOT cmd.Flag()
func getAPIKey() string {
    return viper.GetString("api-key")  // Correctly resolved value
}
```

### Configuration File Formats

Viper supports multiple formats:
- JSON
- YAML
- TOML
- INI
- HCL
- envfile

Choose based on your target audience:
- **YAML** - Most popular for modern CLIs, good for complex configs
- **INI** - Simple, familiar to sysadmins
- **TOML** - Good balance of readability and features
- **JSON** - Machine-friendly, less human-readable

### Environment Variables

```go
// Automatic environment variable binding
viper.SetEnvPrefix("GRIST")  // Will look for GRIST_* vars
viper.AutomaticEnv()         // Automatically bind env vars

// Manual binding
viper.BindEnv("api.key", "GRIST_API_KEY")
```

### CobraFlags Helper

Consider using the `cobraflags` module to automate binding:

```go
import "github.com/go-extras/cobraflags"

// Automatically binds all flags to viper and environment variables
cobraflags.Init(rootCmd, viper.GetViper())
```

---

## Error Handling

### Use RunE Instead of Run

**CRITICAL**: Prefer `RunE` to `Run`. Returning an error keeps your command logic clean and lets Cobra handle the exit code.

```go
// INCORRECT - using Run
var badCmd = &cobra.Command{
    Use: "bad",
    Run: func(cmd *cobra.Command, args []string) {
        if err := doSomething(); err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)  // Hard to test, bypasses error handling
        }
    },
}

// CORRECT - using RunE
var goodCmd = &cobra.Command{
    Use: "good",
    RunE: func(cmd *cobra.Command, args []string) error {
        if err := doSomething(); err != nil {
            return err  // Cobra handles exit code, testable
        }
        return nil
    },
}
```

### Error Handling Best Practices

#### 1. Control Usage Display

```go
var cmd = &cobra.Command{
    Use: "export",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Suppress usage on runtime errors
        cmd.SilenceUsage = true

        if err := validateInput(); err != nil {
            return err  // Won't show usage
        }

        return exportDoc()
    },
}
```

#### 2. Custom Error Types

```go
type ErrorLevel int

const (
    ErrorLevelUser ErrorLevel = iota
    ErrorLevelSystem
    ErrorLevelInternal
)

type CLIError struct {
    Level   ErrorLevel
    Message string
    Err     error
}

func (e *CLIError) Error() string {
    return e.Message
}

func (e *CLIError) Unwrap() error {
    return e.Err
}

// Usage
func validateDocID(id string) error {
    if id == "" {
        return &CLIError{
            Level:   ErrorLevelUser,
            Message: "doc-id is required",
        }
    }
    return nil
}
```

#### 3. Panic Recovery

```go
var riskyCmd = &cobra.Command{
    Use: "risky",
    RunE: func(cmd *cobra.Command, args []string) (err error) {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("command panicked: %v\n%s", r, debug.Stack())
            }
        }()

        return riskyOperation()
    },
}
```

#### 4. Error Wrapping

```go
func exportDoc(docID string) error {
    doc, err := fetchDoc(docID)
    if err != nil {
        return fmt.Errorf("failed to fetch document %s: %w", docID, err)
    }

    if err := writeFile(doc); err != nil {
        return fmt.Errorf("failed to write export file: %w", err)
    }

    return nil
}
```

### PreRunE for Validation

Use `PreRunE` for validation that should happen before the main command:

```go
var exportCmd = &cobra.Command{
    Use: "export <doc-id>",
    PreRunE: func(cmd *cobra.Command, args []string) error {
        // Validate format flag
        format, _ := cmd.Flags().GetString("format")
        validFormats := []string{"json", "yaml", "csv"}
        if !contains(validFormats, format) {
            return fmt.Errorf("invalid format: %s (must be one of %v)", format, validFormats)
        }
        return nil
    },
    RunE: func(cmd *cobra.Command, args []string) error {
        // Validation already done, just execute
        return doExport(args[0])
    },
}
```

---

## CLI UX Best Practices

### Modern CLI Design Principles (CLIG)

The [Command Line Interface Guidelines](https://clig.dev/) (CLIG) provides foundational principles:

1. **Human-First Design** - Design for humans first, machines second
2. **Composability** - Small, simple programs with clean interfaces
3. **Empathy** - Consider the user's context and experience
4. **Consistency** - Follow established patterns and conventions

### Help Text Formatting

#### Basic Structure

```go
var rootCmd = &cobra.Command{
    Use:   "gristctl [command]",
    Short: "Gristctl - The meaty CLI for Grist",
    Long: `Gristctl is a command-line tool for interacting with Grist.
It provides commands to manage organizations, workspaces, documents, and more.

Run with no arguments to launch the interactive TUI.`,
    Example: `  # Launch TUI
  gristctl

  # List organizations
  gristctl org list

  # Export a document
  gristctl doc export myDocId --format excel`,
}
```

#### Enhanced Help with Glamour

Use Charm's [Glamour](https://github.com/charmbracelet/glamour) for styled help text:

```go
import "github.com/charmbracelet/glamour"

func renderHelp(text string) string {
    r, _ := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
    )
    out, _ := r.Render(text)
    return out
}

cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    helpText := cmd.Long
    fmt.Println(renderHelp(helpText))
})
```

#### Command Grouping

Group related commands in help output:

```go
rootCmd.AddGroup(&cobra.Group{
    ID:    "resource",
    Title: "Resource Management Commands:",
})

orgCmd.GroupID = "resource"
workspaceCmd.GroupID = "resource"
docCmd.GroupID = "resource"
```

### Output Formatting

#### Multi-Format Support

```go
type OutputFormat string

const (
    OutputTable OutputFormat = "table"
    OutputJSON  OutputFormat = "json"
    OutputYAML  OutputFormat = "yaml"
    OutputCSV   OutputFormat = "csv"
)

func display(data interface{}, format OutputFormat) error {
    switch format {
    case OutputTable:
        return displayTable(data)
    case OutputJSON:
        return displayJSON(data)
    case OutputYAML:
        return displayYAML(data)
    case OutputCSV:
        return displayCSV(data)
    default:
        return fmt.Errorf("unsupported format: %s", format)
    }
}
```

#### Progress Indicators

Modern CLIs should provide feedback for long-running operations:

**Three Popular Patterns:**
1. **Spinner** - For unknown duration operations
2. **X of Y** - For countable operations (processing 5 of 10 files)
3. **Progress Bar** - For operations with known total

Never leave users staring at a blinking cursor on a dark terminal screen.

```go
import "github.com/schollz/progressbar/v3"

bar := progressbar.Default(100)
for i := 0; i < 100; i++ {
    bar.Add(1)
    time.Sleep(40 * time.Millisecond)
}
```

#### Silent Mode

Consider adding a `--quiet` flag for silent mode:

```go
rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress output")

func log(msg string) {
    if !quiet {
        fmt.Println(msg)
    }
}
```

### Color and Styling

Use the [Charm Lipgloss](https://github.com/charmbracelet/lipgloss) library for consistent styling:

```go
import "github.com/charmbracelet/lipgloss"

var (
    errorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("9")).  // Red
        Bold(true)

    successStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("10"))  // Green
)

fmt.Println(errorStyle.Render("Error: Document not found"))
fmt.Println(successStyle.Render("Export completed successfully"))
```

### Error Messages

Good error messages should:
1. **Be clear** - Explain what went wrong
2. **Be actionable** - Suggest how to fix it
3. **Provide context** - Include relevant details

```go
// BAD
return fmt.Errorf("error")

// BETTER
return fmt.Errorf("failed to connect to Grist server")

// BEST
return fmt.Errorf("failed to connect to Grist server at %s: %w\n\nTry:\n  - Check your internet connection\n  - Verify the server URL with 'gristctl config get url'\n  - Ensure your API key is valid", serverURL, err)
```

---

## Testing Strategies

### Unit Testing Commands

#### Basic Command Test

```go
func TestOrgListCmd(t *testing.T) {
    // Setup
    cmd := NewOrgListCmd()
    b := bytes.NewBufferString("")
    cmd.SetOut(b)
    cmd.SetErr(b)
    cmd.SetArgs([]string{})

    // Execute
    err := cmd.Execute()

    // Assert
    assert.NoError(t, err)
    assert.Contains(t, b.String(), "Organization")
}
```

#### Testing with Flags

```go
func TestDocExportCmd(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
    }{
        {
            name:    "valid export",
            args:    []string{"myDocId", "--format", "excel"},
            wantErr: false,
        },
        {
            name:    "invalid format",
            args:    []string{"myDocId", "--format", "invalid"},
            wantErr: true,
        },
        {
            name:    "missing doc-id",
            args:    []string{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewDocExportCmd()
            cmd.SetArgs(tt.args)
            err := cmd.Execute()

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Golden File Testing

Golden files contain the expected output of a test. When tests run, they compare actual output to the golden file.

```go
import "github.com/sebdah/goldie/v2"

func TestOrgListOutput(t *testing.T) {
    g := goldie.New(t)

    cmd := NewOrgListCmd()
    b := bytes.NewBufferString("")
    cmd.SetOut(b)
    cmd.SetArgs([]string{})

    err := cmd.Execute()
    assert.NoError(t, err)

    // Compare output to golden file
    g.Assert(t, "org_list", b.Bytes())
}

// Update golden files with:
// go test -update ./...
```

**Benefits:**
- Excellent for validating exact output format
- Easy to code review (changes show in git diff)
- Great for testing error messages and help text
- Simple to update when output changes

### Integration Testing

```go
func TestEndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup test server
    server := setupTestGristServer(t)
    defer server.Close()

    // Set test configuration
    os.Setenv("GRIST_URL", server.URL)
    os.Setenv("GRIST_API_KEY", "test-key")

    // Execute command
    cmd := NewOrgListCmd()
    err := cmd.Execute()

    assert.NoError(t, err)
}
```

### Testing Best Practices

1. **Extract logic from commands** - Make it testable
2. **Use table-driven tests** - Cover multiple scenarios
3. **Test error cases** - Not just happy paths
4. **Mock external dependencies** - Use interfaces
5. **Test flag validation** - Ensure proper validation
6. **Use golden files for output** - Validate exact formatting

---

## Shell Completion

### Overview

Cobra provides built-in support for shell completion across bash, zsh, fish, and PowerShell.

### Generating Completion Files

Cobra automatically provides a completion command:

```bash
# Generate completion file
gristctl completion bash > /etc/bash_completion.d/gristctl
gristctl completion zsh > /usr/local/share/zsh/site-functions/_gristctl
gristctl completion fish > ~/.config/fish/completions/gristctl.fish
gristctl completion powershell > gristctl.ps1
```

### Custom Completions

**IMPORTANT**: Use `RegisterFlagCompletionFunc()` and `ValidArgsFunction` (not legacy bash-only methods).

#### Flag Completion

```go
cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"json", "yaml", "csv", "table"}, cobra.ShellCompDirectiveDefault
})
```

#### Argument Completion

```go
var docGetCmd = &cobra.Command{
    Use:   "get <doc-id>",
    Short: "Get document details",
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        if len(args) != 0 {
            return nil, cobra.ShellCompDirectiveNoFileComp
        }
        // Fetch and return available doc IDs
        docs := fetchDocs()
        ids := make([]string, len(docs))
        for i, doc := range docs {
            ids[i] = fmt.Sprintf("%s\t%s", doc.ID, doc.Name)  // ID + description
        }
        return ids, cobra.ShellCompDirectiveNoFileComp
    },
    RunE: func(cmd *cobra.Command, args []string) error {
        return getDoc(args[0])
    },
}
```

### Shell Completion Directives

- `ShellCompDirectiveDefault` - Default behavior
- `ShellCompDirectiveNoFileComp` - Don't suggest files
- `ShellCompDirectiveNoSpace` - Don't add space after completion
- `ShellCompDirectiveError` - An error occurred

### Debugging Completions

```bash
# Call the hidden completion command directly
gristctl __complete doc get har
```

---

## Advanced Patterns

### Lifecycle Hooks

Cobra provides hooks that execute before and after commands:

**Execution Order:**
1. `PersistentPreRun()`
2. `PreRun()`
3. `Run()`
4. `PostRun()`
5. `PersistentPostRun()`

**Important Notes:**
- Only executed if `Run()` is declared
- `PersistentPre/PostRun` are inherited by children
- `Pre/PostRun` are NOT inherited
- Use the `E` variants (`PreRunE`, `PostRunE`) to return errors

```go
var rootCmd = &cobra.Command{
    Use: "gristctl",
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Initialize config, logging, etc.
        return initializeApp()
    },
    PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
        // Cleanup, metrics, etc.
        return cleanup()
    },
}

var exportCmd = &cobra.Command{
    Use: "export",
    PreRunE: func(cmd *cobra.Command, args []string) error {
        // Command-specific validation
        return validateExportArgs()
    },
    RunE: func(cmd *cobra.Command, args []string) error {
        return doExport()
    },
    PostRunE: func(cmd *cobra.Command, args []string) error {
        // Command-specific cleanup
        return cleanupTempFiles()
    },
}
```

### Aliases

```go
var workspaceCmd = &cobra.Command{
    Use:     "workspace",
    Aliases: []string{"ws", "work"},  // Allow multiple aliases
    Short:   "Manage workspaces",
}

// All of these work:
// gristctl workspace list
// gristctl ws list
// gristctl work list
```

### Hidden Commands

```go
var debugCmd = &cobra.Command{
    Use:    "debug",
    Hidden: true,  // Won't show in help
    RunE: func(cmd *cobra.Command, args []string) error {
        return debugInfo()
    },
}
```

### Deprecated Commands

```go
var oldCmd = &cobra.Command{
    Use:        "old-command",
    Deprecated: "use 'new-command' instead",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Still works but shows deprecation warning
        return doThing()
    },
}
```

### Plugin Architecture

While Cobra doesn't have built-in plugin support, several patterns are used:

#### External Binary Pattern

```go
func executePlugin(pluginName string, args []string) error {
    // Look for gristctl-<plugin> in PATH
    pluginPath, err := exec.LookPath(fmt.Sprintf("gristctl-%s", pluginName))
    if err != nil {
        return fmt.Errorf("plugin %s not found", pluginName)
    }

    cmd := exec.Command(pluginPath, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    return cmd.Run()
}

// kubectl-style plugin discovery
var rootCmd = &cobra.Command{
    Use: "gristctl",
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) > 0 {
            // Try to execute as plugin
            if err := executePlugin(args[0], args[1:]); err == nil {
                return nil
            }
        }
        return cmd.Help()
    },
}
```

#### Dynamic Command Registration

```go
type PluginCommands interface {
    Commands() []*cobra.Command
}

func loadPlugins(rootCmd *cobra.Command) error {
    plugins := discoverPlugins()
    for _, plugin := range plugins {
        for _, cmd := range plugin.Commands() {
            rootCmd.AddCommand(cmd)
        }
    }
    return nil
}
```

### Context Support

```go
import "context"

func executeWithContext(ctx context.Context) error {
    cmd := NewRootCmd()

    // Pass context through
    go func() {
        <-ctx.Done()
        // Handle cancellation
    }()

    return cmd.ExecuteContext(ctx)
}

// In main.go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        cancel()
    }()

    if err := executeWithContext(ctx); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

---

## Analysis of Current gristctl Implementation

### Current Structure

```
grist-ctl/
├── main.go              # Entry point
├── cmd/
│   ├── root.go          # Root command with TUI launcher
│   ├── org.go           # Organization commands
│   ├── workspace.go     # Workspace commands (with alias 'ws')
│   ├── doc.go           # Document commands
│   ├── users.go         # User management
│   ├── create.go        # Resource creation
│   ├── delete.go        # Resource deletion
│   ├── move.go          # Resource moving
│   ├── purge.go         # Resource purging
│   ├── import.go        # Import operations
│   ├── config.go        # Configuration
│   ├── version.go       # Version display
│   └── mcp.go           # MCP server
├── tui/                 # Bubble Tea TUI
├── mcp/                 # MCP server implementation
├── gristapi/            # API client
├── gristtools/          # CLI display helpers
└── common/              # Configuration and utilities
```

### Strengths

1. **Good Organization** - Commands organized in separate files
2. **TUI Integration** - Launches TUI by default (innovative UX)
3. **Aliases** - Workspace command has 'ws' alias
4. **Output Formats** - Supports table and JSON output via persistent flags
5. **Clean Command Structure** - Follows Cobra patterns

### Areas for Improvement

#### 1. Error Handling

**Current:**
```go
var orgGetCmd = &cobra.Command{
    Use:  "get <org-id>",
    Args: cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        gristtools.DisplayOrg(args[0])
    },
}
```

**Issues:**
- Uses `Run` instead of `RunE`
- No error handling visible
- Errors likely handled inside `DisplayOrg()` with `os.Exit(1)`
- Makes testing difficult

**Recommended:**
```go
var orgGetCmd = &cobra.Command{
    Use:   "get <org-id>",
    Short: "Get organization details",
    Long:  "Display detailed information about a specific organization",
    Example: `  # Get org details
  gristctl org get myOrgId

  # Get org details as JSON
  gristctl org get myOrgId --json`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        cmd.SilenceUsage = true
        return gristtools.DisplayOrg(args[0])
    },
}
```

#### 2. Flag Validation

**Current:**
```go
// workspace.go
var workspaceGetCmd = &cobra.Command{
    Use:  "get <workspace-id>",
    Args: cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        wsID, err := strconv.Atoi(args[0])
        if err != nil {
            fmt.Fprintf(os.Stderr, "Invalid workspace ID: %s\n", args[0])
            os.Exit(1)
        }
        gristtools.DisplayWorkspace(wsID)
    },
}
```

**Issues:**
- Validation mixed with execution
- Direct `os.Exit()` prevents testing
- No use of PreRunE

**Recommended:**
```go
type WorkspaceGetConfig struct {
    ID int
}

func (c *WorkspaceGetConfig) Complete(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("workspace-id is required")
    }
    id, err := strconv.Atoi(args[0])
    if err != nil {
        return fmt.Errorf("invalid workspace ID %q: must be a number", args[0])
    }
    c.ID = id
    return nil
}

func (c *WorkspaceGetConfig) Run() error {
    return gristtools.DisplayWorkspace(c.ID)
}

func NewWorkspaceGetCmd() *cobra.Command {
    config := &WorkspaceGetConfig{}

    cmd := &cobra.Command{
        Use:   "get <workspace-id>",
        Short: "Get workspace details",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            cmd.SilenceUsage = true
            if err := config.Complete(args); err != nil {
                return err
            }
            return config.Run()
        },
    }

    return cmd
}
```

#### 3. Help Text and Examples

**Current:**
```go
var docExportCmd = &cobra.Command{
    Use:       "export <doc-id> <format>",
    Short:     "Export document",
    Long:      `Export document in the specified format: excel or grist`,
    Args:      cobra.ExactArgs(2),
    ValidArgs: []string{"excel", "grist"},
    // ...
}
```

**Issues:**
- Minimal help text
- No examples
- Format as positional arg instead of flag

**Recommended:**
```go
var docExportCmd = &cobra.Command{
    Use:   "export <doc-id>",
    Short: "Export a Grist document",
    Long: `Export a Grist document to various formats.

Supported formats:
  excel - Export to Microsoft Excel format (.xlsx)
  grist - Export to Grist's native format
  csv   - Export to CSV (requires --table flag)`,
    Example: `  # Export document to Excel
  gristctl doc export myDocId --format excel

  # Export to Grist format with custom filename
  gristctl doc export myDocId --format grist -o backup.grist

  # Export specific table to CSV
  gristctl doc export myDocId --format csv --table Users`,
    Args: cobra.ExactArgs(1),
    RunE: exportDocFunc,
}

cmd.Flags().StringVarP(&format, "format", "f", "excel", "Export format (excel, grist, csv)")
cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
cmd.Flags().StringVar(&table, "table", "", "Table name (required for CSV format)")
cmd.RegisterFlagCompletionFunc("format", completeFormats)
```

#### 4. Configuration Management

**Current:**
```go
// root.go
var (
    outputFormat string
    jsonOutput   bool
)

rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")
rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON (shorthand for -o json)")
```

**Issues:**
- No Viper integration visible
- Configuration loaded in `common/` package, not integrated with flags
- Manual flag handling in `PersistentPreRun`

**Recommended:**
```go
// root.go
import (
    "github.com/spf13/viper"
)

func init() {
    cobra.OnInitialize(initConfig)

    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gristctl)")
    rootCmd.PersistentFlags().String("api-key", "", "Grist API key")
    rootCmd.PersistentFlags().String("url", "", "Grist server URL")
    rootCmd.PersistentFlags().StringP("output", "o", "table", "Output format (table, json, yaml, csv)")

    // Bind to viper
    viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
    viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
    viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

    // Environment variables
    viper.SetEnvPrefix("GRIST")
    viper.AutomaticEnv()
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        viper.AddConfigPath(home)
        viper.SetConfigName(".gristctl")
        viper.SetConfigType("ini")
    }

    viper.ReadInConfig()
}
```

#### 5. Testing

**Current:** No visible test files in cmd/ directory

**Recommended:** Add comprehensive tests:

```go
// cmd/org_test.go
package cmd

import (
    "bytes"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestOrgListCmd(t *testing.T) {
    // Test implementation
}

func TestOrgGetCmd(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
    }{
        {"valid org id", []string{"myOrg"}, false},
        {"missing org id", []string{}, true},
        {"too many args", []string{"org1", "org2"}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewOrgGetCmd()
            cmd.SetArgs(tt.args)
            err := cmd.Execute()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 6. Shell Completions

**Current:** No custom completion functions

**Recommended:** Add completions for better UX:

```go
// Complete org IDs
orgGetCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    if len(args) != 0 {
        return nil, cobra.ShellCompDirectiveNoFileComp
    }

    client := gristapi.NewClient()
    orgs, err := client.ListOrgs()
    if err != nil {
        return nil, cobra.ShellCompDirectiveError
    }

    suggestions := make([]string, len(orgs))
    for i, org := range orgs {
        suggestions[i] = fmt.Sprintf("%s\t%s", org.ID, org.Name)
    }
    return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// Complete formats
docExportCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{
        "excel\tMicrosoft Excel format",
        "grist\tGrist native format",
        "csv\tComma-separated values",
    }, cobra.ShellCompDirectiveDefault
})
```

---

## Recommendations for gristctl

### High Priority

1. **Convert Run to RunE** across all commands
   - Enables proper error handling
   - Makes testing easier
   - Follows modern best practices

2. **Add Comprehensive Help Text**
   - Add `Example` field to all commands
   - Expand `Long` descriptions
   - Document all flags

3. **Implement Shell Completions**
   - Add ValidArgsFunction for resource IDs
   - Add flag completion for formats, types, etc.
   - Significantly improves UX

4. **Add Tests**
   - Unit tests for all commands
   - Table-driven tests for validation
   - Golden file tests for output formats

5. **Integrate Viper Properly**
   - Bind all flags to Viper
   - Use Viper as single source of truth
   - Implement configuration precedence

### Medium Priority

6. **Implement kubectl Config Pattern**
   - Create Config structs for complex commands
   - Implement Complete/Validate/Run pattern
   - Improves testability and maintainability

7. **Add PreRunE Validation**
   - Move validation logic to PreRunE
   - Separate validation from execution
   - Clearer error messages

8. **Enhance Error Messages**
   - More descriptive errors
   - Suggest solutions
   - Include relevant context

9. **Add Progress Indicators**
   - Spinners for API calls
   - Progress bars for exports
   - Better user feedback

10. **Command Grouping**
    - Group related commands in help
    - Improves discoverability
    - Cleaner help output

### Low Priority

11. **Support More Output Formats**
    - Add YAML output
    - Add CSV for list commands
    - Make format handling consistent

12. **Add Silent Mode**
    - `--quiet` flag for scripts
    - Suppress progress output
    - Only show results

13. **Implement Plugin Architecture**
    - Allow external extensions
    - kubectl-style plugin discovery
    - Enable community contributions

14. **Add Context Support**
    - Graceful shutdown on signals
    - Timeout for long operations
    - Better resource cleanup

### Example Refactor: org.go

**Before:**
```go
var orgGetCmd = &cobra.Command{
    Use:   "get <org-id>",
    Short: "Get organization details",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        gristtools.DisplayOrg(args[0])
    },
}
```

**After:**
```go
type OrgGetConfig struct {
    OrgID string
}

func (c *OrgGetConfig) Complete(args []string) error {
    if len(args) > 0 {
        c.OrgID = args[0]
    }
    return nil
}

func (c *OrgGetConfig) Validate() error {
    if c.OrgID == "" {
        return fmt.Errorf("org-id is required")
    }
    return nil
}

func (c *OrgGetConfig) Run() error {
    return gristtools.DisplayOrg(c.OrgID)
}

func NewOrgGetCmd() *cobra.Command {
    config := &OrgGetConfig{}

    cmd := &cobra.Command{
        Use:   "get <org-id>",
        Short: "Get organization details",
        Long:  "Display detailed information about a specific Grist organization",
        Example: `  # Get organization details
  gristctl org get myOrgId

  # Get organization as JSON
  gristctl org get myOrgId --json

  # Get organization as YAML
  gristctl org get myOrgId -o yaml`,
        Args: cobra.ExactArgs(1),
        ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            if len(args) != 0 {
                return nil, cobra.ShellCompDirectiveNoFileComp
            }
            return completeOrgIDs(toComplete)
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            cmd.SilenceUsage = true
            if err := config.Complete(args); err != nil {
                return err
            }
            if err := config.Validate(); err != nil {
                return err
            }
            return config.Run()
        },
    }

    return cmd
}

func completeOrgIDs(toComplete string) ([]string, cobra.ShellCompDirective) {
    client := gristapi.NewClient()
    orgs, err := client.ListOrgs()
    if err != nil {
        return nil, cobra.ShellCompDirectiveError
    }

    suggestions := make([]string, len(orgs))
    for i, org := range orgs {
        suggestions[i] = fmt.Sprintf("%s\t%s", org.ID, org.Name)
    }
    return suggestions, cobra.ShellCompDirectiveNoFileComp
}
```

---

## Code Quality Checklist

Use this checklist when implementing or reviewing Cobra commands:

### Command Definition
- [ ] Uses `RunE` instead of `Run`
- [ ] Has clear `Use`, `Short`, and `Long` descriptions
- [ ] Includes `Example` field with practical examples
- [ ] Uses appropriate `Args` validator (ExactArgs, MinimumNArgs, etc.)
- [ ] Has `ValidArgsFunction` for dynamic completions

### Flag Management
- [ ] Flags bound to config struct (not package variables)
- [ ] Required flags marked with `MarkFlagRequired()`
- [ ] Mutually exclusive flags grouped with `MarkFlagsMutuallyExclusive()`
- [ ] Flags have completion functions via `RegisterFlagCompletionFunc()`
- [ ] Persistent vs. local flags used appropriately

### Error Handling
- [ ] Returns errors (doesn't call `os.Exit()`)
- [ ] Sets `cmd.SilenceUsage = true` for runtime errors
- [ ] Uses error wrapping for context (`fmt.Errorf("...: %w", err)`)
- [ ] Error messages are clear and actionable
- [ ] PreRunE used for validation

### Testing
- [ ] Has unit tests
- [ ] Tests both success and error cases
- [ ] Uses table-driven tests for multiple scenarios
- [ ] Tests flag validation
- [ ] Has integration tests for complex commands

### Documentation
- [ ] Help text is clear and complete
- [ ] Examples are up-to-date
- [ ] All flags are documented
- [ ] Command purpose is obvious

### UX
- [ ] Provides progress feedback for long operations
- [ ] Supports multiple output formats (table, JSON, etc.)
- [ ] Has shell completions
- [ ] Error messages suggest solutions
- [ ] Follows consistent naming conventions

---

## References

### Official Documentation
- [Cobra Official Site](https://cobra.dev/)
- [Cobra GitHub Repository](https://github.com/spf13/cobra)
- [Cobra User Guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md)
- [Cobra Enterprise Guide](https://cobra.dev/docs/explanations/enterprise-guide/)

### Cobra + Viper Integration
- [Sting of the Viper: Getting Cobra and Viper to Work Together](https://carolynvanslyck.com/blog/2020/08/sting-of-the-viper/)
- [Building a 12-Factor App with Viper Integration](https://cobra.dev/docs/tutorials/12-factor-app/)
- [Building CLI Apps in Go with Cobra & Viper](https://www.glukhov.org/post/2025/11/go-cli-applications-with-cobra-and-viper/)

### CLI Design Principles
- [Command Line Interface Guidelines (CLIG)](https://clig.dev/)
- [CLI Guidelines GitHub](https://github.com/cli-guidelines/cli-guidelines)
- [10 Design Principles for Delightful CLIs - Atlassian](https://www.atlassian.com/blog/it-teams/10-design-principles-for-delightful-clis)

### Testing
- [Testing Cobra CLI Commands in GoLang](https://nayaktapan37.medium.com/testing-cobra-commands-in-golang-ca1fe4ad6657)
- [Testing a Cobra CLI in Go - BradCypert.com](https://www.bradcypert.com/testing-a-cobra-cli-in-go/)
- [Testing with Golden Files in Go](https://medium.com/soon-london/testing-with-golden-files-in-go-7fccc71c43d3)
- [Golden File Testing - Ieftimov](https://ieftimov.com/posts/testing-in-go-golden-files/)

### Error Handling
- [Error Handling Strategies - Mastering Cobra](https://app.studyraid.com/en/read/11421/357754/error-handling-strategies)
- [Error Handling in Cobra - JetBrains Guide](https://www.jetbrains.com/guide/go/tutorials/cli-apps-go-cobra/error_handling/)
- [Taming Cobras: Making the Most of Cobra CLIs](https://gopheradvent.com/calendar/2022/taming-cobras-making-most-of-cobra-clis/)

### Shell Completions
- [Shell Completion Guide](https://cobra.dev/docs/how-to-guides/shell-completion/)
- [Auto-Completing CLI Arguments in Golang with Cobra](https://www.raftt.io/post/auto-completing-cli-arguments-in-golang-with-cobra.html)
- [Shell Completions with Go Cobra Library](https://blog.chmouel.com/posts/cobra-completions/)

### UX & Output
- [CLI UX Best Practices: Progress Displays - Evil Martians](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays)
- [Writing Better Cobra CLI Help Messages with Glamour](https://dev.to/lukehagar/writing-better-cobra-cli-help-messages-with-glamour-1525)
- [API Inspector CLI Example](https://cobra.dev/docs/examples/03-api-inspector/)

### Real-World Examples
- [kubectl Command Structure](https://github.com/kubernetes/kubernetes/commit/426ef9335865ebef43f682da90796bd8bf976637)
- [GitHub CLI (gh)](https://github.com/cli/cli)
- [Hugo](https://github.com/gohugoio/hugo)
- [Docker CLI](https://github.com/docker/cli)

### Tutorials & Guides
- [Build CLI Tools with Cobra in Go: 2025 Developer Guide](https://codezup.com/create-cli-cobra-go-guide/)
- [How to Create CLI Applications in Go using Cobra and Viper](https://www.faizanbashir.me/how-create-cli-applications-in-golang-using-cobra-and-viper)
- [Mastering Cobra: Building Professional Command-Line Applications in Go](https://app.studyraid.com/en/read/11421/357739/cobra-architecture-and-design-principles)

### Libraries & Tools
- [Charm Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Charm Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Goldie](https://github.com/sebdah/goldie) - Golden file testing
- [CobraFlags](https://github.com/go-extras/cobraflags) - Automated flag binding

---

## Appendix: Quick Reference

### Command Template

```go
type MyCommandConfig struct {
    RequiredArg string
    OptionalFlag string
}

func (c *MyCommandConfig) Complete(cmd *cobra.Command, args []string) error {
    if len(args) > 0 {
        c.RequiredArg = args[0]
    }
    return nil
}

func (c *MyCommandConfig) Validate() error {
    if c.RequiredArg == "" {
        return fmt.Errorf("required-arg is required")
    }
    return nil
}

func (c *MyCommandConfig) Run() error {
    // Command logic here
    return nil
}

func NewMyCommand() *cobra.Command {
    config := &MyCommandConfig{}

    cmd := &cobra.Command{
        Use:   "mycommand <required-arg>",
        Short: "Brief description",
        Long:  "Longer description with details",
        Example: `  # Example 1
  app mycommand value1

  # Example 2
  app mycommand value2 --flag option`,
        Args: cobra.ExactArgs(1),
        ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            // Return dynamic completions
            return []string{"option1", "option2"}, cobra.ShellCompDirectiveDefault
        },
        PreRunE: func(cmd *cobra.Command, args []string) error {
            return config.Complete(cmd, args)
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            cmd.SilenceUsage = true
            if err := config.Validate(); err != nil {
                return err
            }
            return config.Run()
        },
    }

    cmd.Flags().StringVarP(&config.OptionalFlag, "flag", "f", "default", "Flag description")
    cmd.RegisterFlagCompletionFunc("flag", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        return []string{"opt1", "opt2"}, cobra.ShellCompDirectiveDefault
    })

    return cmd
}
```

### Test Template

```go
func TestMyCommand(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        flags   map[string]string
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid input",
            args:    []string{"value1"},
            flags:   map[string]string{"flag": "option"},
            wantErr: false,
        },
        {
            name:    "missing required arg",
            args:    []string{},
            wantErr: true,
            errMsg:  "required-arg is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewMyCommand()
            b := bytes.NewBufferString("")
            cmd.SetOut(b)
            cmd.SetErr(b)
            cmd.SetArgs(tt.args)

            for key, val := range tt.flags {
                cmd.Flags().Set(key, val)
            }

            err := cmd.Execute()

            if tt.wantErr {
                assert.Error(t, err)
                if tt.errMsg != "" {
                    assert.Contains(t, err.Error(), tt.errMsg)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

**Document Version**: 1.0
**Date**: 2025-12-30
**Author**: Research based on Cobra 2025 best practices
**Project**: gristctl CLI Enhancement Research
