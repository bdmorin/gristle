# MCP Server Security Best Practices

Research conducted December 2025 on security for MCP servers in Go.

## Executive Summary

MCP tools receive arbitrary input from AI models which may be influenced by prompt injection attacks. This document covers input validation, authentication, sandboxing, rate limiting, and known vulnerabilities.

---

## 1. Input Validation

### Using go-playground/validator

```go
import "github.com/go-playground/validator/v10"

var validate *validator.Validate

func init() {
    validate = validator.New(validator.WithRequiredStructEnabled())
    validate.RegisterValidation("safepath", validateSafePath)
    validate.RegisterValidation("docid", validateDocID)
}

type ExportDocInput struct {
    DocID    string `validate:"required,docid"`
    Format   string `validate:"required,oneof=excel grist"`
    Filename string `validate:"omitempty,safepath,max=255"`
}

// Custom validator: prevent path traversal
func validateSafePath(fl validator.FieldLevel) bool {
    path := fl.Field().String()
    if path == "" {
        return true
    }
    if strings.Contains(path, "..") {
        return false
    }
    if filepath.IsAbs(path) {
        return false
    }
    safePattern := regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
    return safePattern.MatchString(path)
}
```

### Filename Sanitization

```go
func SanitizeFilename(filename string) string {
    // Remove path separators
    filename = strings.ReplaceAll(filename, "/", "_")
    filename = strings.ReplaceAll(filename, "\\", "_")

    // Remove dangerous characters
    dangerous := []string{"..", "~", "$", "`", "|", ";", "&", ">", "<"}
    for _, char := range dangerous {
        filename = strings.ReplaceAll(filename, char, "_")
    }

    // Limit length
    if len(filename) > 200 {
        filename = filename[:200]
    }

    return filename
}
```

---

## 2. Rate Limiting

### Per-Client Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    visitors sync.Map
    rps      rate.Limit
    burst    int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    rl := &RateLimiter{
        rps:   rate.Limit(rps),
        burst: burst,
    }
    go rl.cleanupVisitors()
    return rl
}

func (rl *RateLimiter) Allow(clientID string) error {
    v := rl.getVisitor(clientID)
    if !v.limiter.Allow() {
        return errors.New("rate limit exceeded")
    }
    return nil
}
```

### Per-Tool Rate Limits

```go
var toolLimits = map[string]*rate.Limiter{
    "export_doc": rate.NewLimiter(0.1, 2),   // 6/min, burst 2
    "list_orgs":  rate.NewLimiter(1, 10),    // 1/sec, burst 10
}
```

---

## 3. Secure Credential Handling

### Using memguard for Protected Memory

```go
import "github.com/awnumar/memguard"

var gristToken *memguard.Enclave

func LoadCredentials() error {
    token := os.Getenv("GRIST_TOKEN")
    if token == "" {
        return errors.New("GRIST_TOKEN not configured")
    }

    gristToken = memguard.NewEnclave([]byte(token))
    os.Unsetenv("GRIST_TOKEN")  // Clear from environment
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
```

### Secret Redaction in Logs

```go
var sensitivePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]+`),
    regexp.MustCompile(`(?i)(token|key|secret)\s*[:=]\s*['"]?[^\s'"]+`),
}

func RedactSecrets(input string) string {
    for _, pattern := range sensitivePatterns {
        input = pattern.ReplaceAllString(input, "[REDACTED]")
    }
    return input
}
```

---

## 4. Authentication Patterns

### Token Passthrough Prevention

The MCP specification forbids "token passthrough" - accepting tokens not issued for the MCP server.

```go
// WRONG: Passing client token to downstream API
func badHandler(r *http.Request) {
    clientToken := r.Header.Get("Authorization")
    gristapi.SetToken(clientToken)  // SECURITY VIOLATION
}

// CORRECT: Use server's own credentials
func goodHandler(r *http.Request) {
    claims, err := validateMCPToken(r)
    if err != nil {
        return
    }
    gristapi.UseServiceCredentials()  // Server's own creds
}
```

### Session Binding

```go
type Session struct {
    ID        string
    UserID    string
    CreatedAt time.Time
    ExpiresAt time.Time
    UserAgent string
    IPAddress string
}

func (sm *SessionManager) ValidateSession(sessionID, userAgent, ip string) error {
    session, ok := sm.sessions.Load(sessionID)
    if !ok {
        return errors.New("session not found")
    }

    // Validate binding (prevents session hijacking)
    if session.UserAgent != userAgent || session.IPAddress != ip {
        return errors.New("session binding mismatch")
    }

    return nil
}
```

---

## 5. Known CVEs (2025)

| CVE | Severity | Component | Issue | Fix |
|-----|----------|-----------|-------|-----|
| CVE-2025-6514 | 9.6 Critical | mcp-remote | RCE via untrusted server | Update to v0.1.16+ |
| CVE-2025-49596 | 9.4 Critical | MCP Inspector | DNS rebinding RCE | Update to v0.14.1+ |
| CVE-2025-53967 | Critical | Framelink Figma MCP | Fallback mechanism RCE | Update to v0.6.3+ |
| CVE-2025-52882 | 8.8 High | Claude Code Extensions | WebSocket auth bypass | Apply latest patches |
| CVE-2025-53109 | 8.4 High | Filesystem MCP | Symlink bypass | Update to v0.6.3+ |

---

## 6. Tool Poisoning Prevention

### Description Hash Verification

```go
type ToolRegistry struct {
    approvedTools map[string]ToolMetadata
}

type ToolMetadata struct {
    Name            string
    DescriptionHash string
}

func (tr *ToolRegistry) ValidateTool(name, description string) error {
    approved, exists := tr.approvedTools[name]
    if !exists {
        return errors.New("tool not in approved registry")
    }

    hash := sha256.Sum256([]byte(description))
    currentHash := hex.EncodeToString(hash[:])

    if currentHash != approved.DescriptionHash {
        return errors.New("tool description modified - possible poisoning")
    }

    return nil
}
```

### Injection Pattern Detection

```go
var injectionPatterns = []struct {
    pattern string
    warning string
}{
    {`\[INST\]`, "LLAMA instruction injection"},
    {`System:`, "System prompt injection"},
    {`ignore previous`, "Instruction override attempt"},
    {`<\|im_start\|>`, "ChatML injection"},
}

func ScanForInjection(description string) []string {
    var warnings []string
    for _, p := range injectionPatterns {
        if regexp.MustCompile(`(?i)`+p.pattern).MatchString(description) {
            warnings = append(warnings, p.warning)
        }
    }
    return warnings
}
```

---

## 7. Transport Security

### TLS Configuration for HTTP

```go
func NewSecureHTTPServer(addr string, handler http.Handler) *http.Server {
    return &http.Server{
        Addr:    addr,
        Handler: handler,
        TLSConfig: &tls.Config{
            MinVersion: tls.VersionTLS13,
            CipherSuites: []uint16{
                tls.TLS_AES_256_GCM_SHA384,
                tls.TLS_CHACHA20_POLY1305_SHA256,
            },
        },
    }
}
```

### Security Headers

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Content-Security-Policy", "default-src 'none'")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000")
        next.ServeHTTP(w, r)
    })
}
```

---

## 8. Security Checklist

### Input Validation
- [ ] Use go-playground/validator for struct validation
- [ ] Implement custom validators for domain types
- [ ] Sanitize file paths and names
- [ ] Enforce input length limits

### Authentication
- [ ] Use OAuth 2.1 with PKCE for HTTP transport
- [ ] Generate cryptographically secure session IDs
- [ ] Bind sessions to user-specific information
- [ ] Never pass through tokens to downstream APIs

### Rate Limiting
- [ ] Implement per-client rate limiting
- [ ] Apply stricter limits on expensive operations
- [ ] Maintain dynamic deny lists
- [ ] Log rate limit violations

### Credential Management
- [ ] Load credentials from secure sources
- [ ] Protect secrets in memory with memguard
- [ ] Clear environment variables after loading
- [ ] Redact secrets from all logs

### Transport Security
- [ ] Use TLS 1.3 minimum for HTTP
- [ ] Apply security headers
- [ ] Validate Origin headers for SSE
- [ ] Bind local servers to localhost only

---

## Sources

- [OWASP MCP Top 10](https://owasp.org/www-project-mcp-top-10/)
- [MCP Security Best Practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)
- [State of MCP Server Security 2025](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)
- [Top 10 MCP Security Risks](https://prompt.security/blog/top-10-mcp-security-risks)
- [CVE-2025-6514 Analysis](https://jfrog.com/blog/2025-6514-critical-mcp-remote-rce-vulnerability/)
- [go-playground/validator](https://github.com/go-playground/validator)
- [MemGuard Library](https://github.com/awnumar/memguard)
