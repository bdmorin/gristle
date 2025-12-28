# Go MCP Libraries Research

Research conducted December 2025 on the Go MCP library ecosystem.

## Executive Summary

The Model Context Protocol (MCP) ecosystem in Go has matured significantly, with an official SDK now available alongside several battle-tested community libraries.

## Library Comparison

### 1. mark3labs/mcp-go (Community SDK)

**Repository:** https://github.com/mark3labs/mcp-go
**Stars:** 7,900+
**Forks:** 744
**Imports:** 1,307+ packages
**License:** MIT

#### Key Features
- High-level API with minimal boilerplate
- Multiple transports: stdio, SSE, streamable-HTTP
- Functional options pattern (`WithDescription()`, `WithString()`, `Required()`)
- Comprehensive hook system (BeforeAny, OnSuccess, OnError)
- Session management with per-session tool filtering
- Panic recovery middleware (`WithRecovery()`)
- Testing utilities in `mcptest/` package

#### Example
```go
s := server.NewMCPServer("Demo", "1.0.0",
    server.WithToolCapabilities(false),
    server.WithRecovery(),
)

tool := mcp.NewTool("hello_world",
    mcp.WithDescription("Say hello to someone"),
    mcp.WithString("name", mcp.Required()),
)
s.AddTool(tool, helloHandler)
```

---

### 2. modelcontextprotocol/go-sdk (Official SDK)

**Repository:** https://github.com/modelcontextprotocol/go-sdk
**Stars:** 3,500+
**Forks:** 318
**Contributors:** 70+
**Maintained by:** Google + Anthropic
**License:** MIT
**Latest:** v1.2.0 (stable, December 2025)

#### Package Structure
| Package | Purpose |
|---------|---------|
| `mcp` | Primary APIs for clients and servers |
| `jsonrpc` | Custom transport implementations |
| `auth` | OAuth authentication primitives |
| `oauthex` | OAuth protocol extensions |

#### Key Features
- Official SDK with stability guarantees
- Strongly-typed handlers with generics
- Automatic JSON schema derivation from struct tags
- Uses `github.com/google/jsonschema-go` for schema inference

#### Example
```go
type Input struct {
    Name string `json:"name" jsonschema:"the name of the person"`
}

type Output struct {
    Greeting string `json:"greeting"`
}

mcp.AddTool(server, &mcp.Tool{Name: "greet"},
    func(ctx context.Context, req Request, input Input) (Output, error) {
        return Output{Greeting: "Hello " + input.Name}, nil
    })
```

#### Limitations
- OAuth client-side support pending
- Streamable HTTP resumability needs improvement

---

### 3. metoro-io/mcp-golang

**Repository:** https://github.com/metoro-io/mcp-golang
**Stars:** 1,200+
**Forks:** 117
**Documentation:** https://mcpgolang.com
**License:** MIT

#### Key Features
- Type safety via native Go struct arguments
- Automatic schema generation from structs
- Multiple transports: stdio, HTTP/Gin, SSE
- Minimal boilerplate

#### Example
```go
type CalculatorArgs struct {
    Operation string  `json:"op" jsonschema:"enum=add,enum=subtract"`
    X         float64 `json:"x" jsonschema:"minimum=0"`
    Y         float64 `json:"y" jsonschema:"minimum=0"`
}

server.RegisterTool("calculate", "Perform calculations", handler)
```

#### Limitations
- HTTP transports are stateless (no bidirectional features)
- Less community adoption than mark3labs

---

### 4. go-go-golems/go-go-mcp

**Repository:** https://github.com/go-go-golems/go-go-mcp
**Stars:** 19
**Installation:** Homebrew, apt-get, yum, go get

#### Unique Features
- Shell script templates as MCP tools (Go templates + Sprig)
- YAML-based configuration profiles
- Dynamic file/repository watching with hot reload
- SSE-to-stdio bridge
- UI DSL for defining user interfaces

---

## Comparison Matrix

| Feature | Official SDK | mark3labs/mcp-go | metoro-io/mcp-golang |
|---------|-------------|------------------|---------------------|
| **Stars** | 3.5k | 7.9k | 1.2k |
| **Stability** | Stable (v1.x) | Active development | Active development |
| **Transport: stdio** | Yes | Yes | Yes |
| **Transport: SSE** | Yes | Yes | Yes |
| **Transport: HTTP** | Yes | Yes | Yes |
| **OAuth Support** | Partial | No | No |
| **Type-safe schemas** | Generics | Explicit | Reflection |
| **Testing utilities** | In-memory transport | mcptest package | Mock transport |

---

## Recommendation

For grist-ctl, **mark3labs/mcp-go** remains the best choice:
- Most mature and battle-tested
- Largest community (1,307+ dependent packages)
- Comprehensive hook system for observability
- Built-in panic recovery
- Testing utilities with `mcptest`

Consider migrating to the **official SDK** if:
- OAuth support becomes a requirement
- Need official long-term support guarantees
- Want generics-based type-safe handlers

---

## MCP Ecosystem Statistics (2025)

- **MCP Protocol:** 37k GitHub stars
- **Total MCP Servers:** 5,867+ (up from 100 in Nov 2024)
- **Downloads:** 8 million (up from 100k in Nov 2024)

### Major Adopters
- **OpenAI:** MCP support in ChatGPT desktop, Agents SDK
- **Google:** Native MCP in Gemini 2.5 Pro, maintains official Go SDK
- **Microsoft:** MCP support in Copilot Studio
- **GitHub:** MCP Server migrated from mark3labs to official SDK

---

## Sources

- [Official Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [mcp-go Documentation](https://mcp-go.dev)
- [metoro-io/mcp-golang](https://github.com/metoro-io/mcp-golang)
- [MCP Adoption Statistics](https://mcpmanager.ai/blog/mcp-adoption-statistics/)
- [Go SDK Design Discussion](https://github.com/orgs/modelcontextprotocol/discussions/364)
