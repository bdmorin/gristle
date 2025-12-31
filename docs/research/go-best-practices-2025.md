# Go Best Practices 2025: API Clients and Testing Patterns

**Research Date:** December 30, 2025
**Target Project:** gristctl (Grist API Client)
**Go Version:** 1.24+ (using Go 1.24.0 per go.mod)

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Go API Client Design Patterns](#go-api-client-design-patterns)
3. [HTTP Client Architecture](#http-client-architecture)
4. [Error Handling Patterns](#error-handling-patterns)
5. [Testing Best Practices](#testing-best-practices)
6. [Project Structure & Organization](#project-structure--organization)
7. [Observability & Logging](#observability--logging)
8. [Dependency Injection](#dependency-injection)
9. [Generics Usage Guidelines](#generics-usage-guidelines)
10. [Analysis of Current gristctl Implementation](#analysis-of-current-gristctl-implementation)
11. [Recommendations for gristctl](#recommendations-for-gristctl)
12. [Migration Path](#migration-path)
13. [References](#references)

---

## Executive Summary

Modern Go API client development in 2025 emphasizes **simplicity first, complexity when necessary**. The landscape has matured significantly with:

- **Native structured logging** via `log/slog` (Go 1.21+)
- **Native fuzzing support** in the testing package (Go 1.18+)
- **Improved benchmarking** with `testing.B.Loop()` (Go 1.24+)
- **Refined error handling** patterns with multi-error support (Go 1.20+)
- **Generics best practices** solidifying after initial introduction

Key architectural patterns prioritize **testability, observability, and resilience** while avoiding premature optimization and over-engineering.

---

## Go API Client Design Patterns

### Core Design Patterns for 2025

Modern Go API clients follow five essential patterns:

#### 1. **Repository Pattern**
Abstracts data access, making code flexible and testable. Prevents database/API access code from scattering throughout the application.

**Benefits:**
- Loose coupling between business logic and data access
- Easy to swap implementations (mock for testing, different backends)
- Single source of truth for data operations

#### 2. **Service Pattern**
Encapsulates business logic separately from HTTP/transport concerns.

**Structure:**
```go
// Service layer handles business logic
type UserService interface {
    GetUser(ctx context.Context, id string) (*User, error)
    CreateUser(ctx context.Context, user *User) error
}

// Repository layer handles data access
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

#### 3. **Handler Pattern**
Provides structured HTTP request processing with clear separation of concerns.

**Pattern:**
- Parse and validate request
- Call service layer
- Format and return response
- Handle errors consistently

#### 4. **Middleware Pattern**
Centralizes cross-cutting concerns (logging, auth, rate limiting) keeping handlers focused.

#### 5. **Context Pattern**
Robust request lifecycle management with cancellation, timeouts, and deadline propagation.

**Key Principle:** *"Accept interfaces, return structs"* - This allows consistent abstraction of dependencies while maintaining concrete return types for clarity.

### Architectural Principles

**Decoupling Goals:**
- Business logic should NOT be bound to specific databases/APIs
- System should swap MySQL, PostgreSQL, MongoDB, DynamoDB without breaking logic
- Implementation details hidden behind interfaces

**Resilience Patterns:**
- **Circuit Breaker**: Design systems to be fault-tolerant and withstand service failures
- Wrap all external calls: database, Redis, API calls
- Prevents cascading failures in distributed systems

---

## HTTP Client Architecture

### Production-Ready HTTP Client Patterns

#### HTTP Client Configuration

**Best Practice:** Reuse HTTP clients, configure connection pooling

```go
// Good: Reusable client with proper configuration
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false,
    },
}

// Bad: Creating new client per request
func makeRequest() {
    client := &http.Client{} // Creates new connection each time
    // ...
}
```

#### Context Integration

**Pattern:** Always pass `context.Context` as first argument to enable cancellation and timeouts.

```go
func (c *APIClient) GetResource(ctx context.Context, id string) (*Resource, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.buildURL(id), nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()

    // Return immediately when context is cancelled
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // continue processing
    }

    // ...
}
```

#### Retry and Backoff Strategies

**Popular Libraries:**

1. **HashiCorp go-retryablehttp**
   - Automatic retries for connection errors and 500-range responses (except 501)
   - Exponential backoff with jitter
   - Request body "rewinding" for POST/PUT operations

2. **ProjectDiscovery retryablehttp-go**
   - FullJitterBackoff: capped exponential backoff without floating point arithmetic
   - Fast, efficient implementation

**Implementation Pattern:**
```go
import "github.com/hashicorp/go-retryablehttp"

client := retryablehttp.NewClient()
client.RetryMax = 3
client.RetryWaitMin = 1 * time.Second
client.RetryWaitMax = 10 * time.Second

resp, err := client.Get("https://api.example.com/resource")
```

#### Rate Limiting

**Recommended:** `golang.org/x/time/rate` - Token bucket implementation

```go
import "golang.org/x/time/rate"

type RateLimitedClient struct {
    client  *http.Client
    limiter *rate.Limiter
}

func NewRateLimitedClient(requestsPerSecond int) *RateLimitedClient {
    return &RateLimitedClient{
        client:  &http.Client{},
        limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
    }
}

func (c *RateLimitedClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Wait blocks until request can proceed or context is cancelled
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit wait: %w", err)
    }
    return c.client.Do(req)
}
```

**Methods:**
- `Allow()`: Drop/skip events exceeding rate limit
- `Reserve()`: Wait and slow down without dropping events
- `Wait()`: Respect deadline or cancel delay via context

#### Request/Response Middleware

**Pattern:** Chain middleware for cross-cutting concerns

```go
type Middleware func(http.RoundTripper) http.RoundTripper

func ChainMiddleware(base http.RoundTripper, middlewares ...Middleware) http.RoundTripper {
    for i := len(middlewares) - 1; i >= 0; i-- {
        base = middlewares[i](base)
    }
    return base
}

// Example: Logging middleware
func LoggingMiddleware(logger *slog.Logger) Middleware {
    return func(next http.RoundTripper) http.RoundTripper {
        return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
            start := time.Now()
            logger.Info("request started",
                slog.String("method", req.Method),
                slog.String("url", req.URL.String()),
            )

            resp, err := next.RoundTrip(req)

            logger.Info("request completed",
                slog.Duration("duration", time.Since(start)),
                slog.Int("status", resp.StatusCode),
            )
            return resp, err
        })
    }
}
```

### Popular HTTP Client Libraries

#### Comparison: Resty vs Heimdall vs Gentleman

| Library | Best For | Key Features | Go Version |
|---------|----------|--------------|------------|
| **Resty** | Medium projects needing simplicity + features | Chaining API, HTTP/2, middleware, retry/timeout built-in | 1.23+ |
| **Heimdall** | Microservices, fault-tolerant systems | Circuit breaker, retries, timeouts, resilience-focused | Any |
| **Gentleman** | Plugin-driven extensibility | Full-featured plugin system | Any |

**Recommendation for 2025:** Resty or standard library with custom middleware for most projects; Heimdall for microservices requiring circuit breakers.

### Real-World API Client Patterns

#### Stripe Go SDK Pattern

**Key Innovation:** Client-based pattern (introduced v82.1)

```go
// New pattern (recommended)
client := &stripe.Client{
    Key: "sk_test_...",
}
customer, err := customer.Get("cus_123", &stripe.CustomerParams{})

// Supports multiple keys
client1 := &stripe.Client{Key: key1}
client2 := &stripe.Client{Key: key2}
```

**Legacy pattern (deprecated):** Resource pattern using global client

#### AWS SDK for Go Pattern

**Pattern:** Session-based shared configuration

```go
sess := session.Must(session.NewSession(&aws.Config{
    Region: aws.String("us-west-2"),
}))

// Share session across service clients
s3Client := s3.New(sess)
dynamoClient := dynamodb.New(sess)
```

**Key Features:**
- Base operations
- Request-suffixed methods for manual request construction
- Pages methods for automatic pagination

---

## Error Handling Patterns

### Modern Error Handling (Go 1.13 - 1.24)

#### Error Wrapping Evolution

**Go 1.13:** Introduced `errors.Is`, `errors.As`, `fmt.Errorf` with `%w`

```go
if err != nil {
    return fmt.Errorf("failed to fetch user: %w", err)
}
```

**Go 1.20:** Added `errors.Join` for multiple errors

```go
err1 := validateName(user.Name)
err2 := validateEmail(user.Email)
if err1 != nil || err2 != nil {
    return errors.Join(err1, err2)
}
```

**Go 1.23:** Iterator support affects error handling in loops

#### Best Practices for 2025

**1. Always Wrap Errors for Context**

```go
// Good: Breadcrumb trail through call stack
func GetUser(id string) (*User, error) {
    user, err := db.FindUser(id)
    if err != nil {
        return nil, fmt.Errorf("get user %s: %w", id, err)
    }
    return user, nil
}

// Bad: Loses context
func GetUser(id string) (*User, error) {
    return db.FindUser(id) // No context about what failed
}
```

**2. Use Sentinel Errors for Expected Conditions**

```go
var (
    ErrNotFound      = errors.New("resource not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrRateLimited   = errors.New("rate limit exceeded")
)

// Check with errors.Is
if errors.Is(err, ErrNotFound) {
    // Handle not found case
}
```

**3. Custom Error Types for Rich Information**

```go
type APIError struct {
    StatusCode int
    Message    string
    Endpoint   string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %d %s", e.Endpoint, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.Err
}

// Usage with errors.As
var apiErr *APIError
if errors.As(err, &apiErr) {
    log.Printf("API error: status=%d endpoint=%s", apiErr.StatusCode, apiErr.Endpoint)
}
```

**4. Multi-Error Handling (Go 1.20+)**

```go
// With multiple %w verbs or errors.Join
err := fmt.Errorf("validation failed: %w and %w", errName, errEmail)

// Unwrap returns []error
type multiErr interface {
    Unwrap() []error
}

// errors.Is and errors.As traverse all errors (pre-order, depth-first)
```

#### Production-Ready Error Handling

```go
type Client struct {
    httpClient *http.Client
    baseURL    string
    logger     *slog.Logger
}

func (c *Client) GetResource(ctx context.Context, id string) (*Resource, error) {
    url := fmt.Sprintf("%s/resources/%s", c.baseURL, id)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request for resource %s: %w", id, err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        c.logger.Error("http request failed",
            slog.String("url", url),
            slog.String("error", err.Error()),
        )
        return nil, fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, &APIError{
            StatusCode: resp.StatusCode,
            Message:    string(body),
            Endpoint:   url,
        }
    }

    var resource Resource
    if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &resource, nil
}
```

---

## Testing Best Practices

### Table-Driven Tests (Modern Patterns)

#### Core Structure

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name    string
        user    User
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid user",
            user:    User{Name: "Alice", Email: "alice@example.com"},
            wantErr: false,
        },
        {
            name:    "missing email",
            user:    User{Name: "Bob"},
            wantErr: true,
            errMsg:  "email required",
        },
        {
            name:    "invalid email format",
            user:    User{Name: "Charlie", Email: "invalid"},
            wantErr: true,
            errMsg:  "invalid email",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateUser(tt.user)

            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateUser() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if tt.wantErr && err.Error() != tt.errMsg {
                t.Errorf("ValidateUser() error message = %v, want %v", err.Error(), tt.errMsg)
            }
        })
    }
}
```

#### Best Practices for 2025

**1. Always Use t.Run() for Subtests**

Benefits:
- Better test organization
- Selective test execution: `go test -run TestUserValidation/valid_user`
- Parallel execution support
- Clear failure reporting

**2. Parallel Test Execution**

```go
func TestConcurrentOperations(t *testing.T) {
    tests := []struct {
        name string
        op   func() error
    }{
        {"operation1", func() error { /* ... */ }},
        {"operation2", func() error { /* ... */ }},
    }

    for _, tt := range tests {
        tt := tt // Capture range variable (Go 1.22+ not needed)
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Runs concurrently with other parallel tests

            if err := tt.op(); err != nil {
                t.Errorf("operation failed: %v", err)
            }
        })
    }
}
```

**Control parallelism:**
- Default: `GOMAXPROCS` (number of CPU cores)
- Override: `go test -parallel 4`

**3. Map vs Slice for Test Cases**

**Map advantage:** Undefined iteration order exposes faulty test setups where tests only pass in specific order.

```go
func TestWithMap(t *testing.T) {
    tests := map[string]struct {
        input string
        want  string
    }{
        "uppercase": {input: "hello", want: "HELLO"},
        "lowercase": {input: "WORLD", want: "world"},
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

**4. Avoid t.Fatalf in Table Tests**

```go
// Bad: Stops after first failure
if got != tt.want {
    t.Fatalf("got %v, want %v", got, tt.want)
}

// Good: Reports all failures
if got != tt.want {
    t.Errorf("got %v, want %v", got, tt.want)
}
```

**5. Test Exported Functions First**

Focus on public API behavior, not implementation details.

**6. Keep Tests Simple**

Each test should verify one thing. Avoid complex setup unless necessary.

**7. Use -race Flag**

```bash
go test -race ./...
```

Catches data races early in development.

### Mocking Strategies for 2025

#### Interface-Based Mocking

**Pattern:** "Accept interfaces, return structs"

```go
// Define interface for dependency
type UserRepository interface {
    GetUser(ctx context.Context, id string) (*User, error)
}

// Real implementation
type PostgresUserRepository struct {
    db *sql.DB
}

func (r *PostgresUserRepository) GetUser(ctx context.Context, id string) (*User, error) {
    // Real database query
}

// Mock implementation
type MockUserRepository struct {
    GetUserFunc func(ctx context.Context, id string) (*User, error)
}

func (m *MockUserRepository) GetUser(ctx context.Context, id string) (*User, error) {
    if m.GetUserFunc != nil {
        return m.GetUserFunc(ctx, id)
    }
    return nil, errors.New("not implemented")
}

// Test usage
func TestUserService(t *testing.T) {
    mockRepo := &MockUserRepository{
        GetUserFunc: func(ctx context.Context, id string) (*User, error) {
            return &User{ID: id, Name: "Test User"}, nil
        },
    }

    service := NewUserService(mockRepo)
    user, err := service.GetUser(context.Background(), "123")
    // Assertions...
}
```

#### Popular Mocking Frameworks

| Framework | Best For | Key Features | Adoption Impact |
|-----------|----------|--------------|-----------------|
| **Testify** | Simplicity, assertions | Clean API, assertion helpers | Most widely used |
| **GoMock** | Strict mocking, type safety | Code generation, strict verification | 45% testing time reduction |
| **Ginkgo/Gomega** | BDD-style tests | Behavior-driven development | Readable test specs |

#### Contract Tests vs Scenario Mocks

**Modern Pattern (2025):** "Tactical Pairs"

- **Contract Tests**: Verify interfaces work correctly (truth)
- **Scenario Mocks**: Test business logic with various states (logic)

**Example:**
```go
// Contract test: Verify repository implementation
func TestUserRepository_Contract(t *testing.T) {
    repo := NewPostgresUserRepository(testDB)

    // Save and retrieve
    user := &User{Name: "Alice"}
    err := repo.Save(context.Background(), user)
    require.NoError(t, err)

    retrieved, err := repo.GetUser(context.Background(), user.ID)
    require.NoError(t, err)
    assert.Equal(t, user.Name, retrieved.Name)
}

// Scenario mock: Test service with various repository states
func TestUserService_Scenarios(t *testing.T) {
    tests := []struct {
        name     string
        mockUser *User
        mockErr  error
        wantErr  bool
    }{
        {"user found", &User{Name: "Alice"}, nil, false},
        {"user not found", nil, ErrNotFound, true},
        {"database error", nil, errors.New("db error"), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := &MockUserRepository{
                GetUserFunc: func(ctx context.Context, id string) (*User, error) {
                    return tt.mockUser, tt.mockErr
                },
            }
            service := NewUserService(mock)
            // Test logic...
        })
    }
}
```

### Integration Testing with httptest

#### httptest.NewRecorder Pattern

**Use for:** Unit testing HTTP handlers (fast, isolated)

```go
func TestUserHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/users/123", nil)
    rr := httptest.NewRecorder()

    handler := NewUserHandler(mockService)
    handler.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)

    var user User
    json.NewDecoder(rr.Body).Decode(&user)
    assert.Equal(t, "123", user.ID)
}
```

#### httptest.Server Pattern

**Use for:** Integration tests with real server behavior

```go
func TestUserClient_Integration(t *testing.T) {
    // Start test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/users/123" {
            json.NewEncoder(w).Encode(User{ID: "123", Name: "Alice"})
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    // Test client against server
    client := NewUserClient(server.URL)
    user, err := client.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

#### TestMain Pattern (2025)

**Setup/teardown for integration tests:**

```go
var (
    testServer *httptest.Server
    testClient *APIClient
)

func TestMain(m *testing.M) {
    // Setup
    testServer = httptest.NewServer(setupRouter())
    testClient = NewAPIClient(testServer.URL)

    // Run tests
    code := m.Run()

    // Teardown
    testServer.Close()

    os.Exit(code)
}

func TestGetResource(t *testing.T) {
    // testClient is already configured
    resource, err := testClient.GetResource(context.Background(), "123")
    // Assertions...
}
```

### Benchmark Testing (Go 1.24+)

#### New testing.B.Loop() Method

**Preferred way to write benchmarks in Go 1.24:**

```go
func BenchmarkProcessData(b *testing.B) {
    data := generateTestData()

    // Setup code here (excluded from timing)

    for range b.Loop() {
        // Code to benchmark
        processData(data)
    }

    // Cleanup code here (excluded from timing)
}
```

**Advantages over old `b.N` pattern:**
- Prevents unwanted compiler optimizations
- Automatically excludes setup/cleanup from timing
- Can't accidentally depend on iteration count

#### Best Practices

**1. Use Stable Testing Environments**
- Minimal background activity
- Consistent hardware
- Disable CPU throttling

**2. Memory Profiling**
```bash
go test -bench=. -benchmem
```

Output:
```
BenchmarkProcessData-8    100000    10234 ns/op    48 B/op    2 allocs/op
```
- `48 B/op`: Average memory per operation
- `2 allocs/op`: Average allocations per operation

**3. Statistical Comparison with benchstat**
```bash
go test -bench=. -count=10 > old.txt
# Make changes
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

**4. Subbenchmarks**
```go
func BenchmarkEncode(b *testing.B) {
    data := generateData()

    b.Run("JSON", func(b *testing.B) {
        for range b.Loop() {
            json.Marshal(data)
        }
    })

    b.Run("Protobuf", func(b *testing.B) {
        for range b.Loop() {
            proto.Marshal(data)
        }
    })
}
```

**5. Integration with CI/CD**
Automatically detect performance regressions in pipeline.

### Fuzzing and Property-Based Testing

#### Native Go Fuzzing (Go 1.18+)

**Pattern:**
```go
func FuzzParseURL(f *testing.F) {
    // Seed corpus
    f.Add("https://example.com")
    f.Add("http://localhost:8080/path?query=value")

    f.Fuzz(func(t *testing.T, input string) {
        _, err := url.Parse(input)
        if err != nil {
            // Expected for invalid input
            return
        }

        // Property: valid URLs should round-trip
        parsed, _ := url.Parse(input)
        reconstructed := parsed.String()
        _, err = url.Parse(reconstructed)
        if err != nil {
            t.Errorf("round-trip failed for %q", input)
        }
    })
}
```

**Run fuzzing:**
```bash
go test -fuzz=FuzzParseURL -fuzztime=30s
```

#### Property-Based Testing

**Fuzzing vs Property-Based Testing:**

| Aspect | Fuzzing | Property-Based Testing |
|--------|---------|------------------------|
| **Focus** | Slow, multi-day brute force | Quick feedback, diverse inputs |
| **Goal** | Maximize coverage, find edge cases | Check properties on small input set |
| **Runtime** | Hours to days | Milliseconds to seconds |
| **Use Case** | Security vulnerabilities, edge cases | Development-time verification |

**Libraries:**
- `pgregory.net/rapid`: Most widely used PBT library
- `github.com/leanovate/gopter`: Full-featured, customizable

**Example with Rapid:**
```go
func TestUserValidation_Properties(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate random user
        name := rapid.String().Draw(t, "name")
        email := rapid.String().Draw(t, "email")

        user := User{Name: name, Email: email}

        // Property: validation should be deterministic
        result1 := ValidateUser(user)
        result2 := ValidateUser(user)

        if (result1 == nil) != (result2 == nil) {
            t.Fatalf("non-deterministic validation")
        }
    })
}
```

#### 2025 Testing Strategy

**Combined approach:**
1. **Example-based tests**: Common cases, regression tests
2. **Property-based tests**: During development, verify invariants
3. **Fuzzing**: CI/CD pipeline, long-running security testing

---

## Project Structure & Organization

### 2025 Principles

**Core Rule:** *"Simplicity first, complexity when necessary"*

Common mistakes:
- Over-engineering initial structure
- Deeply nested directories
- Premature abstractions

**Approach:** Start with what you need today, refactor as project grows.

### Evolution Pattern

#### Stage 1: Small Tool/Prototype
```
myproject/
├── main.go
├── go.mod
└── go.sum
```

#### Stage 2: Growing Project
```
myproject/
├── main.go
├── client.go
├── types.go
├── client_test.go
├── go.mod
└── go.sum
```

#### Stage 3: Medium Project
```
myproject/
├── cmd/
│   └── myproject/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── client.go
│   │   └── client_test.go
│   ├── models/
│   │   └── types.go
│   └── config/
│       └── config.go
├── go.mod
└── go.sum
```

### The `internal/` Directory

**Purpose:** Contains packages that cannot be imported by external projects. Go compiler enforces this restriction.

**Best Practice:** Organize by feature/domain, NOT by technical layer.

```
# Good: Domain-driven structure
internal/
├── user/
│   ├── service.go
│   ├── repository.go
│   └── handler.go
├── order/
│   ├── service.go
│   ├── repository.go
│   └── handler.go
└── payment/
    ├── service.go
    └── handler.go

# Bad: Technical layer structure
internal/
├── handlers/
│   ├── user_handler.go
│   ├── order_handler.go
│   └── payment_handler.go
├── services/
│   ├── user_service.go
│   └── order_service.go
└── repositories/
    └── user_repository.go
```

### Recommended Structure for API Clients

```
gristctl/
├── cmd/
│   └── gristctl/
│       └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   ├── client.go            # HTTP client configuration
│   │   ├── orgs.go              # Organization endpoints
│   │   ├── workspaces.go        # Workspace endpoints
│   │   ├── documents.go         # Document endpoints
│   │   ├── records.go           # Records endpoints
│   │   ├── webhooks.go          # Webhooks endpoints
│   │   └── attachments.go       # Attachments endpoints
│   ├── models/
│   │   └── types.go             # Shared types
│   ├── config/
│   │   └── config.go            # Configuration loading
│   └── middleware/
│       ├── logging.go           # Logging middleware
│       ├── retry.go             # Retry middleware
│       └── ratelimit.go         # Rate limiting middleware
├── pkg/                         # Public API (if library)
│   └── gristctl/
│       └── client.go
├── go.mod
└── go.sum
```

### Package Organization Guidelines

**1. Avoid Generic Names**

```
# Bad
utils/
helpers/
common/
base/

# Good
validator/
auth/
storage/
cache/
```

**2. Shallow Hierarchies**

Prefer 1-2 levels deep. Deep nesting increases cognitive load.

**3. Create Directories Only for New Packages**

Don't create directories just to organize `.go` files. In Go, a directory = a package.

**4. Application Directory Pattern**

```
cmd/myapp/main.go  # Keep main.go small

// main.go
func main() {
    // Import from internal/ and pkg/
    // Minimal logic here
}
```

**5. Official Go Guidance**

Place code in `internal/` to prevent external dependencies. Free to refactor without breaking external users.

---

## Observability & Logging

### Structured Logging with log/slog (Go 1.21+)

**Standard for 2025:** Native `log/slog` package replaces third-party logging libraries.

#### Key Features

1. **Structured key-value pairs**: Parseable, filterable, searchable
2. **High performance**: Optimized for common patterns (≤5 attributes = 95% of use cases)
3. **Handler-based**: TextHandler (dev), JSONHandler (production)
4. **Context support**: Propagate request context through logs

#### Basic Usage

```go
import "log/slog"

func main() {
    // Create logger with handler
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    logger.Info("user logged in",
        slog.String("user_id", "123"),
        slog.String("ip", "192.168.1.1"),
        slog.Duration("session_duration", 30*time.Minute),
    )
}
```

**Output (JSON):**
```json
{
  "time": "2025-12-30T10:30:00Z",
  "level": "INFO",
  "msg": "user logged in",
  "user_id": "123",
  "ip": "192.168.1.1",
  "session_duration": 1800000000000
}
```

#### Handler Selection Pattern

```go
func NewLogger() *slog.Logger {
    var handler slog.Handler

    if os.Getenv("ENV") == "prod" {
        // Production: JSON for log aggregation systems
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        })
    } else {
        // Development: Human-readable text
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        })
    }

    return slog.New(handler)
}
```

#### Context Methods

**Pattern:** Two sets of methods - with and without context

```go
// Without context
logger.Info("operation completed", slog.String("op", "fetch"))

// With context (propagates request ID, trace ID, etc.)
logger.InfoContext(ctx, "operation completed", slog.String("op", "fetch"))
```

#### LogValuer Interface

**Use for:** Custom types, sensitive data masking, consistent representation

```go
type User struct {
    ID       string
    Email    string
    Password string // Sensitive
}

func (u User) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("id", u.ID),
        slog.String("email", u.Email),
        // Password omitted for security
    )
}

// Usage
logger.Info("user created", slog.Any("user", user))
// Output: {"msg":"user created","user":{"id":"123","email":"user@example.com"}}
```

#### Global Logger vs Dependency Injection

**Two patterns:**

**1. Global Logger (Convenient)**
```go
var logger = slog.Default()

func ProcessRequest(r *http.Request) {
    logger.Info("processing request", slog.String("path", r.URL.Path))
}
```

**Pros:** Simple, no boilerplate
**Cons:** Hidden dependency, hard to test

**2. Dependency Injection (Testable)**
```go
type Service struct {
    logger *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
    return &Service{logger: logger}
}

func (s *Service) ProcessRequest(r *http.Request) {
    s.logger.Info("processing request", slog.String("path", r.URL.Path))
}
```

**Pros:** Explicit dependencies, easy to test
**Cons:** More verbose

**Recommendation:** Use dependency injection for services; global logger acceptable for simple utilities.

#### Production Configuration

```go
type APIClient struct {
    httpClient *http.Client
    baseURL    string
    logger     *slog.Logger
}

func (c *APIClient) GetResource(ctx context.Context, id string) (*Resource, error) {
    start := time.Now()

    c.logger.InfoContext(ctx, "fetching resource",
        slog.String("resource_id", id),
    )

    resource, err := c.fetchResource(ctx, id)

    if err != nil {
        c.logger.ErrorContext(ctx, "failed to fetch resource",
            slog.String("resource_id", id),
            slog.String("error", err.Error()),
            slog.Duration("duration", time.Since(start)),
        )
        return nil, err
    }

    c.logger.InfoContext(ctx, "resource fetched successfully",
        slog.String("resource_id", id),
        slog.Duration("duration", time.Since(start)),
    )

    return resource, nil
}
```

### OpenTelemetry Integration (Optional)

**2025 Goals for OpenTelemetry Go:**

1. **Semantic Conventions**: Weaver project for generating conventions
2. **SDK Self-Observability**: Metrics about tracing portions
3. **Go Runtime Metrics**: Opt-in (moving to opt-out)
4. **Logs API Stabilization**: Beta implementation available
5. **HTTP Instrumentation**: Stabilizing otelhttp package

**Performance Consideration:** ~35% CPU overhead, increased network traffic and latency under load

**When to use:**
- Distributed systems requiring distributed tracing
- Need for metrics, logs, traces correlation
- Integration with observability platforms (Prometheus, Jaeger, etc.)

**When to skip:**
- Simple applications
- Performance-critical paths
- Team lacks observability expertise

---

## Dependency Injection

### Patterns for 2025

**Three approaches:**

#### 1. Manual DI (Recommended for Most Projects)

**Best for:** Small to medium applications, simplicity, compile-time guarantees

```go
type UserService struct {
    repo   UserRepository
    logger *slog.Logger
}

func NewUserService(repo UserRepository, logger *slog.Logger) *UserService {
    return &UserService{
        repo:   repo,
        logger: logger,
    }
}

// Wire up in main.go
func main() {
    logger := slog.Default()
    db := setupDatabase()
    repo := NewPostgresUserRepository(db)
    service := NewUserService(repo, logger)
    handler := NewUserHandler(service)

    http.ListenAndServe(":8080", handler)
}
```

**Pros:**
- Simple, explicit
- No magic, easy to debug
- Go idiomatic
- Compile-time safety

**Cons:**
- Boilerplate in main.go
- Manual wiring can be error-prone for large apps

#### 2. Google Wire (Compile-Time)

**Best for:** Large projects with static dependency graphs

```go
// wire.go
//go:build wireinject

func InitializeServer() (*Server, error) {
    wire.Build(
        NewDatabase,
        NewUserRepository,
        NewUserService,
        NewUserHandler,
        NewServer,
    )
    return nil, nil
}

// Generated code handles wiring
// Run: wire gen ./...
```

**Pros:**
- No runtime overhead
- Type-safe
- Generated code is reviewable
- Detects circular dependencies at compile time

**Cons:**
- Code generation step
- Learning curve
- Build tool integration

#### 3. Uber Fx (Runtime)

**Best for:** Large microservices, dynamic configurations, lifecycle management

```go
func main() {
    fx.New(
        fx.Provide(
            NewDatabase,
            NewUserRepository,
            NewUserService,
            NewUserHandler,
        ),
        fx.Invoke(func(h *UserHandler) {
            http.Handle("/users", h)
        }),
    ).Run()
}
```

**Pros:**
- Lifecycle hooks (startup, shutdown)
- Modular architecture
- Powerful for complex apps
- Runtime flexibility

**Cons:**
- Runtime overhead
- More complex
- Magic can be hard to debug
- Reflection-based

### Recommendation Matrix

| Project Size | Recommendation | Reason |
|--------------|---------------|--------|
| Small (<10 packages) | Manual DI | Simple, explicit, idiomatic |
| Medium (10-50 packages) | Manual DI or Wire | Wire if boilerplate becomes painful |
| Large (50+ packages) | Wire or Fx | Wire for static graphs, Fx for dynamic/complex |
| Microservices | Fx | Lifecycle management, modularity |

---

## Generics Usage Guidelines

### Go Generics (Go 1.18+, Refined in 1.25)

**Go 1.25 Changes:**
- Removed "Core Types" concept (major syntax simplification)
- Parameterized type aliases
- Enhanced expressiveness

### When to Use Generics

**Official Guidance (go.dev):**

> "Write Go programs by writing code, not by defining types. When it comes to generics, if you start writing your program by defining type parameter constraints, you are probably on the wrong path. Start by writing functions. It's easy to add type parameters later when it's clear that they will be useful."

**Use generics when:**
1. Writing the exact same code multiple times
2. Only difference is the type used
3. Type parameters eliminate meaningful duplication

**Example: Generic Slice Functions**
```go
// Before generics: Need separate functions
func MapInt(s []int, f func(int) int) []int {
    result := make([]int, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

func MapString(s []string, f func(string) string) []string {
    result := make([]string, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}

// With generics: Single function
func Map[T any, U any](s []T, f func(T) U) []U {
    result := make([]U, len(s))
    for i, v := range s {
        result[i] = f(v)
    }
    return result
}
```

### When NOT to Use Generics

**1. Method Calls Only**

> "If all you need to do with a value of some type is call a method on that value, use an interface type, not a type parameter."

```go
// Bad: Unnecessary generic
func PrintString[T fmt.Stringer](v T) {
    fmt.Println(v.String())
}

// Good: Use interface
func PrintString(v fmt.Stringer) {
    fmt.Println(v.String())
}
```

**2. Performance Optimization**

> "Using a type parameter will generally not be faster than using an interface type. So don't change from interface types to type parameters just for speed."

**3. Premature Generalization**

Start with concrete types. Refactor to generics when duplication is clear.

### Appropriate Use Cases

**1. Data Structures**
```go
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
    if len(s.items) == 0 {
        var zero T
        return zero, false
    }
    item := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return item, true
}
```

**2. Algorithms with Multiple Type Support**
```go
func Min[T constraints.Ordered](a, b T) T {
    if a < b {
        return a
    }
    return b
}

// Works with int, float64, string, etc.
result := Min(10, 20)           // int
minFloat := Min(3.14, 2.71)     // float64
minStr := Min("apple", "banana") // string
```

**3. Type-Safe Collections**
```go
type Result[T any] struct {
    Value T
    Error error
}

func (r Result[T]) IsOK() bool {
    return r.Error == nil
}

func (r Result[T]) Unwrap() T {
    if r.Error != nil {
        panic(r.Error)
    }
    return r.Value
}
```

### Best Practices 2025

**1. Start Simple**
```go
// Start concrete
func ProcessInts(items []int) []int { /* ... */ }

// Generalize when needed
func Process[T any](items []T) []T { /* ... */ }
```

**2. Use Constraint Interfaces Wisely**
```go
// Good: Meaningful constraint
func Sum[T constraints.Integer | constraints.Float](nums []T) T {
    var sum T
    for _, n := range nums {
        sum += n
    }
    return sum
}

// Bad: Overly generic
func Sum[T any](nums []T) T { // T might not support +
    var sum T
    for _, n := range nums {
        sum += n // Compile error
    }
    return sum
}
```

**3. Prefer Simpler Solutions**
```go
// Sometimes a simple interface is better
type Comparable interface {
    Less(other Comparable) bool
}

func Sort(items []Comparable) {
    // Implementation
}
```

---

## Analysis of Current gristctl Implementation

### Current Architecture

**File:** `/Users/bdmorin/src/github.com/bdmorin/grist-ctl/gristapi/gristapi.go`

#### Strengths

1. **Comprehensive API Coverage**
   - Organizations, Workspaces, Documents
   - Records (full CRUD)
   - Attachments
   - Webhooks
   - SCIM bulk operations

2. **Good Test Coverage**
   - Table-driven tests with `httptest`
   - Mock server setup
   - Tests for Records, SCIM, Attachments, Webhooks

3. **Functional Approach**
   - Works correctly
   - Simple to understand for basic use cases

#### Areas for Improvement

### 1. **HTTP Client Reuse**

**Current Pattern:**
```go
func httpRequest(action string, myRequest string, data *bytes.Buffer) (string, int) {
    client := &http.Client{}  // Creates new client each time
    // ...
}
```

**Problem:**
- Creates new HTTP client per request
- No connection pooling
- No timeout configuration
- Inefficient for high-volume usage

**Recommended:**
```go
type Client struct {
    httpClient *http.Client
    baseURL    string
    token      string
    logger     *slog.Logger
}

func NewClient(baseURL, token string, options ...ClientOption) *Client {
    c := &Client{
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
        baseURL: baseURL,
        token:   token,
        logger:  slog.Default(),
    }

    for _, opt := range options {
        opt(c)
    }

    return c
}
```

### 2. **Context Support**

**Current:** No context usage

**Problem:**
- Cannot cancel requests
- No timeout control per request
- No trace/request ID propagation

**Recommended:**
```go
func (c *Client) GetOrgs(ctx context.Context) ([]Org, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.buildURL("orgs"), nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := c.doRequest(ctx, req)
    // ...
}
```

### 3. **Error Handling**

**Current:**
```go
if err != nil {
    log.Fatalf("Error creating request %s: %s", url, err)
}
```

**Problems:**
- `log.Fatalf` terminates program (unsuitable for library)
- Loses error context
- No error wrapping

**Recommended:**
```go
if err != nil {
    return nil, fmt.Errorf("create request for %s: %w", url, err)
}

// Define sentinel errors
var (
    ErrNotFound      = errors.New("resource not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrRateLimited   = errors.New("rate limit exceeded")
)

// Custom error type
type APIError struct {
    StatusCode int
    Message    string
    Endpoint   string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error [%d] %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.Err
}
```

### 4. **Configuration Management**

**Current:**
```go
func init() {
    GetConfig()
}

func GetConfig() string {
    home := os.Getenv("HOME")
    configFile := filepath.Join(home, ".gristle")
    if os.Getenv("GRIST_TOKEN") == "" || os.Getenv("GRIST_URL") == "" {
        err := godotenv.Load(configFile)
        // ...
    }
    return configFile
}
```

**Problems:**
- Side effects in `init()`
- Global state
- Hard to test
- Mixes config loading with client creation

**Recommended:**
```go
type Config struct {
    BaseURL string
    Token   string
}

func LoadConfig() (*Config, error) {
    cfg := &Config{}

    // Try environment variables first
    if url := os.Getenv("GRIST_URL"); url != "" {
        cfg.BaseURL = url
    }
    if token := os.Getenv("GRIST_TOKEN"); token != "" {
        cfg.Token = token
    }

    // Load from file if env vars not set
    if cfg.BaseURL == "" || cfg.Token == "" {
        home, _ := os.UserHomeDir()
        configFile := filepath.Join(home, ".gristle")
        if err := godotenv.Load(configFile); err == nil {
            cfg.BaseURL = os.Getenv("GRIST_URL")
            cfg.Token = os.Getenv("GRIST_TOKEN")
        }
    }

    if cfg.BaseURL == "" {
        return nil, errors.New("GRIST_URL not configured")
    }
    if cfg.Token == "" {
        return nil, errors.New("GRIST_TOKEN not configured")
    }

    return cfg, nil
}

// Usage in main.go
func main() {
    cfg, err := LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    client := NewClient(cfg.BaseURL, cfg.Token)
    // ...
}
```

### 5. **Structured Logging**

**Current:** Mix of `fmt.Printf` and `log.Printf`

**Recommended:** Migrate to `log/slog`

```go
type Client struct {
    // ...
    logger *slog.Logger
}

func (c *Client) GetOrgs(ctx context.Context) ([]Org, error) {
    c.logger.InfoContext(ctx, "fetching organizations")

    orgs, err := c.fetchOrgs(ctx)
    if err != nil {
        c.logger.ErrorContext(ctx, "failed to fetch organizations",
            slog.String("error", err.Error()),
        )
        return nil, err
    }

    c.logger.InfoContext(ctx, "organizations fetched",
        slog.Int("count", len(orgs)),
    )

    return orgs, nil
}
```

### 6. **Package Organization**

**Current:** Single large file (`gristapi.go` with 1468 lines)

**Recommended:** Split by domain

```
internal/api/
├── client.go           # Core client, HTTP methods
├── config.go           # Configuration
├── errors.go           # Error types
├── orgs.go             # Organization endpoints
├── workspaces.go       # Workspace endpoints
├── documents.go        # Document endpoints
├── records.go          # Records endpoints
├── attachments.go      # Attachments endpoints
├── webhooks.go         # Webhooks endpoints
├── scim.go             # SCIM endpoints
└── types.go            # Shared types
```

### 7. **Request/Response Handling**

**Current:** Direct `json.Unmarshal` in each function

**Recommended:** Centralized handling

```go
func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
    req.Header.Set("Authorization", "Bearer "+c.token)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "gristctl/1.0")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("execute request: %w", err)
    }

    if resp.StatusCode >= 400 {
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        return nil, &APIError{
            StatusCode: resp.StatusCode,
            Message:    string(body),
            Endpoint:   req.URL.Path,
        }
    }

    return resp, nil
}

func decodeJSON[T any](resp *http.Response) (T, error) {
    var result T
    defer resp.Body.Close()

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return result, fmt.Errorf("decode response: %w", err)
    }

    return result, nil
}

// Usage
func (c *Client) GetOrgs(ctx context.Context) ([]Org, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.buildURL("orgs"), nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.doRequest(ctx, req)
    if err != nil {
        return nil, err
    }

    return decodeJSON[[]Org](resp)
}
```

---

## Recommendations for gristctl

### Priority 1: Foundation (High Impact)

#### 1.1 Introduce Client Struct

**Goal:** Move from function-based API to client-based pattern

**Benefits:**
- Connection pooling
- Middleware support
- Configuration encapsulation
- Better testability

**Implementation:**
```go
type Client struct {
    httpClient *http.Client
    baseURL    string
    token      string
    logger     *slog.Logger

    // Middleware chain
    middleware []Middleware
}

type ClientOption func(*Client)

func WithLogger(logger *slog.Logger) ClientOption {
    return func(c *Client) {
        c.logger = logger
    }
}

func WithTimeout(timeout time.Duration) ClientOption {
    return func(c *Client) {
        c.httpClient.Timeout = timeout
    }
}

func WithMiddleware(m Middleware) ClientOption {
    return func(c *Client) {
        c.middleware = append(c.middleware, m)
    }
}
```

#### 1.2 Add Context Support

**Goal:** Enable request cancellation and timeout control

**Breaking Change:** Yes (changes all function signatures)

**Migration:** Introduce v2 package or major version bump

```go
// v1 (current)
func GetOrgs() []Org

// v2 (proposed)
func (c *Client) GetOrgs(ctx context.Context) ([]Org, error)
```

#### 1.3 Improve Error Handling

**Goal:** Return errors instead of `log.Fatal`, add error wrapping

**Non-breaking:** Can be done incrementally

```go
// Define error types
var (
    ErrNotFound      = errors.New("resource not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrRateLimited   = errors.New("rate limit exceeded")
)

type APIError struct {
    StatusCode int
    Message    string
    Endpoint   string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error [%d] %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.Err
}

func (e *APIError) Is(target error) bool {
    if target == ErrNotFound && e.StatusCode == 404 {
        return true
    }
    if target == ErrUnauthorized && e.StatusCode == 401 {
        return true
    }
    if target == ErrRateLimited && e.StatusCode == 429 {
        return true
    }
    return false
}
```

#### 1.4 Add Structured Logging

**Goal:** Replace fmt.Printf/log.Printf with log/slog

**Benefits:**
- Production-ready logging
- Log aggregation compatibility
- Better debugging

```go
import "log/slog"

func NewClient(baseURL, token string, opts ...ClientOption) *Client {
    c := &Client{
        logger: slog.Default(),
        // ...
    }

    for _, opt := range opts {
        opt(c)
    }

    return c
}

func (c *Client) logRequest(ctx context.Context, method, path string) {
    c.logger.DebugContext(ctx, "API request",
        slog.String("method", method),
        slog.String("path", path),
    )
}

func (c *Client) logResponse(ctx context.Context, statusCode int, duration time.Duration) {
    c.logger.InfoContext(ctx, "API response",
        slog.Int("status", statusCode),
        slog.Duration("duration", duration),
    )
}
```

### Priority 2: Resilience (Medium Impact)

#### 2.1 Add Retry Logic

**Goal:** Automatically retry transient failures

**Library:** `github.com/hashicorp/go-retryablehttp` or custom implementation

```go
type RetryConfig struct {
    MaxRetries  int
    MinWait     time.Duration
    MaxWait     time.Duration
    RetryPolicy func(resp *http.Response, err error) bool
}

func DefaultRetryPolicy(resp *http.Response, err error) bool {
    // Retry on network errors
    if err != nil {
        return true
    }

    // Retry on 5xx errors (except 501)
    if resp.StatusCode >= 500 && resp.StatusCode != 501 {
        return true
    }

    // Retry on 429 (rate limit)
    if resp.StatusCode == 429 {
        return true
    }

    return false
}
```

#### 2.2 Add Rate Limiting

**Goal:** Respect Grist API rate limits

**Library:** `golang.org/x/time/rate`

```go
import "golang.org/x/time/rate"

type Client struct {
    // ...
    rateLimiter *rate.Limiter
}

func WithRateLimit(requestsPerSecond int) ClientOption {
    return func(c *Client) {
        c.rateLimiter = rate.NewLimiter(rate.Limit(requestsPerSecond), 1)
    }
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
    if c.rateLimiter != nil {
        if err := c.rateLimiter.Wait(ctx); err != nil {
            return nil, fmt.Errorf("rate limit wait: %w", err)
        }
    }

    return c.httpClient.Do(req)
}
```

#### 2.3 Add Circuit Breaker (Optional)

**Goal:** Prevent cascading failures

**When:** Only needed if gristctl is used in production services

**Library:** `github.com/sony/gobreaker`

### Priority 3: Developer Experience (Lower Impact)

#### 3.1 Split Package by Domain

**Goal:** Improve code organization and maintainability

**Structure:**
```
internal/api/
├── client.go
├── orgs.go
├── workspaces.go
├── documents.go
├── records.go
├── attachments.go
├── webhooks.go
└── scim.go
```

#### 3.2 Add Examples and Godoc

**Goal:** Better documentation

```go
// Package api provides a client for the Grist API.
//
// Example usage:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	client := api.NewClient(cfg.BaseURL, cfg.Token)
//	ctx := context.Background()
//
//	orgs, err := client.GetOrgs(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, org := range orgs {
//	    fmt.Printf("Org: %s (ID: %d)\n", org.Name, org.Id)
//	}
package api
```

#### 3.3 Benchmark Critical Paths

**Goal:** Establish performance baselines

```go
func BenchmarkGetOrgs(b *testing.B) {
    client := setupTestClient()
    ctx := context.Background()

    for range b.Loop() {
        _, _ = client.GetOrgs(ctx)
    }
}

func BenchmarkBulkRecords(b *testing.B) {
    client := setupTestClient()
    ctx := context.Background()
    records := generateTestRecords(100)

    for range b.Loop() {
        _, _ = client.AddRecords(ctx, "doc123", "Table1", records, nil)
    }
}
```

---

## Migration Path

### Phase 1: Non-Breaking Improvements (Week 1-2)

**Goal:** Internal improvements without API changes

1. Refactor httpRequest to use persistent client
2. Add structured logging internally
3. Split large gristapi.go into domain files
4. Improve test organization

**Impact:** Zero breaking changes

### Phase 2: New v2 Package (Week 3-4)

**Goal:** Introduce modern patterns in parallel package

1. Create `internal/api/v2/`
2. Implement Client struct with context support
3. Add error types and wrapping
4. Migrate one domain (e.g., Orgs) completely
5. Write migration guide

**Impact:** New package, v1 remains unchanged

### Phase 3: Full v2 Migration (Week 5-8)

**Goal:** Complete v2 implementation

1. Migrate all domains to v2
2. Add retry logic
3. Add rate limiting
4. Comprehensive documentation
5. Update all callers (cmd/, tui/, mcp/)

**Impact:** Major version bump

### Phase 4: Deprecate v1 (Week 9-10)

**Goal:** Encourage adoption of v2

1. Mark v1 functions as deprecated
2. Add deprecation warnings
3. Update examples to use v2
4. Set sunset timeline (e.g., 6 months)

### Incremental Adoption Strategy

**For minimal disruption:**

```go
// v1: Keep existing functions
func GetOrgs() []Org {
    client := getDefaultClient()
    orgs, err := client.GetOrgs(context.Background())
    if err != nil {
        log.Fatal(err) // Maintain v1 behavior
    }
    return orgs
}

// v2: New API
func (c *Client) GetOrgs(ctx context.Context) ([]Org, error) {
    // New implementation
}
```

**Callers can migrate gradually:**

```go
// Before
orgs := GetOrgs()

// After
client := NewClient(baseURL, token)
ctx := context.Background()
orgs, err := client.GetOrgs(ctx)
if err != nil {
    // Handle error
}
```

---

## References

### Go API Client Design Patterns

- [5 API Design Patterns in Go That Solve Your Biggest Problems (2025)](https://cristiancurteanu.com/5-api-design-patterns-in-go-that-solve-your-biggest-problems-2025/)
- [Best Practices for Design Patterns in Go | Leapcell](https://leapcell.io/blog/best-practices-design-patterns-go)
- [Let's Go Further! Advanced patterns for APIs and web applications in Go](https://lets-go-further.alexedwards.net/)
- [Design Patterns in Go | tmrts/go-patterns](https://github.com/tmrts/go-patterns)

### HTTP Client Architecture

- [Go HTTP Client Patterns: A Production-Ready Implementation Guide](https://jsschools.com/golang/go-http-client-patterns-a-production-ready-implem/)
- [Building Resilient Go Services: Context, Graceful Shutdown, and Retry/Timeout Patterns](https://medium.com/@serifcolakel/building-resilient-go-services-context-graceful-shutdown-and-retry-timeout-patterns-041eea332162)
- [HashiCorp go-retryablehttp](https://github.com/hashicorp/go-retryablehttp)
- [ProjectDiscovery retryablehttp-go](https://pkg.go.dev/github.com/projectdiscovery/retryablehttp-go)

### Rate Limiting

- [Rate limiting in Golang HTTP client | MFloW](https://medium.com/mflow/rate-limiting-in-golang-http-client-a22fba15861a)
- [golang.org/x/time/rate package](https://pkg.go.dev/golang.org/x/time/rate)
- [Go Wiki: Rate Limiting](https://go.dev/wiki/RateLimiting)

### Testing Patterns

- [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [Parallel Table-Driven Tests in Go](https://www.glukhov.org/post/2025/12/parallel-table-driven-tests-in-go/)
- [Go Unit Testing: Structure & Best Practices](https://www.glukhov.org/post/2025/11/unit-tests-in-go/)
- [Advanced Go Testing Patterns](https://jsschools.com/golang/advanced-go-testing-patterns-from-table-driven-te/)

### Mocking

- [Scaling Go Testing with Contract and Scenario Mocks](https://funnelstory.ai/blog/engineering/scaling-go-testing-with-contract-and-scenario-mocks)
- [5 Mocking Techniques for Go](https://www.myhatchpad.com/insight/mocking-techniques-for-go/)
- [Testify](https://github.com/stretchr/testify)
- [GoMock](https://github.com/golang/mock)

### Integration Testing

- [Unit and Integration Testing Go Web Applications with httptest](https://leapcell.io/blog/unit-and-integration-testing-go-web-applications-with-httptest)
- [HTTP Testing in Go](https://webreference.com/go/testing/http-testing/)

### Benchmarking

- [More predictable benchmarking with testing.B.Loop](https://go.dev/blog/testing-b-loop)
- [Benchmarking in Go: A Comprehensive Handbook](https://betterstack.com/community/guides/scaling-go/golang-benchmarking/)
- [Common pitfalls in Go benchmarking](https://eli.thegreenplace.net/2023/common-pitfalls-in-go-benchmarking/)

### Fuzzing & Property-Based Testing

- [Go Testing in 2025: Mocks, Fuzzing & Property-Based Testing](https://dev.to/aleksei_aleinikov/go-testing-in-2025-mocks-fuzzing-property-based-testing-1gmg)
- [Tutorial: Getting started with fuzzing](https://go.dev/doc/tutorial/fuzz)
- [Property-Based Testing In Go: Principles And Implementation](https://volito.digital/property-based-testing-in-go-principles-and-implementation/)
- [Rapid - Property-based testing library](https://pkg.go.dev/pgregory.net/rapid)

### Error Handling

- [How to Master Error Handling in Go: Best Practices & Patterns](https://codezup.com/mastering-error-handling-in-go-best-practices/)
- [A practical guide to error handling in Go | Datadog](https://www.datadoghq.com/blog/go-error-handling/)
- [Advanced Go Error Handling: Strategies and Best Practices for 2025](https://toxigon.com/advanced-golang-error-handling)
- [errors package documentation](https://pkg.go.dev/errors)

### Project Structure

- [Go Project Structure: Practices & Patterns](https://www.glukhov.org/post/2025/12/go-project-structure/)
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- [Organizing a Go module](https://go.dev/doc/modules/layout)
- [Best Practices for Go Project Structure and Code Organization](https://medium.com/@nandoseptian/best-practices-for-go-project-structure-and-code-organization-486898990d0a)

### Structured Logging

- [Structured Logging with slog](https://go.dev/blog/slog)
- [log/slog package](https://pkg.go.dev/log/slog)
- [Logging in Go with Slog: A Practitioner's Guide](https://www.dash0.com/guides/logging-in-go-with-slog)
- [The Complete Guide to slog (Go 1.21+)](https://www.buanacoding.com/2025/09/complete-guide-slog-go-structured-logging-2025.html)
- [Golang Logging Libraries in 2025 | Uptrace](https://uptrace.dev/blog/golang-logging)

### Observability

- [OpenTelemetry Go 2025 Goals](https://opentelemetry.io/blog/2025/go-goals/)
- [Observability in Go: What Real Engineers Are Saying in 2025](https://quesma.com/blog/observability-in-go-what-real-engineers-are-saying-in-2025/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)

### Dependency Injection

- [Go Dependency Injection Approaches - Wire vs. fx, and Manual Best Practices](https://leapcell.io/blog/go-dependency-injection-approaches-wire-vs-fx-and-manual-best-practices)
- [Dependency Injection in Go: Patterns & Best Practices](https://www.glukhov.org/post/2025/12/dependency-injection-in-go/)
- [Google Wire](https://github.com/google/wire)
- [Uber Fx](https://github.com/uber-go/fx)

### Generics

- [When To Use Generics](https://go.dev/blog/when-generics)
- [Go Generics: Use Cases and Patterns](https://www.glukhov.org/post/2025/11/generics-in-go/)
- [The Go 1.25 Upgrade: Generics, Speed, and What You Need to Know](https://leapcell.io/blog/go-1-25-upgrade-guide)
- [Advanced Go Generics: Production-Ready Patterns](https://jsschools.com/golang/advanced-go-generics-production-ready-patterns-/)

### API Client Examples

- [Stripe Go SDK](https://github.com/stripe/stripe-go)
- [AWS SDK for Go](https://pkg.go.dev/github.com/aws/aws-sdk-go)
- [Resty](https://github.com/go-resty/resty)
- [Heimdall](https://github.com/gojek/heimdall)

---

**Document Version:** 1.0
**Last Updated:** December 30, 2025
**Maintainer:** Research for gristctl project
