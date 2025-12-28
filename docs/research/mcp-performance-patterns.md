# MCP Performance Best Practices for Go

Research conducted December 2025 on performance optimization for MCP servers.

## Executive Summary

Go's native concurrency model, efficient garbage collection, and static binary compilation make it excellent for high-performance MCP servers. Key optimizations include HTTP client reuse, JSON encoding improvements, and buffered I/O.

---

## 1. HTTP Client Reuse

### Problem
Creating a new `http.Client` per request prevents TCP connection reuse:

```go
// BAD: New client every request
func httpRequest(...) {
    client := &http.Client{}  // No connection reuse!
}
```

### Solution
```go
var (
    httpClient *http.Client
    clientOnce sync.Once
)

func getHTTPClient() *http.Client {
    clientOnce.Do(func() {
        httpClient = &http.Client{
            Transport: &http.Transport{
                MaxIdleConnsPerHost: 10,
                MaxIdleConns:        100,
                IdleConnTimeout:     90 * time.Second,
                TLSHandshakeTimeout: 10 * time.Second,
                ExpectContinueTimeout: 1 * time.Second,
            },
            Timeout: 30 * time.Second,
        }
    })
    return httpClient
}
```

### Critical: Read Response Body Completely
```go
resp, err := httpClient.Do(req)
if err != nil {
    return nil, err
}
defer resp.Body.Close()

// MUST read body completely for connection reuse
body, err := io.ReadAll(resp.Body)
```

---

## 2. JSON Encoding Optimization

### Benchmark Results

| Method | Speed | Notes |
|--------|-------|-------|
| `json.MarshalIndent` | Baseline | Pretty-prints, slow |
| `json.Marshal` | ~1.5x faster | Standard library |
| `jsoniter` | ~2-3x faster | Drop-in replacement |
| `sonic` | ~17x faster | ByteDance, requires CGO |

### Standard Library Optimization
```go
// BEFORE (slow)
jsonBytes, err := json.MarshalIndent(result, "", "  ")

// AFTER (faster)
jsonBytes, err := json.Marshal(result)
```

### jsoniter (Recommended)
```go
import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Drop-in replacement
jsonBytes, err := json.Marshal(result)
```

### sonic (Maximum Performance)
```go
import "github.com/bytedance/sonic"

// Pre-touch for large schemas (avoids JIT delay)
sonic.Pretouch(reflect.TypeOf(MyStruct{}))

jsonBytes, err := sonic.Marshal(result)
```

---

## 3. Buffered I/O for stdio Transport

### Why Buffering Matters
- Unbuffered I/O: ~901ms per iteration
- Buffered I/O: ~39ms per iteration (23x improvement)

### Implementation
```go
import (
    "bufio"
    "os"
)

func newBufferedStdio() (*bufio.Reader, *bufio.Writer) {
    // 64KB buffer for high-throughput scenarios
    reader := bufio.NewReaderSize(os.Stdin, 64*1024)
    writer := bufio.NewWriterSize(os.Stdout, 64*1024)
    return reader, writer
}

// Critical: Always flush buffered writer
func writeResponse(w *bufio.Writer, data []byte) error {
    _, err := w.Write(data)
    if err != nil {
        return err
    }
    return w.Flush()  // Don't forget this!
}
```

---

## 4. sync.Pool for Buffer Reuse

### Pattern
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func encodeJSON(v interface{}) ([]byte, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()

    if err := json.NewEncoder(buf).Encode(v); err != nil {
        return nil, err
    }

    // Return a copy (buffer goes back to pool)
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}
```

### When to Use
- Frequently called handlers
- Large JSON payloads
- High-throughput scenarios

---

## 5. Concurrency Patterns

### Worker Pool for Parallel Tool Execution
```go
import "github.com/alitto/pond"

pool := pond.New(10, 100)  // 10 workers, 100 task queue

for _, toolCall := range toolCalls {
    call := toolCall
    pool.Submit(func() {
        result := executeToolCall(call)
        resultsChan <- result
    })
}

pool.StopAndWait()
```

### Bounded Concurrency with Semaphore
```go
import "golang.org/x/sync/semaphore"

var sem = semaphore.NewWeighted(10)  // Max 10 concurrent

func executeTool(ctx context.Context, call ToolCall) (Result, error) {
    if err := sem.Acquire(ctx, 1); err != nil {
        return Result{}, err
    }
    defer sem.Release(1)

    return executeToolLogic(ctx, call)
}
```

### Fan-Out/Fan-In Pattern
```go
func executeToolsConcurrently(ctx context.Context, calls []ToolCall) []Result {
    results := make([]Result, len(calls))
    var wg sync.WaitGroup

    for i, call := range calls {
        wg.Add(1)
        go func(idx int, tc ToolCall) {
            defer wg.Done()
            select {
            case <-ctx.Done():
                results[idx] = Result{Error: ctx.Err()}
            default:
                results[idx] = executeTool(ctx, tc)
            }
        }(i, call)
    }

    wg.Wait()
    return results
}
```

---

## 6. SSE Server Optimization

### Performance Benchmark
- ~100K messages/second across ~1000 connections
- Single CPU core (GOMAXPROCS=1)

### Essential Headers and Flushing
```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", 500)
        return
    }

    for {
        select {
        case msg := <-messageChan:
            fmt.Fprintf(w, "data: %s\n\n", msg)
            flusher.Flush()  // Immediate delivery
        case <-r.Context().Done():
            return
        }
    }
}
```

---

## 7. Graceful Shutdown

```go
import "os/signal"

func runMCPServer() error {
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    server := NewMCPServer()

    errChan := make(chan error, 1)
    go func() {
        errChan <- server.Serve(ctx)
    }()

    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(
            context.Background(), 30*time.Second)
        defer cancel()
        return server.Shutdown(shutdownCtx)
    }
}
```

---

## 8. Transport Performance Comparison

| Transport | Use Case | Throughput |
|-----------|----------|------------|
| stdio | CLI tools, local AI | Highest (no network) |
| HTTP | Web services | Good with pooling |
| SSE | Real-time streaming | ~100K msgs/sec |

---

## Quick Wins Checklist

- [ ] Replace per-request `&http.Client{}` with singleton
- [ ] Switch `MarshalIndent` to `Marshal`
- [ ] Consider jsoniter for 2-3x JSON speedup
- [ ] Add sync.Pool for frequently allocated buffers
- [ ] Ensure HTTP response bodies are fully read
- [ ] Configure proper connection pool sizes

---

## Sources

- [Go Performance Guide - Buffered I/O](https://goperf.dev/01-common-patterns/buffered-io/)
- [HTTP Connection Pooling](https://davidbacisin.com/writing/golang-http-connection-pools-1)
- [VictoriaMetrics sync.Pool Guide](https://victoriametrics.com/blog/go-sync-pool/)
- [jsoniter Benchmarks](https://jsoniter.com/benchmark.html)
- [ByteDance sonic](https://github.com/bytedance/sonic)
- [pond Worker Pool](https://github.com/alitto/pond)
- [mroth/sseserver](https://github.com/mroth/sseserver)
