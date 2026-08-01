---
sidebar_position: 3
title: Safety & Resource Limits
description: How SLOP enforces iteration limits, rate limits, and timeouts to keep agent scripts bounded and safe.
---

# SLOP Safety Model

## Verification, Bounds, and Guarantees

---

## 1. Core Guarantees

SLOP provides three guarantees impossible in general-purpose languages:

| Guarantee | How |
|-----------|-----|
| **Termination** | No `while`, no recursion, bounded `for` |
| **Resource Bounds** | Static analysis of max operations |
| **Auditability** | Linear control flow, explicit state |

---

## 2. Termination Proof

### 2.1 No Unbounded Loops

```python
# ALLOWED: bounded iteration
for i in range(100):
    work()

for item in items with limit(1000):
    process(item)

# FORBIDDEN: unbounded iteration
while True:        # Syntax error
    work()

while not done:    # Syntax error
    work()
```

### 2.2 No Recursion

```python
# FORBIDDEN: direct recursion
def factorial(n):
    if n <= 1: return 1
    return n * factorial(n - 1)  # Error: recursive call

# FORBIDDEN: indirect recursion
def a():
    b()

def b():
    a()  # Error: cycle in call graph

# ALLOWED: same-name functions in different modules (no cycle)
```

### 2.3 Termination Analysis

```go
func VerifyTermination(ast *AST) error {
    // 1. Check no while statements
    if whiles := ast.Find(WhileStmt); len(whiles) > 0 {
        return Error("while loops not allowed")
    }
    
    // 2. Build call graph
    callGraph := BuildCallGraph(ast)
    
    // 3. Check for cycles
    if cycles := callGraph.FindCycles(); len(cycles) > 0 {
        return Error("recursive calls: %v", cycles)
    }
    
    // 4. Check all for loops are bounded
    for _, loop := range ast.Find(ForStmt) {
        if !loop.HasBound() {
            return Error("unbounded loop at line %d", loop.Line)
        }
    }
    
    return nil  // Termination guaranteed
}
```

---

## 3. Resource Bounds

### 3.1 Static Bound Calculation

Every script has a calculable maximum operation count:

```python
# Example script
for i in range(10):           # 10 iterations
    for j in range(20):       # × 20 iterations
        api.call()            # × 1 API call
                              # = 200 API calls max

for item in items with limit(100):  # 100 max
    llm.call(...)                   # × 1 LLM call
                                    # = 100 LLM calls max

# Total: 200 API + 100 LLM = 300 external calls max
```

### 3.2 Bound Analysis

```go
type ResourceBounds struct {
    MaxIterations   int64
    MaxLLMCalls     int64
    MaxAPICalls     int64
    MaxDuration     time.Duration
    MaxMemory       int64
}

func AnalyzeBounds(ast *AST) ResourceBounds {
    bounds := ResourceBounds{}
    
    for _, loop := range ast.Find(ForStmt) {
        loopBound := loop.GetBound()
        innerBounds := AnalyzeBounds(loop.Body)
        
        bounds.MaxIterations += loopBound * innerBounds.MaxIterations
        bounds.MaxLLMCalls += loopBound * innerBounds.MaxLLMCalls
        bounds.MaxAPICalls += loopBound * innerBounds.MaxAPICalls
    }
    
    for _, call := range ast.Find(LLMCall) {
        bounds.MaxLLMCalls++
    }
    
    for _, call := range ast.Find(ServiceCall) {
        bounds.MaxAPICalls++
    }
    
    return bounds
}
```

### 3.3 Pre-Execution Limits

```bash
# Set limits from the CLI (only these two flags exist today):
slop run script.slop --max-iterations 1000 --max-llm-calls 50

# Script bounds analysis
$ slop plan script.slop
Execution Plan: script.slop

Resource Bounds:
  Max iterations:  200
  Max LLM calls:   0
  Max API calls:   0

Termination: Guaranteed (no recursion detected)

Summary:
  ✓ Script is valid and will terminate
  ✓ Worst-case: 200 iterations, 0 LLM calls, 0 API calls
```

`MaxAPICalls`, `MaxDuration`, and `MaxCost` are enforced internally
(`internal/evaluator/context.go`) when set through the `pkg/slop.Config`
Go API, but the `run` CLI command only exposes `--max-iterations`,
`--max-llm-calls`, and `--checkpoint-dir` — there is no
`--max-api-calls`/`--max-duration`/`--max-cost` flag, and `slop plan`
does not estimate wall-clock duration or dollar cost.

---

## 4. Pre-Execution Verification

### 4.1 Verification Checks

`slop check <file>` (`cmd/slop/main.go` → `internal/analyzer.Analyze`)
runs, in order:

1. Lex + parse — syntax errors abort the check.
2. Termination analysis — no `while`, no direct or indirect recursion,
   every `for` loop has a static bound.
3. Bounds computation — `MaxIterations`/`MaxLLMCalls`/`MaxAPICalls`.

There is no separate type-checking pass, dependency-graph check, or
security check baked into `slop check` — the "type checking" in the next
section is enforced by the LLM service's schema validation at call time,
not by static analysis, and there is no `CheckSecurity` equivalent
anywhere in the codebase.

### 4.2 Type Checking

```python
# Schema validation for LLM calls
result = llm.call(
    prompt: "...",
    schema: {name: string, age: int}
)

print(result.name)      # OK: name exists in schema
print(result.age + 1)   # OK: age is int
print(result.email)     # No error — returns none, not in schema
print(result.age + "x") # Runtime error: unsupported operation: int + string
```

This is runtime behavior, not a parse-time check: SLOP has no static type
checker, so accessing a field the schema didn't produce silently returns
`none` rather than failing, and a bad operator combination (like `int +
string`) only errors when that line actually executes.

### 4.3 Dependency Checking

```python
===USE: lib/processor===
===USE: lib/utils===

===MAIN===
# Checked: processor exists
# Checked: all processor's dependencies satisfied
# Checked: no circular dependencies
result = processor.run(data)
```

---

## 5. Runtime Guardrails

### 5.1 Execution Limits

The real type (`internal/evaluator/context.go`) is `ExecutionLimits`, held
on the evaluator `Context` and checked as counters increment:

```go
type ExecutionLimits struct {
    MaxIterations int64
    MaxLLMCalls   int64
    MaxAPICalls   int64
    MaxDuration   int64   // seconds, not time.Duration
    MaxCost       float64
    MaxCallDepth  int64   // nested user function/lambda call depth

    // Counters, updated during execution
    IterationCount int64
    LLMCallCount   int64
    APICallCount   int64
    StartTime      int64
    TotalCost      float64
    CallDepth      int64
}
```

A limit of `0` means unlimited for that dimension — each check is
`if Limits.MaxX > 0 && Limits.XCount > Limits.MaxX { return error }`.

### 5.2 Cost Tracking

There is no separate cost-tracking type — `MaxCost` and `TotalCost` are
plain fields on `ExecutionLimits` (5.1), checked the same way as the other
limits: `if MaxCost > 0 && TotalCost > MaxCost { return error }`. Nothing
in the codebase computes a per-model or per-service cost automatically;
`TotalCost` only moves if the host application sets it explicitly.

### 5.3 Rate Limiting

```python
# Automatic rate limiting
for item in items with rate(10/s):
    api.call(item)  # Automatically throttled to 10/s
```

`parallel(n)` is parsed but not implemented yet
(`internal/evaluator/evaluator.go` has a `// TODO: Implement parallel
execution` where it should run the body concurrently) — loops with
`parallel(n)` currently execute sequentially, with no concurrency and no
combined rate limit.

---

## 6. Transaction Log

### 6.1 Operation Logging

Service calls are recorded on the evaluator `Context`
(`internal/evaluator/context.go`):

```go
type Operation struct {
    ID         int64
    Timestamp  int64
    Type       string  // "call", "write", "delete", "update", etc.
    Service    string
    Method     string
    Args       []Value
    Kwargs     map[string]Value
    Result     Value
    Error      error
    Reversible bool
}

type TransactionLog struct {
    Operations []Operation
    // ...
}
```

### 6.2 Checkpoints

There is no `checkpoint` loop modifier and no callable `checkpoint()`
function. Checkpointing is a CLI/host feature, not a script-level one: run
with `--checkpoint-dir <dir>` and resume with `slop resume <checkpoint>` —
see section 3.3 and the [Modules](/docs/language/modules) page.

### 6.3 Rollback

```python
# Automatic rollback: `stop with rollback` reverts every logged
# Reversible operation via the transaction log
try:
    for item in items:
        api.update(item)  # Logged, and reversible if the service supports it
catch error:
    emit(error: str(error))
    stop with rollback
```

There is no standalone `rollback()` function and no `rollback(to:
"checkpoint")` — the only rollback trigger is the `stop with rollback`
statement, which reverts the whole transaction log, not to a named
checkpoint.

---

## 7. Security

### 7.1 Sandboxing

There is no sandbox enforcement layer, allow-list, or security policy
engine in the codebase (no `CheckSecurity` function, no filesystem/network
policy anywhere in `internal/`). What scripts can and cannot do is a
side effect of which built-ins exist:

- No filesystem or process-execution built-ins ship by default, so
  scripts cannot touch the filesystem or run system commands unless a
  host application registers a service that does.
- `env_get(name)` is a real, registered built-in
  (`internal/builtin/control.go`) that wraps `os.Getenv` directly —
  scripts **can** read any environment variable by name. `env_mode()` and
  `env_debug()` also read `SLOP_ENV`/`ENV`/`SLOP_DEBUG` directly.
- Network access is only possible through services a host registers
  (`RegisterService`) — nothing built-in makes an HTTP call.
- There is no dynamic `import`/`eval`/`exec`, so scripts cannot load or
  run arbitrary code.

Treat this as "a small, fixed built-in surface with no network/filesystem
access by default," not as an enforced security sandbox.

### 7.2 Input Validation

`===INPUT===` and `===OUTPUT===` blocks are not implemented. The lexer
recognizes the `INPUT`/`OUTPUT` header tokens
(`internal/lexer/lexer.go`), but the module parser's `parseModule()`
(`internal/parser/parser.go`) only handles `SOURCE`, `USE`, and `MAIN` —
a script starting with `===INPUT===` fails to parse. `input` is available
as a plain global inside a script; it is whatever value the host
application sets, with no schema enforced on it.

### 7.3 Output Validation

There is no output-schema block either, and no built-in step validates an
`emit()` call's shape before it leaves the script — see the LLM call's
`schema` option (section 3) for the schema validation that does exist,
which only applies to `llm.call()` results.

### 7.4 LLM Output Sanitization

```python
# Schema enforcement prevents prompt injection via output
result = llm.call(
    prompt: user_input,  # Potentially malicious
    schema: {action: enum(search, lookup, done)}
)

# result.action is GUARANTEED to be one of: search, lookup, done
# Cannot contain arbitrary code or commands
```

---

## 8. Verification CLI

```bash
# Verification
$ slop check script.slop
✓ script.slop: valid
  Max iterations: 200

# Resource analysis
$ slop plan script.slop
Execution Plan: script.slop

Resource Bounds:
  Max iterations:  200
  Max LLM calls:   0
  Max API calls:   0

Termination: Guaranteed (no recursion detected)

Summary:
  ✓ Script is valid and will terminate
  ✓ Worst-case: 200 iterations, 0 LLM calls, 0 API calls
```

There is no `slop run --dry` dry-run mode.

---

## 9. Comparison

| Property | Python | JavaScript | SLOP |
|----------|--------|------------|------|
| Termination | Undecidable | Undecidable | **Guaranteed** |
| Max iterations | Unknown | Unknown | **Static** |
| Max API calls | Unknown | Unknown | **Static** |
| Cost estimate | Impossible | Impossible | **Calculated** |
| Rollback | Manual | Manual | **Built-in** |
| Type safety | Runtime | Runtime | **Pre-execution** |
| Sandbox | None | Partial | **No enforced sandbox — small built-in surface, no net/fs by default** |

---

## 10. Summary

SLOP's restricted design enables:

1. **Prove termination** before running
2. **Calculate costs** before running  
3. **Verify types** before running
4. **Enforce limits** during running
5. **Rollback changes** after failure
6. **Audit everything** after completion

These restrictions are what make verification possible before a script runs.
