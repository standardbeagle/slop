# Refactoring Task for Ralph Loop

Execute the refactoring plan documented in `docs/REFACTORING_ANALYSIS.md`.

## Task Sequence

### PHASE 1: Walk Function (visitor.go:50) - CC=172 → ~5

**Goal:** Implement node-based traversal pattern

**Steps:**
1. Add `Walkable` interface to `internal/ast/visitor.go`
2. Implement `Walk(v Visitor) error` on Program node (proof of concept)
3. Implement on Block and IfStatement nodes
4. Run tests: `go test ./internal/ast/... -v`
5. If tests pass, implement on all remaining 37+ node types
6. Update Walk function to delegate to node.Walk()
7. Run full test suite: `go test ./...`
8. Add unit tests for individual node Walk methods

**Success Criteria:**
- All existing tests pass
- Walk function CC < 10
- Public API unchanged (Walk signature preserved)

### PHASE 2: NextToken Function (lexer.go:73) - CC=45 → ~12

**Goal:** Extract token handlers into registry pattern

**Steps:**
1. Create `TokenHandler` type: `type TokenHandler func(l *Lexer) (Token, error)`
2. Extract handlers for simple operators (+, -, *, /, etc.)
3. Create `tokenHandlers` map[rune]TokenHandler
4. Refactor NextToken to dispatch via handler map
5. Extract indentation logic into `handleIndentation()` method
6. Run tests: `go test ./internal/lexer/... -v`
7. Extract remaining complex token handlers
8. Run full lexer test suite

**Success Criteria:**
- All 19 NextToken tests pass
- NextToken CC < 15
- No performance regression

### PHASE 3: ValidateAgainstSchema (llm.go:353) - CC~18 → ~8

**Goal:** Extract type validators as helper functions

**Steps:**
1. Extract `validateString(value any, schema *Schema) error`
2. Extract `validateInteger(value any, schema *Schema) error`
3. Extract `validateNumber(value any, schema *Schema) error`
4. Extract `validateBoolean(value any, schema *Schema) error`
5. Extract `validateArray(value any, schema *Schema) error`
6. Extract `validateObject(value any, schema *Schema) error`
7. Update ValidateAgainstSchema to call helpers (switch statement)
8. Extract constraint helpers: `validateEnum`, `validateNumericConstraints`
9. Run tests: `go test ./internal/runtime/... -v`
10. Add unit tests for each validator

**Success Criteria:**
- TestValidateAgainstSchema passes
- ValidateAgainstSchema CC < 8
- All edge cases tested

## Testing Protocol

After each phase, run:
```bash
# Unit tests for modified package
go test ./internal/ast/... -v
go test ./internal/lexer/... -v
go test ./internal/runtime/... -v

# Full test suite
go test ./... -v

# Optional: Benchmarks
go test -bench=. ./internal/ast/...
go test -bench=. ./internal/lexer/...
```

## Work Incrementally

- Make small commits after each working change
- Keep all tests passing at each step
- If a test fails, fix it before proceeding
- Document changes in code comments

## Completion Signal

When all three phases are complete and all tests pass, output:

<promise>REFACTORING COMPLETE: All three functions refactored, all tests passing, benchmarks show no regression</promise>

## Important Notes

- Do NOT skip tests
- Do NOT make breaking changes to public APIs
- Do NOT remove existing functionality
- DO maintain backward compatibility
- DO preserve all existing test coverage
