# Go Standards (Tier 1)

## Required
- `gofmt` (automatic)
- `golangci-lint run` passes
- All exported symbols documented

## Error Handling
- Always check errors: `if err != nil`
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Never `_ = err` without `// nolint:errcheck` comment

## Common Issues
| Pattern | Problem | Fix |
|---------|---------|-----|
| `%v` for errors | Breaks error chain | Use `%w` |
| `panic()` in library | Crashes caller | Return error |
| Naked goroutine | No error handling | errgroup or channels |
| `interface{}` | Type safety loss | Use generics or specific types |

## Interfaces
- Accept interfaces, return structs
- Keep interfaces small (1-3 methods)
- Define interfaces where used, not implemented

## Documentation
- All exported symbols must have godoc comments starting with the symbol name
- Package-level doc in `doc.go` for non-trivial packages
- Include runnable `Example_*` functions in `_test.go` files
- Run `go doc ./...` to verify documentation

## Concurrency
- Always pass `context.Context` as first param
- Use `sync.Mutex` for shared state
- Prefer channels for communication
