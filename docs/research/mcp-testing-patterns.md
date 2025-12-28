# MCP Server Testing Patterns for Go

Research conducted December 2025 on testing best practices for MCP servers.

## Executive Summary

This document covers comprehensive testing patterns for MCP servers including unit tests, integration tests with mcptest, fuzz testing, and benchmarks.

---

## 1. Unit Testing Tool Handlers

### Pattern: Extract Handler Logic

```go
// handlers.go - Extracted, testable logic
type Handlers struct {
    api GristAPI
}

func (h *Handlers) ListOrgs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    orgs := h.api.GetOrgs()
    result := make([]orgInfo, len(orgs))
    for i, org := range orgs {
        result[i] = orgInfo{ID: org.Id, Name: org.Name}
    }
    return encodeResult(result)
}
```

### Pattern: Interface-Based Mocking

```go
// interface.go
type GristAPI interface {
    GetOrgs() []gristapi.Org
    GetOrgWorkspaces(orgID int) []gristapi.Workspace
    GetDoc(docID string) gristapi.Doc
}

// mock.go
type MockGristAPI struct {
    OrgsFunc          func() []gristapi.Org
    OrgWorkspacesFunc func(int) []gristapi.Workspace
}

func (m *MockGristAPI) GetOrgs() []gristapi.Org {
    if m.OrgsFunc != nil {
        return m.OrgsFunc()
    }
    return nil
}
```

### Pattern: Table-Driven Tests

```go
func TestListOrgsHandler(t *testing.T) {
    tests := []struct {
        name      string
        mockOrgs  []gristapi.Org
        wantLen   int
        wantError bool
    }{
        {
            name:     "empty organizations",
            mockOrgs: []gristapi.Org{},
            wantLen:  0,
        },
        {
            name: "multiple organizations",
            mockOrgs: []gristapi.Org{
                {Id: 1, Name: "Org1"},
                {Id: 2, Name: "Org2"},
            },
            wantLen: 2,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := &MockGristAPI{
                OrgsFunc: func() []gristapi.Org { return tt.mockOrgs },
            }

            result, err := listOrgsHandler(context.Background(), mock)

            if (err != nil) != tt.wantError {
                t.Errorf("error = %v, wantError %v", err, tt.wantError)
            }
            if len(result) != tt.wantLen {
                t.Errorf("len = %d, want %d", len(result), tt.wantLen)
            }
        })
    }
}
```

---

## 2. Integration Testing with mcptest

### Basic Setup

```go
import (
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/mcptest"
)

func TestMCPServerIntegration(t *testing.T) {
    s := NewServer()
    testServer, err := mcptest.NewServerFromMCPServer(t, s)
    if err != nil {
        t.Fatalf("Failed to create test server: %v", err)
    }
    defer testServer.Close()

    client := testServer.Client()
    ctx := context.Background()

    // Test tool listing
    tools, err := client.ListTools(ctx)
    if err != nil {
        t.Fatalf("ListTools failed: %v", err)
    }

    if len(tools.Tools) != 6 {
        t.Errorf("got %d tools, want 6", len(tools.Tools))
    }
}
```

### Testing Tool Calls

```go
func TestCallTool(t *testing.T) {
    testServer, _ := mcptest.NewServer(t, serverTool)
    defer testServer.Close()

    client := testServer.Client()

    result, err := client.CallTool(ctx, mcp.CallToolRequest{
        Params: mcp.CallToolRequestParams{
            Name: "list_orgs",
        },
    })

    if err != nil {
        t.Fatalf("CallTool failed: %v", err)
    }

    if result.IsError {
        t.Errorf("Expected success, got error")
    }
}
```

### NewUnstartedServer for Setup

```go
func TestServerWithMultipleTools(t *testing.T) {
    testServer := mcptest.NewUnstartedServer(t)

    testServer.AddTool(
        mcp.NewTool("tool1", mcp.WithDescription("First tool")),
        handler1,
    )
    testServer.AddTool(
        mcp.NewTool("tool2", mcp.WithDescription("Second tool")),
        handler2,
    )

    if err := testServer.Start(); err != nil {
        t.Fatalf("Failed to start: %v", err)
    }
    defer testServer.Close()

    // Test both tools...
}
```

---

## 3. Testing stdio Servers

### Pattern: io.Pipe Simulation

```go
func TestStdioServer(t *testing.T) {
    stdinReader, stdinWriter := io.Pipe()
    stdoutReader, stdoutWriter := io.Pipe()

    go func() {
        runServerWithIO(stdinReader, stdoutWriter)
    }()

    // Send JSON-RPC request
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  "tools/list",
        "params":  map[string]interface{}{},
    }

    requestBytes, _ := json.Marshal(request)
    stdinWriter.Write(append(requestBytes, '\n'))

    // Read response
    scanner := bufio.NewScanner(stdoutReader)
    if scanner.Scan() {
        var response map[string]interface{}
        json.Unmarshal(scanner.Bytes(), &response)

        if response["error"] != nil {
            t.Errorf("Unexpected error: %v", response["error"])
        }
    }

    stdinWriter.Close()
}
```

---

## 4. Fuzz Testing

### Fuzz Tool Inputs

```go
func FuzzExportDocParams(f *testing.F) {
    seeds := []string{
        `{"doc_id": "abc123456789", "format": "excel"}`,
        `{"doc_id": "", "format": "excel"}`,
        `{"doc_id": "../../../etc/passwd", "format": "excel"}`,
        `{}`,
    }

    for _, seed := range seeds {
        f.Add([]byte(seed))
    }

    f.Fuzz(func(t *testing.T, data []byte) {
        var args map[string]interface{}
        if err := json.Unmarshal(data, &args); err != nil {
            return  // Skip invalid JSON
        }

        req := mcp.CallToolRequest{
            Params: mcp.CallToolRequestParams{
                Name:      "export_doc",
                Arguments: args,
            },
        }

        // Handler should never panic
        handler := createExportDocHandler(mockAPI)
        result, err := handler(context.Background(), req)

        if result == nil && err == nil {
            t.Error("Handler returned nil result and nil error")
        }
    })
}
```

### Fuzz Path Traversal

```go
func FuzzPathTraversal(f *testing.F) {
    f.Add("normal_file.xlsx")
    f.Add("../../../etc/passwd")
    f.Add("..\\..\\windows\\system32")
    f.Add("/absolute/path.xlsx")
    f.Add(strings.Repeat("a", 1000))

    f.Fuzz(func(t *testing.T, filename string) {
        sanitized := SanitizeFilename(filename)

        if strings.Contains(sanitized, "..") {
            t.Errorf("Contains '..': %s", sanitized)
        }
        if strings.ContainsAny(sanitized, "/\\") {
            t.Errorf("Contains path separator: %s", sanitized)
        }
    })
}
```

### Running Fuzz Tests

```bash
# Run for 30 seconds
go test -fuzz=FuzzExportDocParams -fuzztime=30s

# Run with reduced parallelism
go test -fuzz=FuzzPathTraversal -fuzztime=60s -parallel=1
```

---

## 5. Benchmark Testing

### Handler Benchmarks

```go
func BenchmarkListOrgsHandler(b *testing.B) {
    mock := &MockGristAPI{
        OrgsFunc: func() []gristapi.Org {
            return make([]gristapi.Org, 10)
        },
    }
    handler := createListOrgsHandler(mock)
    req := createTestRequest("list_orgs", nil)
    ctx := context.Background()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, _ = handler(ctx, req)
    }
}
```

### JSON Encoding Comparison

```go
func BenchmarkJSONEncoding(b *testing.B) {
    orgs := make([]orgInfo, 100)
    for i := range orgs {
        orgs[i] = orgInfo{ID: i, Name: fmt.Sprintf("Org%d", i)}
    }

    b.Run("MarshalIndent", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            json.MarshalIndent(orgs, "", "  ")
        }
    })

    b.Run("Marshal", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            json.Marshal(orgs)
        }
    })
}
```

### Scaling Benchmarks

```go
func BenchmarkListWorkspaces(b *testing.B) {
    sizes := []int{1, 10, 50, 100, 500}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("workspaces_%d", size), func(b *testing.B) {
            mock := createMockWithWorkspaces(size)
            handler := createListWorkspacesHandler(mock)

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                handler(context.Background(), req)
            }
        })
    }
}
```

---

## 6. CI Integration

### GitHub Actions Workflow

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run Tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Run Fuzz Tests
        run: go test -fuzz=. -fuzztime=30s ./mcp/

      - name: Run Benchmarks
        run: go test -bench=. -benchmem ./mcp/ | tee bench.txt

      - name: Compare Benchmarks
        run: |
          if [ -f bench_baseline.txt ]; then
            benchstat bench_baseline.txt bench.txt
          fi
```

---

## 7. Testing Tools Summary

| Tool/Package | Purpose |
|--------------|---------|
| `mcptest` | Integration testing with pre-initialized client |
| `io.Pipe()` | Simulate stdin/stdout |
| `testing.F` | Native Go fuzzing |
| `benchstat` | Benchmark comparison |
| `httptest` | HTTP handler testing |
| `-race` flag | Data race detection |

---

## Sources

- [mcp-go mcptest Package](https://pkg.go.dev/github.com/mark3labs/mcp-go/mcptest)
- [Go Fuzz Testing](https://go.dev/doc/security/fuzz/)
- [Go Benchmark Testing](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Go Unit Testing Best Practices](https://www.glukhov.org/post/2025/11/unit-tests-in-go/)
