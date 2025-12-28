# Analysis of Production Go MCP Servers

Research conducted December 2025 analyzing well-implemented MCP servers in Go.

## Executive Summary

This document analyzes major Go MCP server implementations to extract best practices and architectural patterns.

---

## Production Servers Analyzed

### 1. containers/kubernetes-mcp-server

**Repository:** https://github.com/containers/kubernetes-mcp-server
**Stars:** 913
**License:** Apache-2.0

#### What They Do Well

- **Native Go implementation** - Direct Kubernetes API interaction, no CLI wrappers
- **Modular toolset** - Pluggable feature groups (pods, deployments, services)
- **Multiple distributions** - Binary, npm, pip, Docker, Helm chart
- **TOML configuration** - Drop-in directory support
- **Dynamic reload** - SIGHUP triggers config reload
- **Read-only mode** - Production safety flag
- **Comprehensive tests** - Full test suite

#### Architecture
```
cmd/kubernetes-mcp-server/    # CLI entry point
pkg/                          # Public packages
internal/                     # Private implementations
charts/                       # Helm chart
```

---

### 2. grafana/mcp-grafana

**Repository:** https://github.com/grafana/mcp-grafana
**Stars:** 2,000+
**License:** Apache-2.0

#### What They Do Well

- **Tool categorization** - Enable/disable tool groups
- **Read-only mode** - `--disable-write` flag
- **Proxy pattern** - Datasource query proxying
- **RBAC scoping** - Granular permission patterns
- **TLS support** - Secure connections
- **Context optimization** - JSONPath extraction for targeted data

#### Production Features
- Service account token authentication
- 17+ write operations can be disabled
- Tool categories: search, datasource, prometheus, loki, alerting

---

### 3. strowk/mcp-k8s-go

**Repository:** https://github.com/strowk/mcp-k8s-go
**Stars:** 367
**License:** MIT

#### What They Do Well

- **Multi-arch Docker** - linux/amd64, linux/arm64
- **Security features** - Read-only, context filtering, secret masking
- **Multiple installs** - npm, binaries, Docker, Smithery
- **Go Report Card** - Code quality integration
- **CI/CD** - GitHub Actions automation

---

## Best Patterns Observed

### 1. Functional Options Pattern

All major libraries use functional options:

```go
server := NewMCPServer("name", "version",
    WithLogging(),
    WithRecovery(),
    WithToolCapabilities(true),
)
```

### 2. Type-Safe Handlers with Generics

```go
type Args struct {
    Query string `json:"query" jsonschema:"required"`
}

mcp.AddTool[Args, Result](server, tool, handler)
```

### 3. Automatic JSON Schema Generation

```go
type Input struct {
    Count int    `json:"count" jsonschema:"minimum=1,maximum=100"`
    Name  string `json:"name" jsonschema:"required,minLength=1"`
}
```

### 4. Lifecycle Hooks

```go
hooks := &server.Hooks{
    BeforeAny: func(ctx context.Context, id any, method mcp.MCPMethod, msg any) {
        log.Printf("Request: [%s] %v", method, id)
    },
    AfterAny: func(ctx context.Context, id any, method mcp.MCPMethod, msg any, err error) {
        log.Printf("Response: [%s] %v (err=%v)", method, id, err)
    },
    OnRegisterSession: func(ctx context.Context, session *server.Session) {
        log.Printf("Session connected: %s", session.ID())
    },
}
```

### 5. Middleware Chain

```go
server := NewMCPServer("name", "version",
    WithToolHandlerMiddleware(loggingMiddleware),
    WithToolHandlerMiddleware(authMiddleware),
    WithRecovery(),
)
```

### 6. Transport Abstraction

All support multiple transports:
- **stdio** - CLI and desktop
- **HTTP/SSE** - Web clients
- **Streamable HTTP** - Modern streaming
- **gRPC** - High performance (some)

### 7. Production Safety Features

- Read-only mode flags
- Context filtering for multi-tenancy
- Secret masking by default
- RBAC permission scoping
- TLS configuration

### 8. Clean Package Structure

```
cmd/           # CLI entry points
pkg/           # Public packages
internal/      # Private implementation
examples/      # Usage demonstrations
tests/         # Test suites
docs/          # Documentation
```

### 9. Consistent Error Types

```go
var (
    ErrResourceNotFound = errors.New("resource not found")
    ErrToolNotFound     = errors.New("tool not found")
    ErrUnsupported      = errors.New("not supported")
)
```

---

## Anti-Patterns to Avoid

### 1. CRUD-style Tool Design
```go
// BAD: Generic, unclear
tool := mcp.NewTool("create_record", ...)
tool := mcp.NewTool("update_row", ...)

// GOOD: Domain-aware, specific
tool := mcp.NewTool("submit_expense_report", ...)
tool := mcp.NewTool("schedule_meeting", ...)
```

### 2. Hardcoded Secrets
```go
// BAD
const apiKey = "sk-abc123..."

// GOOD
apiKey := os.Getenv("API_KEY")
```

### 3. Missing Panic Recovery
```go
// BAD: No recovery
server := NewMCPServer("name", "1.0.0")

// GOOD: With recovery
server := NewMCPServer("name", "1.0.0",
    WithRecovery(),
)
```

### 4. Overly Permissive Defaults
```go
// BAD: Writes enabled by default
server := NewMCPServer(...)

// GOOD: Opt-in for writes
if !config.ReadOnly {
    registerWriteTools(server)
}
```

---

## Recommended Project Structure for grist-ctl

```
mcp/
├── server.go           # Server creation and registration
├── handlers.go         # Handler implementations
├── tools.go            # Tool definitions
├── config.go           # Configuration management
├── logging.go          # Structured logging
├── middleware.go       # Rate limiting, validation
├── server_test.go      # Unit tests
├── integration_test.go # Integration tests
├── fuzz_test.go        # Fuzz tests
└── bench_test.go       # Benchmarks
```

---

## Feature Comparison

| Feature | kubernetes-mcp | mcp-grafana | mcp-k8s-go | grist-ctl |
|---------|---------------|-------------|------------|-----------|
| Read-only mode | Yes | Yes | Yes | No |
| Config file | TOML | Flags | YAML | Env vars |
| Hot reload | SIGHUP | No | No | No |
| Multi-arch Docker | No | No | Yes | No |
| Helm chart | Yes | No | No | No |
| Test suite | Yes | Yes | Yes | Minimal |
| Panic recovery | Unknown | Yes | Unknown | No |

---

## Recommendations for grist-ctl

Based on this analysis:

1. **Add read-only mode** - Essential for production safety
2. **Implement lifecycle hooks** - For observability
3. **Add panic recovery** - One-line change
4. **Create proper test suite** - Follow kubernetes-mcp-server pattern
5. **Consider config file** - YAML or TOML for complex setups
6. **Extract tool definitions** - To separate file for clarity
7. **Add middleware support** - For rate limiting, validation

---

## Sources

- [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server)
- [grafana/mcp-grafana](https://github.com/grafana/mcp-grafana)
- [strowk/mcp-k8s-go](https://github.com/strowk/mcp-k8s-go)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- [mcp-go Server Package](https://pkg.go.dev/github.com/mark3labs/mcp-go/server)
