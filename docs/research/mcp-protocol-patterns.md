# MCP Protocol Implementation Patterns

Research conducted December 2025 on MCP protocol best practices.

## Executive Summary

MCP is built on JSON-RPC 2.0 and provides a universal interface for AI-to-tool communication. This document covers protocol-level implementation details including lifecycle management, error handling, and progress reporting.

---

## 1. JSON-RPC 2.0 Fundamentals

### Message Formats

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_weather",
    "arguments": {"location": "New York"}
  }
}
```

**Response (Success):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [{"type": "text", "text": "Weather data..."}],
    "isError": false
  }
}
```

**Response (Error):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {"field": "location"}
  }
}
```

**Notification (No Response):**
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {"progressToken": "abc123", "progress": 50}
}
```

### Reserved Error Codes

| Code | Name | Usage |
|------|------|-------|
| `-32700` | Parse Error | Invalid JSON |
| `-32600` | Invalid Request | Missing JSON-RPC fields |
| `-32601` | Method Not Found | Unknown method |
| `-32602` | Invalid Params | Validation failed |
| `-32603` | Internal Error | Server exception |
| `-32002` | Resource Not Found | MCP-specific |
| `-32800` | Request Cancelled | Client cancelled |

---

## 2. Lifecycle Management

### Initialization Phase

**Step 1 - Client Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "roots": {"listChanged": true}
    },
    "clientInfo": {"name": "ExampleClient", "version": "1.0.0"}
  }
}
```

**Step 2 - Server Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {"listChanged": true},
      "resources": {"subscribe": true}
    },
    "serverInfo": {"name": "gristle", "version": "1.0.0"}
  }
}
```

**Step 3 - Client Notification:**
```json
{"jsonrpc": "2.0", "method": "notifications/initialized"}
```

### Version Negotiation

1. Client sends latest version it supports
2. Server responds with same version if supported
3. Otherwise, server responds with another supported version
4. Client disconnects if it doesn't support server's version

### Shutdown Phase (stdio)

1. Client closes input stream to server
2. Client waits for server to exit
3. Client sends SIGTERM if server doesn't exit
4. Client sends SIGKILL if still running

---

## 3. Error Handling

### Three-Tier Error Model

| Level | Description | Examples |
|-------|-------------|----------|
| **Transport** | Connection issues | Network timeout, broken pipe |
| **Protocol** | JSON-RPC violations | Malformed JSON, invalid method |
| **Application** | Business logic | API errors, validation failures |

### Protocol Error vs Tool Error

**Protocol Error (unknown tool):**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {"code": -32601, "message": "Unknown tool: invalid_tool"}
}
```

**Tool Execution Error (LLM can see this):**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "result": {
    "content": [{"type": "text", "text": "API rate limit exceeded. Retry in 60s."}],
    "isError": true
  }
}
```

### Error Message Best Practices

**For AI Agents - Be Actionable:**
- BAD: "Access denied"
- GOOD: "Access denied: API_TOKEN is invalid or expired. Please reconfigure with valid credentials."

**Human-Readable:**
- BAD: "Error 422: Unprocessable Entity"
- GOOD: "Cannot assign issue: User is not a collaborator on this repository."

---

## 4. Tool Schema Design

### Tool Definition Structure

```json
{
  "name": "create_issue",
  "title": "Create GitHub Issue",
  "description": "Create a new issue in a GitHub repository",
  "inputSchema": {
    "type": "object",
    "properties": {
      "repository": {
        "type": "string",
        "pattern": "^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"
      },
      "title": {
        "type": "string",
        "minLength": 1,
        "maxLength": 256
      }
    },
    "required": ["repository", "title"]
  }
}
```

### Best Practices

| Practice | Description |
|----------|-------------|
| Clear naming | Action-oriented (`create_issue`, not `issue`) |
| Detailed descriptions | Help LLMs understand usage |
| Atomic operations | One tool = one focused task |
| Use enums | Constrain string parameters |
| Type constraints | Use min/max, pattern, required |

### Go Implementation

```go
tool := mcp.NewTool("export_doc",
    mcp.WithDescription("Export a document to a file"),
    mcp.WithString("doc_id",
        mcp.Description("Document ID to export"),
        mcp.Required(),
    ),
    mcp.WithString("format",
        mcp.Description("Export format"),
        mcp.Required(),
        mcp.Enum("excel", "grist"),
    ),
)
```

---

## 5. Progress Reporting

### Progress Token Flow

**Request with Token:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "analyze_repo",
    "_meta": {"progressToken": "op-123"}
  }
}
```

**Progress Notifications:**
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {
    "progressToken": "op-123",
    "progress": 25,
    "total": 100,
    "message": "Analyzing 250/1000 files..."
  }
}
```

### Implementation

```go
func longRunningTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    progressToken := req.GetProgressToken()

    for i := 0; i < total; i++ {
        select {
        case <-ctx.Done():
            return mcp.NewToolResultError("cancelled"), nil
        default:
        }

        if progressToken != nil && i%10 == 0 {  // Rate limit notifications
            session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
                ProgressToken: progressToken,
                Progress:      float64(i + 1),
                Total:         float64(total),
            })
        }
    }
    return result, nil
}
```

### Rules

- Token MUST be unique across active requests
- Progress values MUST increase with each notification
- Total can be omitted if unknown
- Rate limit notifications
- Stop notifications after completion

---

## 6. Cancellation

### Client Cancels Request

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/cancelled",
  "params": {
    "requestId": 1,
    "reason": "User cancelled"
  }
}
```

### Server-Side Handling

```go
func handleToolCall(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    for i := 0; i < totalItems; i++ {
        select {
        case <-ctx.Done():
            return mcp.NewToolResultError("Operation cancelled"), nil
        default:
        }
        // Process item...
    }
    return result, nil
}
```

---

## 7. Notifications

### Server-to-Client

```json
// Logging
{"jsonrpc": "2.0", "method": "notifications/message", "params": {"level": "warning", "data": {...}}}

// List changes
{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"}
{"jsonrpc": "2.0", "method": "notifications/resources/list_changed"}
{"jsonrpc": "2.0", "method": "notifications/resources/updated", "params": {"uri": "..."}}
```

### Client-to-Server

```json
{"jsonrpc": "2.0", "method": "notifications/initialized"}
{"jsonrpc": "2.0", "method": "notifications/roots/list_changed"}
```

### Go Handler Registration

```go
server := mcp.NewServer(impl, &mcp.ServerOptions{
    InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
        log.Println("Client initialized")
    },
    RootsListChangedHandler: func(ctx context.Context, req *mcp.RootsListChangedRequest) {
        roots, _ := req.Session.ListRoots(ctx, nil)
        // Handle root changes
    },
})
```

---

## 8. Capabilities

### Client Capabilities

| Capability | Description |
|------------|-------------|
| `roots.listChanged` | Can provide filesystem roots |
| `sampling` | Supports LLM sampling requests |
| `experimental` | Non-standard features |

### Server Capabilities

| Capability | Description |
|------------|-------------|
| `prompts.listChanged` | Offers prompt templates |
| `resources.subscribe` | Supports resource subscriptions |
| `resources.listChanged` | Notifies on resource changes |
| `tools.listChanged` | Notifies on tool changes |
| `logging` | Emits structured logs |
| `completions` | Supports argument autocompletion |

### Go Registration

```go
s := server.NewMCPServer("gristle", "1.0.0",
    server.WithToolCapabilities(true),
    server.WithResourceCapabilities(true, true),  // subscribe, listChanged
    server.WithPromptCapabilities(true),          // listChanged
    server.WithLogging(),
)
```

---

## 9. Security Considerations

### Trust Boundaries

| Element | Trust Level |
|---------|-------------|
| Tool descriptions | Untrusted (unless verified) |
| Tool annotations | Untrusted |
| User input | Untrusted |
| Server responses | Verify against schema |

### Best Practices

1. Validate all inputs against inputSchema
2. Sanitize for injection attacks
3. Implement access controls
4. Rate limit tool invocations
5. Require user consent for sensitive operations
6. Log tool usage for auditing
7. Implement timeouts
8. Never expose secrets in errors/logs

---

## Sources

- [MCP Specification](https://modelcontextprotocol.io/specification/2025-06-18)
- [MCP Lifecycle](https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle)
- [MCP Tools](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [MCP Progress](https://modelcontextprotocol.io/specification/2025-03-26/basic/utilities/progress)
- [JSON-RPC 2.0](https://www.jsonrpc.org/specification)
- [MCPcat Error Handling](https://mcpcat.io/guides/error-handling-custom-mcp-servers/)
