---
sidebar_position: 3
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

```python
# Set limits on execution
slop run script.slop \
    --max-llm-calls 50 \
    --max-api-calls 1000 \
    --max-duration 5m \
    --max-cost $10.00

# Script analysis output
$ slop plan script.slop
Bounds Analysis:
  Max iterations: 1,000
  Max LLM calls: 50
  Max API calls: 200
  Estimated duration: 30s - 2m
  Estimated cost: $0.50 - $2.00
```

---

## 4. Pre-Execution Verification

### 4.1 Verification Checks

```go
func Verify(script *Script) []Issue {
    issues := []Issue{}
    
    // Syntax
    issues = append(issues, CheckSyntax(script)...)
    
    // Termination
    issues = append(issues, CheckTermination(script)...)
    
    // Type safety
    issues = append(issues, CheckTypes(script)...)
    
    // Bounds
    issues = append(issues, CheckBounds(script)...)
    
    // Dependencies
    issues = append(issues, CheckDependencies(script)...)
    
    // Security
    issues = append(issues, CheckSecurity(script)...)
    
    return issues
}
```

### 4.2 Type Checking

```python
# Schema validation for LLM calls
result = llm.call(
    prompt: "...",
    schema: {name: string, age: int}
)

# These are checked at parse time:
print(result.name)      # OK: name exists in schema
print(result.age + 1)   # OK: age is int
print(result.email)     # Error: email not in schema
print(result.age + "x") # Error: int + string
```

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

```go
type RuntimeLimits struct {
    MaxIterations   int64
    MaxLLMCalls     int64
    MaxAPICalls     int64
    MaxDuration     time.Duration
    MaxMemory       int64
    MaxOutputSize   int64
}

func Execute(script *Script, limits RuntimeLimits) (Result, error) {
    ctx := &Context{
        limits:     limits,
        iterations: 0,
        llmCalls:   0,
        apiCalls:   0,
        startTime:  time.Now(),
    }
    
    return execute(script, ctx)
}

func (ctx *Context) checkLimits() error {
    if ctx.iterations > ctx.limits.MaxIterations {
        return LimitExceeded("iterations")
    }
    if ctx.llmCalls > ctx.limits.MaxLLMCalls {
        return LimitExceeded("LLM calls")
    }
    if time.Since(ctx.startTime) > ctx.limits.MaxDuration {
        return LimitExceeded("duration")
    }
    return nil
}
```

### 5.2 Cost Tracking

```go
type CostTracker struct {
    LLMCosts    map[string]float64  // model -> cost per 1k tokens
    APICosts    map[string]float64  // service -> cost per call
    MaxCost     float64
    CurrentCost float64
}

func (t *CostTracker) RecordLLMCall(model string, tokens int) error {
    cost := t.LLMCosts[model] * float64(tokens) / 1000
    t.CurrentCost += cost
    
    if t.CurrentCost > t.MaxCost {
        return CostLimitExceeded(t.CurrentCost, t.MaxCost)
    }
    return nil
}
```

### 5.3 Rate Limiting

```python
# Automatic rate limiting
for item in items with rate(10/s):
    api.call(item)  # Automatically throttled to 10/s

# Parallel with rate limit
for item in items with parallel(5), rate(20/s):
    api.call(item)  # 5 concurrent, 20/s total
```

---

## 6. Transaction Log

### 6.1 Operation Logging

Every operation is recorded:

```go
type LogEntry struct {
    Seq       int64
    Timestamp time.Time
    Type      string        // "llm_call", "api_call", "assignment", etc.
    Location  Location      // Line/column in script
    Input     any           // Arguments
    Output    any           // Result
    Duration  time.Duration
    Cost      float64
}

type TransactionLog struct {
    ScriptHash string
    StartTime  time.Time
    Entries    []LogEntry
    Status     string  // "running", "completed", "failed", "rolled_back"
}
```

### 6.2 Checkpoints

```python
# Automatic checkpoints at loop iterations
for item in items with limit(100), checkpoint:
    result = expensive_operation(item)
    # Checkpoint saved after each iteration
    # Can resume from last checkpoint on failure

# Manual checkpoints
checkpoint("after_fetch", data)
# ... more processing ...
checkpoint("after_transform", transformed)
```

### 6.3 Rollback

```python
# Automatic rollback on failure
try:
    for item in items:
        api.update(item)  # All updates are logged
catch error:
    rollback()  # Reverts all api.update calls
    emit(error: str(error))

# Manual rollback
if not validate(results):
    rollback(to: "after_fetch")  # Rollback to checkpoint
```

---

## 7. Security

### 7.1 Sandboxing

Scripts cannot:
- Access filesystem (except through approved services)
- Make network calls (except through MCP)
- Execute system commands
- Import arbitrary code
- Access environment variables directly

### 7.2 Input Validation

```python
# Input schema validation
===INPUT===
schema: {
    query: string,
    limit: int(min: 1, max: 1000),
    options: {
        include_draft: bool
    }?  # Optional
}
===MAIN===
# input.query guaranteed to be string
# input.limit guaranteed to be 1-1000
# input.options may be none
```

### 7.3 Output Validation

```python
# Output schema validation
===OUTPUT===
schema: {
    results: list({
        id: string,
        score: float
    }),
    total: int
}
===MAIN===
# ... processing ...
emit(results: results, total: len(results))
# Validated against schema before output
```

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
# Full verification
$ slop check script.slop
✓ Syntax valid
✓ Termination guaranteed
✓ Types consistent
✓ Dependencies resolved
✓ Bounds: max 500 iterations, 50 LLM calls, 200 API calls
✓ No security issues

# Resource analysis
$ slop plan script.slop
Execution Plan:
  1. fetch contacts (1 API call)
  2. for each contact (max 100):
     a. enrich via clearbit (1 API call)
     b. enrich via linkedin (1 API call)
     c. classify via LLM (1 LLM call)
  3. emit results

Resource Bounds:
  Iterations: 100
  LLM calls: 100
  API calls: 201 (1 + 100 + 100)
  
Estimated:
  Duration: 1-5 minutes
  Cost: $1.00 - $5.00

# Dry run (no external calls)
$ slop run --dry script.slop
[DRY] Would call: salesforce.query("SELECT * FROM Contact")
[DRY] Would iterate: 100 items (limit)
[DRY] Would call: clearbit.lookup(...)
[DRY] Would call: linkedin.find(...)
[DRY] Would call: llm.call(...)
[DRY] Would emit: {results: [...], count: 100}
```

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
| Sandbox | None | Partial | **Complete** |

---

## 10. Summary

SLOP's restricted design enables:

1. **Prove termination** before running
2. **Calculate costs** before running  
3. **Verify types** before running
4. **Enforce limits** during running
5. **Rollback changes** after failure
6. **Audit everything** after completion

The restrictions aren't limitations—they're features that make verification possible.
