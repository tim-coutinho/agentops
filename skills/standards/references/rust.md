# Rust Standards (Tier 1)

## Required
- `cargo fmt` (automatic)
- `cargo clippy` passes (no warnings)
- All public items documented (rustdoc)

## Error Handling
- Use `Result<T, E>` for fallible operations
- Implement custom errors with `thiserror` or `anyhow`
- Never `unwrap()` in library code (OK in tests/bins)
- Use `?` operator for error propagation

## Ownership & Borrowing
- Prefer references over cloning
- Use `&str` in function params over `String`
- Add explicit lifetime annotations when needed
- Clone sparingly and document why

## Common Issues
| Pattern | Problem | Fix |
|---------|---------|-----|
| `unwrap()` | Panic on None/Err | Use `?` or pattern match |
| Mutable statics | Data races | Use `once_cell` or `Mutex` |
| String allocation | Performance | Use `&str` in function params |
| Lifetime errors | Borrow checker reject | Add explicit lifetimes |
| Unsafe block | Memory unsafety | Add `// SAFETY:` comment |
| Excessive `.clone()` | Performance waste | Use references or `Cow<T>` |

## Unsafe Code
- Always add `// SAFETY:` comment explaining invariants
- Minimize unsafe scope
- Prefer safe abstractions

## Security
- Minimize `unsafe` blocks — each needs `// SAFETY:` justification
- Use `secrecy::Secret<T>` for sensitive values (prevents accidental logging)
- Validate all external input before deserialization (`serde` validators)
- Prefer `ring` or `rustls` over OpenSSL bindings

## Documentation
- All public items must have rustdoc comments (`///`)
- Include `# Examples` section in doc comments for complex APIs
- Use `#![deny(missing_docs)]` in library crates
- Run `cargo doc --no-deps` to verify doc builds

## Testing
- `cargo test` (built-in)
- `cargo test --doc` (doc tests)
- Use `#[cfg(test)]` modules
- `cargo bench` for benchmarks
