---
name: golang
description: Use when the project uses Go for systems programming or backend services
type: domain
domains: [backend, systems, concurrency]
agent_roles: [builder]
detect_files: ["go.mod", "*.go"]
priority: normal
version: "1.0"
---

# Go Best Practices

## Error Handling

Check every error. The `if err != nil` pattern is intentional -- Go makes error handling explicit. Never discard errors with `_`. Wrap errors with context using `fmt.Errorf("doing X: %w", err)` so call chains produce readable messages.

Use sentinel errors (`var ErrNotFound = errors.New("not found")`) for expected conditions that callers check with `errors.Is()`. Use custom error types for errors that carry structured data.

## Project Structure

Follow the standard layout: `cmd/` for application entry points, `internal/` for private packages, `pkg/` for public libraries (if any). Keep `main.go` minimal -- it wires up dependencies and starts the server.

One package per directory. Package names should be short, lowercase, and descriptive. Avoid `util` or `common` packages -- they become dumping grounds.

## Concurrency

Do not communicate by sharing memory -- share memory by communicating. Use channels for synchronization between goroutines. Use `sync.Mutex` only for protecting simple shared state where channels would be overkill.

Always ensure goroutines can be stopped. Pass `context.Context` and check `ctx.Done()`. Leaking goroutines is a memory leak that grows over time.

Use `sync.WaitGroup` when you need to wait for a known number of goroutines to complete. Use `errgroup.Group` when any goroutine failure should cancel the rest.

## Interfaces

Define interfaces where they are used, not where they are implemented. Keep interfaces small -- one or two methods is ideal. The `io.Reader` and `io.Writer` interfaces are the gold standard.

Accept interfaces, return structs. This makes code testable -- callers can inject mocks that satisfy the interface.

## Testing

Use table-driven tests for functions with multiple input/output combinations. Name test cases clearly. Use `t.Run()` for subtests so failures identify which case broke.

Use `testing.T.Helper()` in test helper functions so failure messages point to the calling test, not the helper.

## Dependencies

Use `go mod tidy` to keep `go.mod` clean. Vendor dependencies with `go mod vendor` for reproducible builds in CI. Avoid unnecessary dependencies -- the standard library is comprehensive. Check if `net/http`, `encoding/json`, or `os` already do what you need.

## Performance

Profile before optimizing with `pprof`. Pre-allocate slices with `make([]T, 0, expectedCap)` when the size is known. Avoid string concatenation in loops -- use `strings.Builder`. Reduce allocations by reusing buffers with `sync.Pool` for hot paths.
