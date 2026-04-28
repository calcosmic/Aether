---
name: rust
description: Use when the project uses Rust for systems programming, CLI tools, WebAssembly, or backend services
type: domain
domains: [backend, systems, wasm, cli]
agent_roles: [builder]
detect_files: ["Cargo.toml", "Cargo.lock", "*.rs", "rust-toolchain.toml"]
detect_packages: ["rust", "tokio", "serde", "axum", "actix-web"]
priority: normal
version: "1.0"
---

# Rust Best Practices

## Ownership and Borrowing

- Follow the ownership model: each value has a single owner, dropped when the owner goes out of scope
- Use `&T` for borrowed references; prefer `&self` over `&mut self` in method signatures
- Apply `Clone` only when necessary -- prefer references or `Copy` for small types
- Use `Rc<T>` for shared ownership in single-threaded contexts, `Arc<T>` for multi-threaded
- Avoid `RefCell<T>` in performance-critical paths; restructure data to satisfy the borrow checker statically

## Error Handling

- Use `Result<T, E>` as the standard return type for fallible operations; never panic in library code
- Define domain errors with `thiserror` for library crates: `#[derive(Error)]` with descriptive variants
- Use `anyhow` for application-level error propagation with context: `.context("failed to open config")?`
- Create a unified `AppError` type at application boundaries with `From` trait impls for conversion
- Use `unwrap()` and `expect()` only in tests or cases where failure is genuinely impossible

## Async with Tokio

- Use `tokio` as the async runtime; annotate `main` with `#[tokio::main]` and select the appropriate flavor (`flavor = "current_thread"` for lightweight, `multi_thread` for CPU-bound)
- Prefer `tokio::spawn` for concurrent tasks; use `JoinHandle` to await results
- Use `tokio::sync::Mutex` (not `std::sync::Mutex`) across `.await` points to avoid holding OS locks across yields
- Apply `tokio::select!` for racing futures and handling cancellation
- Use channels (`mpsc`, `broadcast`, `watch`) for inter-task communication over shared mutable state

## Trait Design

- Design traits for behavior, not data; keep trait methods focused and composable
- Use associated types when the return type varies by implementor: `type Output`
- Apply trait bounds at the function level, not the struct level, to maximize flexibility
- Implement `From`/`Into` for type conversions rather than ad-hoc methods
- Use `dyn Trait` for dynamic dispatch only when static dispatch with generics is impractical

## Cargo and Project Structure

- Use cargo workspaces for multi-crate projects: define `[workspace]` in the root `Cargo.toml`
- Keep binary, library, and test targets separated under `src/bin/`, `src/lib.rs`, and `tests/`
- Pin dependency versions with `Cargo.lock` for binaries; omit it for published libraries
- Use `#[cfg(test)]` modules for unit tests within source files and `tests/` directory for integration tests
- Profile for release builds: set `opt-level`, `lto = true`, and `codegen-units = 1` in `Cargo.toml` for maximum performance

## WebAssembly Targets

- Target `wasm32-unknown-unknown` for browser WASM; use `wasm-bindgen` for JS interop
- Use `wasm-pack` for building and publishing WASM packages
- Avoid `std::fs`, `std::net`, and blocking I/O in WASM targets -- use browser APIs via `web-sys`
- Keep WASM modules small: enable `wee_alloc` as the global allocator and tree-shake with LTO
- Test WASM logic in native Rust first, then verify in browser with `wasm-pack test --chrome`
