# MCP Best Practices 2025: Comprehensive Guide

Research conducted December 30, 2025 on Model Context Protocol server best practices and design patterns.

## Executive Summary

This document consolidates the latest MCP (Model Context Protocol) best practices for 2025, focusing on architecture, security, performance, and testing strategies. The MCP specification version 2025-11-25 was released on November 25, 2025, marking the protocol's first anniversary with significant updates including asynchronous Tasks, enhanced sampling capabilities, and improved security guidelines.

### Key Updates in MCP 2025-11-25

- **Tasks**: New abstraction for tracking asynchronous work with status queries
- **Enhanced Sampling**: Tools can now be included in sampling requests with concurrent execution
- **Security Improvements**: Clarified authorization handling and Resource Indicators to prevent malicious token access
- **Widespread Adoption**: 407% growth in MCP Registry (nearly 2,000 entries), adopted by OpenAI, Google DeepMind

---

## 1. MCP Server Architecture

### 1.1 Tool Handler Patterns

**Best Practice: Structured Input Validation**

```go
import "github.com/go-playground/validator/v10"

type ToolInput struct {
    DocID    string `json:"doc_id" validate:"required,docid"`
    Format   string `json:"format" validate:"required,oneof=excel grist csv"`
    Filename string `json:"filename,omitempty" validate:"omitempty,safepath,max=255"`
}

func (s *MCPServer) handleExportDoc(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 1. Parse input
    var input ToolInput
    if err := req.UnmarshalParams(&input); err != nil {
        return mcp.NewToolResultError("invalid input: " + err.Error()), nil
    }

    // 2. Validate input
    if err := validate.Struct(&input); err != nil {
        return mcp.NewToolResultError("validation failed: " + err.Error()), nil
    }

    // 3. Check rate limits
    if err := rateLimiter.Allow(ctx, "export_doc"); err != nil {
        return mcp.NewToolResultError("rate limit exceeded"), nil
    }

    // 4. Execute with timeout
    execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    result, err := s.executeExport(execCtx, input)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    return mcp.NewToolResultText(result), nil
}
```

**Recommendation for gristle**: Extract handler logic into separate functions for better testability. Current implementation mixes validation, business logic, and formatting.

### 1.2 State Management

**Session Lifecycle Best Practices**

Every MCP connection follows a predictable lifecycle:
1. **Initialization**: Client sends initialize request, server generates unique session ID
2. **Active Use**: Client makes requests with session ID, server maintains context
3. **Idle Period**: No requests for X minutes, server marks for cleanup
4. **Termination**: Explicit close or timeout, server releases all resources

```go
type SessionManager struct {
    sessions sync.Map
    maxAge   time.Duration
}

type Session struct {
    ID        string
    CreatedAt time.Time
    ExpiresAt time.Time
    UserAgent string
    IPAddress string
    Context   map[string]interface{}
}

func (sm *SessionManager) CreateSession(userAgent, ip string) (*Session, error) {
    session := &Session{
        ID:        generateSecureID(),
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(sm.maxAge),
        UserAgent: userAgent,
        IPAddress: ip,
        Context:   make(map[string]interface{}),
    }

    sm.sessions.Store(session.ID, session)
    return session, nil
}

func (sm *SessionManager) ValidateSession(sessionID, userAgent, ip string) error {
    val, ok := sm.sessions.Load(sessionID)
    if !ok {
        return errors.New("session not found")
    }

    session := val.(*Session)

    // Session binding prevents hijacking
    if session.UserAgent != userAgent || session.IPAddress != ip {
        return errors.New("session binding mismatch")
    }

    if time.Now().After(session.ExpiresAt) {
        sm.sessions.Delete(sessionID)
        return errors.New("session expired")
    }

    return nil
}

// Background cleanup
func (sm *SessionManager) startCleanup(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            sm.sessions.Range(func(key, val interface{}) bool {
                session := val.(*Session)
                if time.Now().After(session.ExpiresAt) {
                    sm.sessions.Delete(key)
                }
                return true
            })
        case <-ctx.Done():
            return
        }
    }
}
```

**Critical Issue**: The current mcp-go SDK lacks a disconnect() method, leading to resource leaks. Workaround: Implement custom cleanup with context cancellation.

### 1.3 Resource Exposure

**Best Practice: URI Schemes and Access Control**

```go
type ResourceManager struct {
    handlers map[string]ResourceHandler
}

type ResourceHandler interface {
    CanAccess(ctx context.Context, uri string) (bool, error)
    GetResource(ctx context.Context, uri string) (*Resource, error)
}

func (rm *ResourceManager) Register(scheme string, handler ResourceHandler) {
    rm.handlers[scheme] = handler
}

// Example: config:// scheme for configuration resources
func (s *ConfigResourceHandler) GetResource(ctx context.Context, uri string) (*Resource, error) {
    // Security: Validate URI format
    if !strings.HasPrefix(uri, "config://") {
        return nil, errors.New("invalid scheme")
    }

    // Security: Prevent path traversal
    path := strings.TrimPrefix(uri, "config://")
    if strings.Contains(path, "..") || filepath.IsAbs(path) {
        return nil, errors.New("invalid path")
    }

    // Check access control
    ok, err := s.CanAccess(ctx, uri)
    if err != nil || !ok {
        return nil, errors.New("access denied")
    }

    // Return resource with proper metadata
    return &Resource{
        URI:      uri,
        MimeType: "application/json",
        Contents: s.loadConfig(path),
    }, nil
}
```

**Security Safeguards for Resources**:
- Sanitize URIs to prevent injection attacks
- Apply fine-grained access controls based on user context
- Minimize exposure of sensitive binary data (IDs, private media)
- Log and rate-limit access to sensitive resources
- Use embedded resources in prompts for controlled context injection

### 1.4 Prompt Handling

**Best Practice: User-Controlled Prompts with Clear Names**

```go
type Prompt struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Arguments   []PromptArgument       `json:"arguments,omitempty"`
}

func (s *MCPServer) RegisterPrompt(name, description string) {
    // Use clear, actionable names
    // Good: "summarize-errors", "analyze-performance"
    // Bad: "get-summarized-error-log-output"

    s.prompts[name] = &Prompt{
        Name:        name,
        Description: description,
        Arguments: []PromptArgument{
            {Name: "timeframe", Required: true, Description: "Time range for analysis"},
        },
    }
}

func (s *MCPServer) GetPrompt(ctx context.Context, name string, args map[string]string) (*PromptResult, error) {
    prompt, exists := s.prompts[name]
    if !exists {
        return nil, errors.New("prompt not found")
    }

    // Security: Validate all arguments
    if err := s.validatePromptArgs(prompt, args); err != nil {
        return nil, err
    }

    // Security: Scan for injection attempts
    for _, arg := range args {
        if warnings := ScanForInjection(arg); len(warnings) > 0 {
            return nil, fmt.Errorf("potential injection detected: %v", warnings)
        }
    }

    // Generate prompt with embedded resources
    return s.generatePromptContent(ctx, name, args)
}
```

**Key Principles**:
- Prompts are **user-controlled**, triggered through explicit UI commands
- Validate all required arguments upfront
- Use embedded resources for documentation, code samples, reference materials
- Prevent injection attacks through input scanning
- Implementations MUST validate all inputs/outputs

---

## 2. MCP Security Best Practices

### 2.1 Authentication & Authorization

**OAuth 2.1 with PKCE (Recommended for HTTP Transport)**

```go
import "golang.org/x/oauth2"

type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
    Endpoint     oauth2.Endpoint
}

func (s *MCPServer) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
    // Validate PKCE verifier
    code := r.URL.Query().Get("code")
    verifier := r.URL.Query().Get("code_verifier")

    token, err := s.oauth.Exchange(r.Context(), code,
        oauth2.SetAuthURLParam("code_verifier", verifier))
    if err != nil {
        http.Error(w, "OAuth exchange failed", 401)
        return
    }

    // CRITICAL: Validate token was issued for THIS server
    if !s.validateTokenAudience(token) {
        http.Error(w, "Token audience mismatch", 403)
        return
    }

    // Create session with token binding
    session, err := s.sessions.CreateWithToken(token, r.UserAgent(), r.RemoteAddr)
    if err != nil {
        http.Error(w, "Session creation failed", 500)
        return
    }

    // Set short-lived session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "mcp_session",
        Value:    session.ID,
        MaxAge:   3600, // 1 hour
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })
}
```

**Critical Security Rules**:
1. **Use OAuth 2.1 with PKCE** - Prevents authorization code interception
2. **Short-lived access tokens** - Reduces window for stolen token abuse
3. **NO token passthrough** - NEVER accept tokens not issued for your server
4. **Validate token audience** - Ensure token was issued for your server
5. **Session binding** - Tie sessions to UserAgent + IP to prevent hijacking

**Security Statistics (2025)**:
- 88% of MCP servers require credentials
- 53% rely on insecure long-lived static secrets (API keys, PATs)
- Only 8.5% use modern OAuth - **major security risk**

### 2.2 Input Validation

**Comprehensive Validation Strategy**

```go
import "github.com/go-playground/validator/v10"

var validate *validator.Validate

func init() {
    validate = validator.New(validator.WithRequiredStructEnabled())

    // Register custom validators
    validate.RegisterValidation("safepath", validateSafePath)
    validate.RegisterValidation("docid", validateDocID)
}

// Prevent path traversal
func validateSafePath(fl validator.FieldLevel) bool {
    path := fl.Field().String()
    if path == "" {
        return true
    }

    // Reject path traversal attempts
    if strings.Contains(path, "..") {
        return false
    }

    // Reject absolute paths
    if filepath.IsAbs(path) {
        return false
    }

    // Allow only safe characters
    safePattern := regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
    return safePattern.MatchString(path)
}

// Sanitize filenames
func SanitizeFilename(filename string) string {
    // Remove path separators
    filename = strings.ReplaceAll(filename, "/", "_")
    filename = strings.ReplaceAll(filename, "\\", "_")

    // Remove dangerous characters
    dangerous := []string{"..", "~", "$", "`", "|", ";", "&", ">", "<"}
    for _, char := range dangerous {
        filename = strings.ReplaceAll(filename, char, "_")
    }

    // Limit length to prevent buffer overflows
    if len(filename) > 200 {
        filename = filename[:200]
    }

    return filename
}
```

**Common Vulnerabilities (2025 Statistics)**:
- 43% of MCP servers vulnerable to **Command Injection**
- 30% vulnerable to **Server-Side Request Forgery (SSRF)**
- 22% vulnerable to **Path Traversal**

**Defense Strategy**:
- Treat all AI-generated inputs as potentially malicious
- Use allowlists for structured fields (not denylists)
- Validate data types and formats
- Reject payloads exceeding size limits
- Strip or escape control characters

### 2.3 Rate Limiting

**Multi-Level Rate Limiting Implementation**

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    // Per-client limiters
    visitors    sync.Map

    // Per-tool limiters (global)
    toolLimiters map[string]*rate.Limiter

    // Configuration
    defaultRPS  float64
    defaultBurst int
}

func NewRateLimiter() *RateLimiter {
    rl := &RateLimiter{
        defaultRPS:   10.0,  // 10 requests/second
        defaultBurst: 20,    // Burst of 20
        toolLimiters: map[string]*rate.Limiter{
            // Expensive operations get stricter limits
            "export_doc":    rate.NewLimiter(0.1, 2),   // 6/min, burst 2
            "delete_records": rate.NewLimiter(0.5, 5),  // 30/min, burst 5

            // Read operations more permissive
            "list_orgs":      rate.NewLimiter(2, 20),   // 2/sec, burst 20
            "get_doc":        rate.NewLimiter(1, 10),   // 1/sec, burst 10
        },
    }

    go rl.cleanupVisitors()
    return rl
}

func (rl *RateLimiter) Allow(clientID, toolName string) error {
    // Check per-client limit
    clientLimiter := rl.getClientLimiter(clientID)
    if !clientLimiter.Allow() {
        return errors.New("client rate limit exceeded")
    }

    // Check per-tool limit (if exists)
    if toolLimiter, exists := rl.toolLimiters[toolName]; exists {
        if !toolLimiter.Allow() {
            return fmt.Errorf("tool %s rate limit exceeded", toolName)
        }
    }

    return nil
}

func (rl *RateLimiter) getClientLimiter(clientID string) *rate.Limiter {
    if limiter, exists := rl.visitors.Load(clientID); exists {
        return limiter.(*rate.Limiter)
    }

    limiter := rate.NewLimiter(rate.Limit(rl.defaultRPS), rl.defaultBurst)
    rl.visitors.Store(clientID, limiter)
    return limiter
}

// Cleanup stale client limiters
func (rl *RateLimiter) cleanupVisitors() {
    ticker := time.NewTicker(15 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        // Implementation depends on last-access tracking
    }
}
```

**Recommended Limits**:
- General operations: 120 requests/minute (2/second)
- Export operations: 6 requests/minute
- Delete operations: 30 requests/minute
- List operations: 60-120 requests/minute

### 2.4 Secret Management

**Protected Memory with memguard**

```go
import "github.com/awnumar/memguard"

var gristToken *memguard.Enclave

func LoadCredentials() error {
    token := os.Getenv("GRIST_TOKEN")
    if token == "" {
        return errors.New("GRIST_TOKEN not configured")
    }

    // Store in encrypted memory
    gristToken = memguard.NewEnclave([]byte(token))

    // CRITICAL: Clear from environment
    os.Unsetenv("GRIST_TOKEN")

    return nil
}

func UseToken(fn func(token []byte) error) error {
    lockedBuf, err := gristToken.Open()
    if err != nil {
        return err
    }
    defer lockedBuf.Destroy()

    return fn(lockedBuf.Bytes())
}

// Redact secrets in logs
var sensitivePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]+`),
    regexp.MustCompile(`(?i)(token|key|secret|password)\s*[:=]\s*['"]?[^\s'"]+`),
}

func RedactSecrets(input string) string {
    for _, pattern := range sensitivePatterns {
        input = pattern.ReplaceAllString(input, "[REDACTED]")
    }
    return input
}
```

**Best Practices**:
- Store tokens in environment variables (never hardcode)
- Use memguard for runtime secret protection
- Clear environment variables after loading
- Redact secrets from ALL logs
- Rotate tokens regularly (short-lived preferred)

### 2.5 Known CVEs (2025)

| CVE | Severity | Component | Issue | Fix |
|-----|----------|-----------|-------|-----|
| CVE-2025-6514 | 9.6 Critical | mcp-remote | RCE via untrusted server | Update to v0.1.16+ |
| CVE-2025-49596 | 9.4 Critical | MCP Inspector | DNS rebinding RCE | Update to v0.14.1+ |
| CVE-2025-53967 | Critical | Framelink Figma MCP | Fallback mechanism RCE | Update to v0.6.3+ |
| CVE-2025-52882 | 8.8 High | Claude Code Extensions | WebSocket auth bypass | Apply latest patches |
| CVE-2025-53109 | 8.4 High | Filesystem MCP | Symlink bypass | Update to v0.6.3+ |

### 2.6 Security Checklist

#### Input Validation
- [ ] Use go-playground/validator for struct validation
- [ ] Implement custom validators for domain types (docid, safepath)
- [ ] Sanitize file paths and names
- [ ] Enforce input length limits
- [ ] Scan for injection patterns

#### Authentication & Authorization
- [ ] Use OAuth 2.1 with PKCE for HTTP transport
- [ ] Generate cryptographically secure session IDs
- [ ] Bind sessions to UserAgent + IP
- [ ] NEVER pass through tokens to downstream APIs
- [ ] Validate token audience matches your server

#### Rate Limiting
- [ ] Implement per-client rate limiting
- [ ] Apply stricter limits on expensive operations (export, delete)
- [ ] Log rate limit violations
- [ ] Consider dynamic deny lists for abusive clients

#### Credential Management
- [ ] Load credentials from environment variables
- [ ] Protect secrets in memory with memguard
- [ ] Clear environment variables after loading
- [ ] Redact secrets from all logs
- [ ] Implement token rotation

#### Transport Security
- [ ] Use TLS 1.3 minimum for HTTP
- [ ] Apply security headers (X-Frame-Options, CSP, HSTS)
- [ ] Validate Origin headers for SSE
- [ ] Bind local servers to localhost only

---

## 3. MCP Performance Optimization

### 3.1 HTTP Client Reuse

**Critical Performance Issue**

Creating a new `http.Client` per request prevents TCP connection reuse, causing massive performance degradation.

```go
var (
    httpClient *http.Client
    clientOnce sync.Once
)

func getHTTPClient() *http.Client {
    clientOnce.Do(func() {
        httpClient = &http.Client{
            Transport: &http.Transport{
                // Connection pooling
                MaxIdleConnsPerHost:   10,
                MaxIdleConns:          100,
                IdleConnTimeout:       90 * time.Second,

                // Timeouts
                TLSHandshakeTimeout:   10 * time.Second,
                ExpectContinueTimeout: 1 * time.Second,
                ResponseHeaderTimeout: 30 * time.Second,

                // Keep-alive
                DisableKeepAlives: false,
            },
            Timeout: 30 * time.Second,
        }
    })
    return httpClient
}

// CRITICAL: Always read body completely for connection reuse
func makeRequest(req *http.Request) ([]byte, error) {
    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // MUST read body completely
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    return body, nil
}
```

**Performance Impact**: Proper connection pooling can reduce latency by 50-80% for multiple requests.

**Current gristle Issue**: The gristapi package likely creates clients per-request. Needs investigation.

### 3.2 Caching Strategies

**Multi-Level Caching for Read-Heavy Operations**

```go
import (
    "github.com/dgraph-io/ristretto"
    "time"
)

type CacheManager struct {
    l1Cache *ristretto.Cache  // In-memory cache
    ttl     time.Duration
}

func NewCacheManager() (*CacheManager, error) {
    cache, err := ristretto.NewCache(&ristretto.Config{
        NumCounters: 1e7,     // Number of keys to track frequency
        MaxCost:     1 << 30, // Maximum cost of cache (1GB)
        BufferItems: 64,      // Number of keys per Get buffer
    })
    if err != nil {
        return nil, err
    }

    return &CacheManager{
        l1Cache: cache,
        ttl:     15 * time.Minute,
    }, nil
}

func (cm *CacheManager) GetOrFetch(key string, fetch func() (interface{}, error)) (interface{}, error) {
    // Try L1 cache
    if val, found := cm.l1Cache.Get(key); found {
        return val, nil
    }

    // Cache miss - fetch
    val, err := fetch()
    if err != nil {
        return nil, err
    }

    // Store in cache with TTL
    cm.l1Cache.SetWithTTL(key, val, 1, cm.ttl)

    return val, nil
}

// Example usage in MCP tool
func (s *MCPServer) handleListOrgs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    cacheKey := "orgs:list"

    orgs, err := s.cache.GetOrFetch(cacheKey, func() (interface{}, error) {
        return gristapi.GetOrgs(), nil
    })
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // ... format response
}
```

**Performance Testing Results**:
- Cache hits: 15.71ms average
- API calls: 648.84ms average
- **Speedup: 41.31x**

**Cache Implementation Guidelines**:
- Use caching for read-heavy operations (list_orgs, list_workspaces, prompts/list)
- DO NOT cache side-effect operations (delete_records, create, update)
- Implement cache invalidation on mutations
- Consider Redis/Memcached for distributed deployments

### 3.3 JSON Optimization

**JSON Encoding Performance Comparison**

| Method | Speed | Notes |
|--------|-------|-------|
| `json.MarshalIndent` | Baseline | Pretty-prints, slow |
| `json.Marshal` | ~1.5x faster | Standard library |
| `jsoniter` | ~2-3x faster | Drop-in replacement |
| `sonic` | ~17x faster | ByteDance, requires CGO |

**Recommended: jsoniter (No CGO Required)**

```go
import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Drop-in replacement for encoding/json
func marshalResponse(v interface{}) ([]byte, error) {
    return json.Marshal(v)
}
```

**Current gristle Issue**: Uses `json.MarshalIndent` throughout, which is slower. For production, switch to `json.Marshal` or jsoniter.

### 3.4 Token Optimization

**Minimize JSON Response Size**

One of the most impactful optimizations involves trimming JSON responses to essential elements.

```go
// Before: Verbose response
type OrgInfoVerbose struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Domain      string `json:"domain"`
    Owner       User   `json:"owner"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
    Billing     Billing `json:"billing"`
    // ... many more fields
}

// After: Essential fields only
type OrgInfo struct {
    ID     int    `json:"id"`
    Name   string `json:"name"`
    Domain string `json:"domain,omitempty"`
}
```

**Benefits**:
- Faster JSON encoding/decoding
- Reduced token usage (extends AI working memory)
- Lower bandwidth consumption
- Faster response times

### 3.5 Buffered I/O (stdio Transport)

**Critical for Local MCP Servers**

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

func writeResponse(w *bufio.Writer, data []byte) error {
    _, err := w.Write(data)
    if err != nil {
        return err
    }
    // CRITICAL: Always flush
    return w.Flush()
}
```

**Performance Impact**:
- Unbuffered: ~901ms per iteration
- Buffered: ~39ms per iteration
- **Speedup: 23x**

**Note**: NEVER use fmt.Println() or log.Println() in stdio mode - it corrupts JSON-RPC messages!

### 3.6 Resource Cleanup & Connection Pooling

**Database Connection Pooling**

```go
import "database/sql"

func configureDBPool(db *sql.DB) {
    // Maximum number of open connections
    db.SetMaxOpenConns(25)

    // Maximum number of idle connections
    db.SetMaxIdleConns(10)

    // Maximum lifetime of a connection
    db.SetConnMaxLifetime(5 * time.Minute)

    // Maximum idle time for a connection
    db.SetConnMaxIdleTime(10 * time.Minute)
}
```

**Lifecycle Management with lifespan**

```go
type ServerLifespan struct {
    db    *sql.DB
    cache *CacheManager
}

func (sl *ServerLifespan) Initialize(ctx context.Context) error {
    // Initialize resources
    db, err := sql.Open("postgres", connString)
    if err != nil {
        return err
    }
    configureDBPool(db)
    sl.db = db

    cache, err := NewCacheManager()
    if err != nil {
        return err
    }
    sl.cache = cache

    return nil
}

func (sl *ServerLifespan) Cleanup() error {
    // Proper cleanup prevents resource leaks
    if sl.db != nil {
        sl.db.Close()
    }

    if sl.cache != nil {
        sl.cache.Close()
    }

    return nil
}

// Usage with context manager
func runServer(ctx context.Context) error {
    lifespan := &ServerLifespan{}

    if err := lifespan.Initialize(ctx); err != nil {
        return err
    }
    defer lifespan.Cleanup()

    // Server operation
    return server.Serve(ctx)
}
```

**Without proper cleanup**:
- Resource leaks occur
- Over thousands of sessions, infrastructure degrades
- Leads to "too many connections" errors
- Memory leaks in long-running applications

### 3.7 Performance Checklist

- [ ] Replace per-request `&http.Client{}` with singleton
- [ ] Configure connection pooling (MaxIdleConnsPerHost: 10)
- [ ] Switch `MarshalIndent` to `Marshal` or jsoniter
- [ ] Implement caching for read-heavy operations (list_orgs, list_workspaces)
- [ ] Add cache invalidation for mutations
- [ ] Use buffered I/O for stdio transport (64KB buffers)
- [ ] Minimize JSON response payload (essential fields only)
- [ ] Implement proper resource cleanup with lifespan managers
- [ ] Configure database connection pooling
- [ ] Ensure HTTP response bodies are fully read

---

## 4. MCP Testing Strategies

### 4.1 Unit Testing MCP Tools

**Test Structure**

```go
import (
    "context"
    "testing"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/stretchr/testify/assert"
)

func TestListOrgs(t *testing.T) {
    tests := []struct {
        name        string
        mockOrgs    []Org
        expectedErr bool
    }{
        {
            name: "successful list",
            mockOrgs: []Org{
                {ID: 1, Name: "Org1", Domain: "org1.com"},
                {ID: 2, Name: "Org2", Domain: "org2.com"},
            },
            expectedErr: false,
        },
        {
            name:        "empty list",
            mockOrgs:    []Org{},
            expectedErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock
            mockAPI := &MockGristAPI{
                Orgs: tt.mockOrgs,
            }

            server := NewMCPServer(WithGristAPI(mockAPI))

            // Execute
            req := mcp.CallToolRequest{
                Params: mcp.CallToolRequestParams{
                    Name: "list_orgs",
                },
            }

            result, err := server.HandleToolCall(context.Background(), req)

            // Assert
            if tt.expectedErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

### 4.2 Integration Testing

**Test with Mock MCP Client**

```go
func TestMCPIntegration(t *testing.T) {
    // Start MCP server
    server := NewMCPServer()

    // Create test client
    client := NewTestMCPClient(server)

    // Test tool discovery
    tools, err := client.ListTools()
    assert.NoError(t, err)
    assert.GreaterOrEqual(t, len(tools), 5)

    // Test tool execution
    result, err := client.CallTool("list_orgs", nil)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 4.3 Mocking Strategies

**Mock External Dependencies**

```go
type GristAPI interface {
    GetOrgs() []Org
    GetOrgWorkspaces(orgID int) []Workspace
    GetDoc(docID string) Doc
    ExportDocExcel(docID, filename string) error
}

type MockGristAPI struct {
    Orgs       []Org
    Workspaces map[int][]Workspace
    Docs       map[string]Doc
    ExportErr  error
}

func (m *MockGristAPI) GetOrgs() []Org {
    return m.Orgs
}

func (m *MockGristAPI) ExportDocExcel(docID, filename string) error {
    return m.ExportErr
}
```

**Benefits**:
- Tests remain fast and deterministic
- No external API dependencies
- Isolated from network issues
- Predictable behavior for assertions

### 4.4 Security Testing

**Input Validation Tests**

```go
func TestPathTraversalPrevention(t *testing.T) {
    tests := []struct {
        filename    string
        shouldReject bool
    }{
        {"valid.xlsx", false},
        {"../etc/passwd", true},
        {"../../secret", true},
        {"/etc/passwd", true},
        {"file;rm -rf /", true},
        {"file`whoami`", true},
    }

    for _, tt := range tests {
        t.Run(tt.filename, func(t *testing.T) {
            err := validateSafePath(tt.filename)
            if tt.shouldReject {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 4.5 Testing Checklist

- [ ] Unit tests for all tool handlers
- [ ] Integration tests for client-server communication
- [ ] Mock external dependencies (API, database)
- [ ] Test input validation edge cases
- [ ] Test rate limiting behavior
- [ ] Test authentication/authorization
- [ ] Test error handling and recovery
- [ ] Performance benchmarks
- [ ] Fuzz testing for input validators

---

## 5. Audit Logging & Observability

### 5.1 Structured Logging

**Essential Log Requirements**

MCP observability logs need to be:
- **Retrievable**: Stored in database, not just syslogs
- **Traceable**: Include Trace ID / Correlation ID for session linkage
- **Verbose**: Detailed metadata for multi-dimensional audits
- **Aggregated**: Query by session, user, server, tool, error type

```go
import (
    "go.uber.org/zap"
    "github.com/google/uuid"
)

type LogEntry struct {
    TraceID     string                 `json:"trace_id"`
    SessionID   string                 `json:"session_id"`
    UserID      string                 `json:"user_id,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
    EventType   string                 `json:"event_type"`
    ToolName    string                 `json:"tool_name,omitempty"`
    Duration    time.Duration          `json:"duration,omitempty"`
    StatusCode  int                    `json:"status_code,omitempty"`
    Error       string                 `json:"error,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type AuditLogger struct {
    logger *zap.Logger
    db     *sql.DB
}

func (al *AuditLogger) LogToolCall(ctx context.Context, entry LogEntry) error {
    // Structured logging to stdout
    al.logger.Info("tool_call",
        zap.String("trace_id", entry.TraceID),
        zap.String("session_id", entry.SessionID),
        zap.String("tool_name", entry.ToolName),
        zap.Duration("duration", entry.Duration),
        zap.Int("status_code", entry.StatusCode),
    )

    // Persist to database for audit trail
    _, err := al.db.ExecContext(ctx, `
        INSERT INTO audit_logs
        (trace_id, session_id, user_id, timestamp, event_type, tool_name, duration, status_code, error, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, entry.TraceID, entry.SessionID, entry.UserID, entry.Timestamp, entry.EventType,
       entry.ToolName, entry.Duration, entry.StatusCode, entry.Error, entry.Metadata)

    return err
}
```

### 5.2 Monitoring Middleware

```go
func (s *MCPServer) MonitoringMiddleware(next ToolHandler) ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Generate trace ID
        traceID := uuid.New().String()
        ctx = context.WithValue(ctx, "trace_id", traceID)

        start := time.Now()

        // Execute tool
        result, err := next(ctx, req)

        duration := time.Since(start)

        // Log audit entry
        entry := LogEntry{
            TraceID:    traceID,
            SessionID:  getSessionID(ctx),
            Timestamp:  start,
            EventType:  "tool_call",
            ToolName:   req.Params.Name,
            Duration:   duration,
            StatusCode: getStatusCode(result, err),
        }

        if err != nil {
            entry.Error = err.Error()
        }

        s.auditLogger.LogToolCall(ctx, entry)

        // Emit metrics
        s.metrics.RecordToolCall(req.Params.Name, duration, err)

        return result, err
    }
}
```

### 5.3 Metrics Collection

```go
import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
    toolCalls    *prometheus.CounterVec
    toolDuration *prometheus.HistogramVec
    toolErrors   *prometheus.CounterVec
}

func NewMetrics() *Metrics {
    m := &Metrics{
        toolCalls: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "mcp_tool_calls_total",
                Help: "Total number of tool calls",
            },
            []string{"tool_name", "status"},
        ),
        toolDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "mcp_tool_duration_seconds",
                Help:    "Tool execution duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"tool_name"},
        ),
        toolErrors: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "mcp_tool_errors_total",
                Help: "Total number of tool errors",
            },
            []string{"tool_name", "error_type"},
        ),
    }

    prometheus.MustRegister(m.toolCalls, m.toolDuration, m.toolErrors)
    return m
}

func (m *Metrics) RecordToolCall(toolName string, duration time.Duration, err error) {
    status := "success"
    if err != nil {
        status = "error"
        m.toolErrors.WithLabelValues(toolName, classifyError(err)).Inc()
    }

    m.toolCalls.WithLabelValues(toolName, status).Inc()
    m.toolDuration.WithLabelValues(toolName).Observe(duration.Seconds())
}
```

### 5.4 Security Monitoring

**Critical Events to Monitor**:
- Invalid access attempts
- Rate limit violations
- Session binding mismatches
- Token validation failures
- Path traversal attempts
- Command injection patterns

```go
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, eventType, description string, metadata map[string]interface{}) {
    entry := LogEntry{
        TraceID:   getTraceID(ctx),
        SessionID: getSessionID(ctx),
        Timestamp: time.Now(),
        EventType: "security_" + eventType,
        Metadata:  metadata,
    }

    al.logger.Warn("security_event",
        zap.String("event_type", eventType),
        zap.String("description", description),
        zap.Any("metadata", metadata),
    )

    // Send to SIEM
    al.sendToSIEM(entry)
}
```

### 5.5 Observability Checklist

- [ ] Implement structured logging with zap or zerolog
- [ ] Add trace IDs to all requests
- [ ] Log tool calls with duration, status, errors
- [ ] Persist audit logs to database
- [ ] Emit Prometheus metrics
- [ ] Monitor security events (invalid access, rate limits)
- [ ] Integrate with SIEM (Splunk, Azure Monitor)
- [ ] Create dashboards for tool usage, errors, performance
- [ ] Set up alerts for anomalies
- [ ] Export logs as CSV for compliance

---

## 6. Comparison with Current gristle Implementation

### 6.1 Current Architecture Analysis

**File: `/Users/bdmorin/src/github.com/bdmorin/grist-ctl/mcp/server.go`**

**Strengths**:
- Clean tool registration pattern
- Uses mark3labs/mcp-go (mature, battle-tested)
- Good separation of concerns (one function per tool)
- Proper error handling with `mcp.NewToolResultError`

**Areas for Improvement**:

#### 1. Input Validation
**Current**: Minimal validation (only required field checks)
```go
orgID, err := req.RequireInt("org_id")
if err != nil {
    return mcp.NewToolResultError("org_id is required"), nil
}
```

**Recommended**: Comprehensive validation with go-playground/validator
```go
type ListWorkspacesInput struct {
    OrgID int `validate:"required,gt=0"`
}

var input ListWorkspacesInput
if err := req.UnmarshalParams(&input); err != nil {
    return mcp.NewToolResultError("invalid input"), nil
}
if err := validate.Struct(&input); err != nil {
    return mcp.NewToolResultError(err.Error()), nil
}
```

#### 2. No Rate Limiting
**Current**: No rate limiting implemented

**Recommended**: Add rate limiter
```go
if err := rateLimiter.Allow(ctx, "export_doc"); err != nil {
    return mcp.NewToolResultError("rate limit exceeded"), nil
}
```

#### 3. No Audit Logging
**Current**: No logging or metrics

**Recommended**: Add monitoring middleware
```go
s.AddTool(tool, s.withMonitoring(handler))
```

#### 4. Filename Security Issue
**Current** (line 246-258):
```go
if filename[len(filename)-5:] != ".xlsx" {
    filename += ".xlsx"
}
```

**Problem**: Panic if filename is less than 5 characters. No path traversal prevention.

**Recommended**:
```go
filename = SanitizeFilename(req.GetString("filename", doc.Name))
if !strings.HasSuffix(filename, ".xlsx") {
    filename += ".xlsx"
}
```

#### 5. JSON Performance
**Current** (line 64, 106, 148, etc.):
```go
jsonBytes, err := json.MarshalIndent(result, "", "  ")
```

**Recommended**: Use `json.Marshal` or jsoniter for production
```go
jsonBytes, err := json.Marshal(result)
```

#### 6. No Context Timeout
**Current**: Tools execute indefinitely

**Recommended**:
```go
execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
result := gristapi.ExportDocExcel(execCtx, docID, filename)
```

#### 7. No Panic Recovery
**Current**: No panic recovery

**Recommended**: Add recovery middleware
```go
func (s *MCPServer) RecoveryMiddleware(next ToolHandler) ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("panic recovered: %v", r)
                result = mcp.NewToolResultError("internal server error")
            }
        }()
        return next(ctx, req)
    }
}
```

### 6.2 Migration Priorities

**P1 - Security (Critical)**:
1. Add input validation with go-playground/validator
2. Fix filename sanitization (path traversal prevention)
3. Implement rate limiting
4. Add audit logging for security events

**P2 - Performance**:
1. Investigate HTTP client reuse in gristapi
2. Switch to `json.Marshal` or jsoniter
3. Implement caching for list operations

**P3 - Reliability**:
1. Add panic recovery middleware
2. Add context timeouts to tool handlers
3. Implement graceful shutdown

**P4 - Observability**:
1. Add structured logging (zap)
2. Add monitoring middleware
3. Emit Prometheus metrics
4. Set up dashboards

**P5 - Code Quality**:
1. Extract handler logic into testable functions
2. Create interfaces for gristapi (mocking)
3. Add comprehensive unit tests
4. Add integration tests

---

## 7. Migration Roadmap

### Phase 1: Security Hardening (Week 1)

**Goal**: Prevent common vulnerabilities

- [ ] Add go-playground/validator dependency
- [ ] Implement custom validators (safepath, docid)
- [ ] Fix filename sanitization in export_doc
- [ ] Add path traversal tests
- [ ] Implement rate limiting with golang.org/x/time/rate
- [ ] Add rate limit tests

### Phase 2: Observability (Week 2)

**Goal**: Monitor production behavior

- [ ] Add zap structured logging
- [ ] Implement audit logging middleware
- [ ] Add trace ID generation
- [ ] Create audit_logs database table
- [ ] Add Prometheus metrics
- [ ] Create Grafana dashboards

### Phase 3: Performance (Week 3)

**Goal**: Optimize for production load

- [ ] Audit gristapi HTTP client usage
- [ ] Implement singleton HTTP client
- [ ] Switch to json.Marshal (or jsoniter)
- [ ] Implement caching for list_orgs, list_workspaces
- [ ] Add cache invalidation
- [ ] Performance benchmarks

### Phase 4: Reliability (Week 4)

**Goal**: Handle edge cases gracefully

- [ ] Add panic recovery middleware
- [ ] Add context timeouts to all tools
- [ ] Implement graceful shutdown
- [ ] Add health check endpoint
- [ ] Test failure scenarios

### Phase 5: Testing (Week 5)

**Goal**: Comprehensive test coverage

- [ ] Create gristapi interface
- [ ] Implement mock gristapi
- [ ] Unit tests for all tools (80% coverage)
- [ ] Integration tests with test client
- [ ] Fuzz testing for validators

---

## 8. Code Examples for gristle

### 8.1 Improved Tool Handler

```go
// Before
func registerExportDoc(s *server.MCPServer) {
    tool := mcp.NewTool("export_doc",
        mcp.WithDescription("Export a document to a file"),
        mcp.WithString("doc_id", mcp.Required(), mcp.Description("The document ID")),
        mcp.WithString("format", mcp.Required(), mcp.Description("Export format"), mcp.Enum("excel", "grist")),
        mcp.WithString("filename", mcp.Description("Output filename (optional, defaults to document name)")),
    )

    s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Handler logic inline
    })
}

// After
type ExportDocInput struct {
    DocID    string `json:"doc_id" validate:"required,docid"`
    Format   string `json:"format" validate:"required,oneof=excel grist"`
    Filename string `json:"filename,omitempty" validate:"omitempty,safepath,max=200"`
}

func (s *MCPServer) handleExportDoc(ctx context.Context, input ExportDocInput) (*ExportDocResult, error) {
    // Get doc name for default filename
    doc, err := s.gristAPI.GetDoc(ctx, input.DocID)
    if err != nil {
        return nil, err
    }

    filename := input.Filename
    if filename == "" {
        filename = doc.Name
    }
    filename = SanitizeFilename(filename)

    // Add extension
    ext := map[string]string{"excel": ".xlsx", "grist": ".grist"}[input.Format]
    if !strings.HasSuffix(filename, ext) {
        filename += ext
    }

    // Execute with timeout
    execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    switch input.Format {
    case "excel":
        err = s.gristAPI.ExportDocExcel(execCtx, input.DocID, filename)
    case "grist":
        err = s.gristAPI.ExportDocGrist(execCtx, input.DocID, filename)
    }

    if err != nil {
        return nil, err
    }

    return &ExportDocResult{
        Filename: filename,
        Message:  fmt.Sprintf("Document exported to %s", filename),
    }, nil
}

// Wrapper for MCP
func registerExportDoc(s *MCPServer) {
    tool := mcp.NewTool("export_doc",
        mcp.WithDescription("Export a document to a file"),
        mcp.WithString("doc_id", mcp.Required(), mcp.Description("The document ID")),
        mcp.WithString("format", mcp.Required(), mcp.Description("Export format"), mcp.Enum("excel", "grist")),
        mcp.WithString("filename", mcp.Description("Output filename (optional)")),
    )

    s.AddTool(tool, s.withMiddleware(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var input ExportDocInput
        if err := req.UnmarshalParams(&input); err != nil {
            return mcp.NewToolResultError("invalid input: " + err.Error()), nil
        }

        if err := validate.Struct(&input); err != nil {
            return mcp.NewToolResultError("validation failed: " + err.Error()), nil
        }

        result, err := s.handleExportDoc(ctx, input)
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        return mcp.NewToolResultText(result.Message), nil
    }))
}
```

### 8.2 Middleware Stack

```go
type ToolHandler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

func (s *MCPServer) withMiddleware(handler ToolHandler) ToolHandler {
    // Apply middleware in reverse order (execution is: recovery -> monitoring -> ratelimit -> handler)
    handler = s.rateLimitMiddleware(handler)
    handler = s.monitoringMiddleware(handler)
    handler = s.recoveryMiddleware(handler)
    return handler
}

func (s *MCPServer) recoveryMiddleware(next ToolHandler) ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
        defer func() {
            if r := recover(); r != nil {
                s.logger.Error("panic recovered",
                    zap.String("tool", req.Params.Name),
                    zap.Any("panic", r),
                    zap.Stack("stack"),
                )
                err = fmt.Errorf("internal server error")
                result = mcp.NewToolResultError("internal server error")
            }
        }()
        return next(ctx, req)
    }
}

func (s *MCPServer) monitoringMiddleware(next ToolHandler) ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        traceID := uuid.New().String()
        ctx = context.WithValue(ctx, "trace_id", traceID)

        start := time.Now()
        result, err := next(ctx, req)
        duration := time.Since(start)

        // Log and emit metrics
        s.auditLogger.LogToolCall(ctx, LogEntry{
            TraceID:   traceID,
            ToolName:  req.Params.Name,
            Duration:  duration,
            Error:     err,
        })

        s.metrics.RecordToolCall(req.Params.Name, duration, err)

        return result, err
    }
}

func (s *MCPServer) rateLimitMiddleware(next ToolHandler) ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        clientID := getClientID(ctx)
        if err := s.rateLimiter.Allow(clientID, req.Params.Name); err != nil {
            return mcp.NewToolResultError("rate limit exceeded"), nil
        }
        return next(ctx, req)
    }
}
```

---

## 9. Production Deployment Checklist

### Security
- [ ] OAuth 2.1 with PKCE configured
- [ ] Input validation on all tools
- [ ] Rate limiting enabled
- [ ] Secrets in protected memory (memguard)
- [ ] TLS 1.3 for HTTP transport
- [ ] Security headers configured
- [ ] Audit logging enabled
- [ ] CVE monitoring and patching process

### Performance
- [ ] HTTP client connection pooling
- [ ] JSON optimization (Marshal or jsoniter)
- [ ] Caching for read-heavy operations
- [ ] Database connection pooling
- [ ] Buffered I/O for stdio
- [ ] Resource cleanup implemented

### Reliability
- [ ] Panic recovery middleware
- [ ] Context timeouts on all tools
- [ ] Graceful shutdown
- [ ] Health check endpoint
- [ ] Circuit breakers for external APIs

### Observability
- [ ] Structured logging (zap)
- [ ] Trace IDs on all requests
- [ ] Prometheus metrics exported
- [ ] Grafana dashboards
- [ ] Alerts configured
- [ ] SIEM integration

### Testing
- [ ] Unit tests (80%+ coverage)
- [ ] Integration tests
- [ ] Security tests (injection, traversal)
- [ ] Performance benchmarks
- [ ] Load testing

---

## 10. References & Sources

### Official Documentation
- [MCP Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [Model Context Protocol GitHub](https://github.com/modelcontextprotocol/modelcontextprotocol)
- [One Year of MCP: November 2025 Spec Release](http://blog.modelcontextprotocol.io/posts/2025-11-25-first-mcp-anniversary/)
- [MCP 2025-11-25 Spec Update - WorkOS](https://workos.com/blog/mcp-2025-11-25-spec-update)

### Security
- [Understanding Authorization in MCP](https://modelcontextprotocol.io/docs/tutorials/security/authorization)
- [Securing MCP Servers: Authentication and Authorization](https://www.infracloud.io/blogs/securing-mcp-servers/)
- [OWASP CheatSheet - Securing Third-Party MCP Servers](https://genai.owasp.org/resource/cheatsheet-a-practical-guide-for-securely-using-third-party-mcp-servers-1-0/)
- [The Complete Guide to MCP Security - WorkOS](https://workos.com/blog/mcp-security-risks-best-practices)
- [State of MCP Server Security 2025](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)
- [Top MCP Security Best Practices for 2025](https://www.akto.io/blog/mcp-security-best-practices)
- [How to Build Secure Remote MCP Servers - GitHub](https://github.blog/ai-and-ml/generative-ai/how-to-build-secure-and-scalable-remote-mcp-servers/)

### Performance
- [Top 10 Advanced Techniques for Optimizing MCP Server Performance](https://superagi.com/top-10-advanced-techniques-for-optimizing-mcp-server-performance-in-2025/)
- [MCP API Gateway: Protocols, Caching, Remote Server Integration](https://www.gravitee.io/blog/mcp-api-gateway-explained-protocols-caching-and-remote-server-integration)
- [Advanced Caching Strategies for MCP Servers](https://medium.com/@parichay2406/advanced-caching-strategies-for-mcp-servers-from-theory-to-production-1ff82a594177)
- [Best Practices for Implementing MCP Caching](https://gist.github.com/eonist/16f74dea1e0110cee3ef6caff2a5856c)

### Go Implementation
- [mark3labs/mcp-go GitHub](https://github.com/mark3labs/mcp-go)
- [MCP-Go Getting Started](https://mcp-go.dev/getting-started/)
- [Build MCP Servers in Go - Complete Guide](https://mcpcat.io/guides/building-mcp-server-go/)
- [MCP SDK Comparison: Python vs TypeScript vs Go](https://www.stainless.com/mcp/mcp-sdk-comparison-python-vs-typescript-vs-go-implementations)
- [How to Implement Golang MCP](https://reliasoftware.com/blog/golang-mcp)

### Testing
- [Unit Testing MCP Servers - Complete Testing Guide](https://mcpcat.io/guides/writing-unit-tests-mcp-servers/)
- [MCP Testing Framework](https://github.com/haakco/mcp-testing-framework)
- [Model Context Protocol: A Guide for QA Teams](https://testcollab.com/blog/model-context-protocol-mcp-a-guide-for-qa-teams)

### Monitoring & Observability
- [MCP Observability - Complete Guide](https://mcpmanager.ai/blog/mcp-observability/)
- [Real-Time MCP Monitoring and Logging](https://www.stainless.com/mcp/real-time-mcp-monitoring-and-logging)
- [How to Monitor MCP Server Activity for Security Risks](https://www.datadoghq.com/blog/mcp-detection-rules/)
- [Implementing Logging & Audit Trails for Compliance](https://www.arsturn.com/blog/implementing-logging-and-audit-trails-in-your-mcp-server-for-compliance)

### Architecture
- [Prompts - Model Context Protocol](https://modelcontextprotocol.io/specification/2025-06-18/server/prompts)
- [How to Effectively Use Prompts, Resources, and Tools in MCP](https://composio.dev/blog/how-to-effectively-use-prompts-resources-and-tools-in-mcp)
- [Understanding MCP Features: Tools, Resources, Prompts, Sampling](https://workos.com/blog/mcp-features-guide)
- [MCP Architecture Deep Dive](https://www.getknit.dev/blog/mcp-architecture-deep-dive-tools-resources-and-prompts-explained)

### Resource Management
- [Configure MCP Servers for Multiple Connections](https://mcpcat.io/guides/configuring-mcp-servers-multiple-simultaneous-connections/)
- [MCP Resource Management & Cleanup Guide](https://www.arsturn.com/blog/the-down-low-on-mcp-resource-management-cleanup-a-no-nonsense-guide)

---

## 11. Summary of Recommendations for gristle

### Immediate Actions (This Sprint)

1. **Fix Security Vulnerability**: Add filename sanitization to prevent path traversal in export_doc
2. **Add Input Validation**: Implement go-playground/validator with custom validators
3. **Implement Rate Limiting**: Use golang.org/x/time/rate for per-client and per-tool limits

### Short Term (Next 2-4 Weeks)

4. **Add Audit Logging**: Implement structured logging with zap and monitoring middleware
5. **Performance Optimization**: Audit HTTP client usage in gristapi, switch to json.Marshal
6. **Add Tests**: Create gristapi interface, implement mocks, write unit tests for all tools

### Medium Term (1-2 Months)

7. **Caching**: Implement caching for list_orgs, list_workspaces with invalidation
8. **Reliability**: Add panic recovery, context timeouts, graceful shutdown
9. **Observability**: Add Prometheus metrics, Grafana dashboards, alerts

### Long Term (3+ Months)

10. **OAuth 2.1**: Implement proper authentication for HTTP transport
11. **Secret Management**: Use memguard for token protection
12. **Advanced Testing**: Integration tests, fuzz testing, performance benchmarks

---

**Research Date**: December 30, 2025
**MCP Specification Version**: 2025-11-25
**Go Version**: 1.24+
**mcp-go Library**: mark3labs/mcp-go v0.43.2
