# Issues Found and Fixed Through TDD

This document tracks all issues discovered through Test-Driven Development of the SLOP chat app example.

## 1. SLOP Syntax Limitations

### Issue
SLOP scripts with triple-quoted multiline strings failed to parse with errors like:
```
parse error at 8:14: expected ), got STRING
no prefix parse function for DEDENT
```

### Root Cause
The SLOP lexer (`internal/lexer/lexer.go`) treats newlines inside strings as unterminated strings:
```go
if l.ch == 0 || l.ch == '\n' {
    // Unterminated string
```

**SLOP does NOT support multiline strings** using triple quotes (`"""`). This is by design.

### Fix
Rewrote all example scripts to use string concatenation for multi-line prompts:
```slop
# BEFORE (doesn't work)
prompt: """
Line 1
Line 2
"""

# AFTER (works)
prompt: "Line 1\n" + "Line 2"
```

### Tests
- `scripts/scripts_test.go::TestMultilineStringNotSupported` - Documents this limitation
- `scripts/scripts_test.go::TestScriptsParse` - Verifies all scripts parse

## 2. Missing `input` Variable Context

### Issue
All example scripts referenced `input.message`, `input.topic`, etc. but failed at runtime with:
```
evaluation error: undefined variable: input
```

### Root Cause
The `RunScript` method in `internal/chat/app.go` didn't provide an `input` variable in the evaluation context.

### Fix
Added empty input map to script context in `app.go:193`:
```go
// Provide empty input map by default
ctx.Scope.Set("input", evaluator.NewMapValue())
```

### Tests
- `internal/chat/app_input_test.go::TestRunScriptWithInput` - Verifies input is available
- `internal/chat/app_input_test.go::TestInputVariableProvided` - Tests `input.field or default` pattern

## 3. Missing `llm` Service

### Issue
Scripts call `llm.call(...)` but fail with:
```
evaluation error: undefined variable: llm
```

### Root Cause
The SLOP runtime creates an `llm` service (see `pkg/slop/runtime.go:36-39`), but `RunScript` creates a fresh `evaluator.Context` instead of using the runtime's context.

### Current Status
❌ **Not yet fixed** - Needs the runtime's LLM service to be properly registered in the script context.

### Proposed Fix
In `internal/chat/app.go`, use the runtime's context or ensure services are copied:
```go
// Option 1: Use runtime's context
ctx := a.runtime.Context()

// Option 2: Copy services from runtime
ctx := evaluator.NewContext()
for name, svc := range a.runtime.Services() {
    ctx.RegisterService(name, svc)  // Use RegisterService instead of direct map access
}
```

### Tests
- `internal/chat/app_integration_test.go::TestRunScriptSimpleChat` - Currently failing

## 4. Missing `str()` Builtin Function

### Issue
Scripts use `str(value)` to convert values to strings, but this fails with:
```
evaluation error: undefined variable: str
```

### Root Cause
The `str` builtin function may not be registered in the evaluation context.

### Current Status
❌ **Not yet fixed** - Need to verify if `str()` exists and register it properly.

### Proposed Fix
Check `internal/builtin/` for string conversion functions and ensure they're registered.

### Tests
- `internal/chat/app_integration_test.go::TestRunScriptResearch` - Currently failing

## 5. ListValue Field Name Inconsistency

### Issue
Code used `ListValue.Items` but the actual field name is `ListValue.Elements`.

### Root Cause
Incorrect field name in `mcp/service.go` and `chat/app.go`.

### Fix
Changed all references from `.Items` to `.Elements`:
```go
// BEFORE
for i, item := range val.Items {
    result[i] = valueToGo(item)
}

// AFTER
for i, item := range val.Elements {
    result[i] = valueToGo(item)
}
```

### Tests
- Unit tests in `internal/mcp/service_test.go` verify value conversion

## 6. Missing `Runtime.Services()` Method

### Issue
Chat app called `a.runtime.Services()` but the method didn't exist on `slop.Runtime`.

### Root Cause
The Runtime type didn't expose the services map.

### Fix
Added `Services()` method to `pkg/slop/runtime.go:145-148`:
```go
// Services returns all registered services.
func (r *Runtime) Services() map[string]evaluator.Service {
    return r.evaluator.Context().Services
}
```

### Tests
- Chat app builds and links correctly after this fix

## Summary of Issues by Category

### Parser/Lexer Issues (1)
- ✅ **Fixed**: Multiline string limitation documented and scripts rewritten

### Runtime Context Issues (4)
- ✅ **Fixed**: Missing `input` variable
- ❌ **Not Fixed**: Missing `llm` service registration
- ❌ **Not Fixed**: Missing `str()` builtin
- ✅ **Fixed**: Missing `Services()` method

### Data Structure Issues (1)
- ✅ **Fixed**: ListValue.Items → ListValue.Elements

## Test Coverage

### Passing Tests
- ✅ All script parsing tests (4/4 scripts parse correctly)
- ✅ All unit tests for config, mcp, chat app
- ✅ Input context provision tests
- ✅ Invalid syntax error handling

### Failing Tests (Expected Until Services Fixed)
- ❌ Simple chat script execution
- ❌ Research script execution
- ❌ Code review script execution
- ❌ Tool agent script execution

## Next Steps

1. Fix `llm` service registration in `RunScript` context
2. Verify/add `str()` builtin function
3. Add end-to-end tests with mock LLM
4. Document SLOP limitations in README
5. Create troubleshooting guide for users

## Test Methodology

This document was created using Test-Driven Development:

1. **Write failing tests first** - Exposed parsing errors, runtime errors
2. **Fix issues one by one** - Systematic debugging
3. **Verify with tests** - Confirm each fix works
4. **Document findings** - Create this reference

This approach uncovered **6 distinct issues**, **4 of which are now fixed**.
