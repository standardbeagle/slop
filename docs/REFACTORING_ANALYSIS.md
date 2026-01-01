# Code Quality Refactoring Analysis

**Generated:** 2025-12-31
**Codebase Quality:** Grade A (Maintainability: 93.17)
**Average Complexity:** 3.42 (Excellent)
**Critical Issues:** 3 high-complexity functions requiring refactoring

---

## Executive Summary

The SLOP codebase maintains excellent overall quality with low average cyclomatic complexity (3.42). However, three critical functions exhibit dangerously high complexity that poses maintenance risks:

1. **Walk** (visitor.go:50) - CC=172 🔴 CRITICAL
2. **NextToken** (lexer.go:73) - CC=45 🟡 HIGH
3. **ValidateAgainstSchema** (llm.go:353) - Moderate complexity, needs decomposition

**Recommended Priority:**
- **Immediate:** Refactor Walk function (CC reduction: 172 → ~30)
- **High:** Refactor NextToken (CC reduction: 45 → ~15)
- **Medium:** Decompose ValidateAgainstSchema for better maintainability
- **Long-term:** Increase functional purity from 29% to 50%+

---

## 1. Walk Function (visitor.go:50) - CC=172 🔴

### Current State
- **Cyclomatic Complexity:** 172 (threshold: 15)
- **Lines of Code:** 455
- **Nature:** Giant switch statement with 40+ AST node type cases
- **Pattern:** Canonical visitor pattern implementation

### Analysis
The Walk function is mechanically simple but structurally complex due to:
- 40+ case statements (one per AST node type)
- Each case follows identical pattern:
  1. Call visitor method
  2. Recursively walk child nodes
  3. Handle errors
- Deep nesting in container nodes (lists, maps, comprehensions)
- Recursive calls creating call-stack depth

**Why This Matters:**
- CC=172 makes the function untestable via traditional paths
- Any modification requires scanning 455 lines
- Adding new AST nodes requires editing this monolithic function
- Error handling is repetitive across all cases

### Refactoring Strategy: Node-Based Traversal

**Recommended Approach:** Move traversal logic into AST nodes themselves

```go
// 1. Add Walkable interface to AST nodes
type Walkable interface {
    Node
    Walk(v Visitor) error
}

// 2. Implement Walk on each node type (example)
func (n *Program) Walk(v Visitor) error {
    if err := v.VisitProgram(n); err != nil {
        return err
    }
    for _, stmt := range n.Statements {
        if w, ok := stmt.(Walkable); ok {
            if err := w.Walk(v); err != nil {
                return err
            }
        }
    }
    return nil
}

// 3. New Walk function becomes trivial
func Walk(v Visitor, node Node) error {
    if w, ok := node.(Walkable); ok {
        return w.Walk(v)
    }
    return nil
}
```

**Benefits:**
- ✅ CC reduction: 172 → ~5 (96% reduction)
- ✅ Each node owns its traversal logic (SRP)
- ✅ Adding new nodes doesn't modify Walk function
- ✅ Node-specific optimizations possible
- ✅ Easier to test (each node's Walk method is small)

**Risks:**
- ⚠️ Requires modifying ~40 node type definitions
- ⚠️ Breaks existing code that depends on visitor.Walk signature
- ⚠️ Migration effort: 2-3 days for implementation + testing

**Alternative Approach: Reflection-Based Visitor**

```go
func Walk(v Visitor, node Node) error {
    // Use reflection to call visitor method
    visitorValue := reflect.ValueOf(v)
    methodName := fmt.Sprintf("Visit%s", reflect.TypeOf(node).Elem().Name())
    method := visitorValue.MethodByName(methodName)

    if !method.IsValid() {
        return nil
    }

    // Call visitor method
    results := method.Call([]reflect.Value{reflect.ValueOf(node)})
    if err := results[0].Interface(); err != nil {
        return err.(error)
    }

    // Recursively walk children using reflection
    return walkChildren(v, node)
}
```

**Benefits:**
- ✅ CC reduction: 172 → ~20
- ✅ No modification to node types
- ✅ Automatically handles new node types

**Risks:**
- ⚠️ Performance penalty (reflection overhead)
- ⚠️ Loses compile-time type safety
- ⚠️ Harder to debug
- ❌ **NOT RECOMMENDED** - violates Go best practices

**Recommended Implementation Plan:**

**Phase 1: Preparation (1 day)**
1. Add comprehensive integration tests for visitor pattern
2. Create Walkable interface
3. Implement on 2-3 representative nodes (Program, IfStatement, BinaryOp)
4. Validate against test suite

**Phase 2: Migration (1-2 days)**
1. Implement Walkable on all statement nodes
2. Implement Walkable on all expression nodes
3. Update Walk function to delegate to node.Walk()
4. Run full test suite

**Phase 3: Cleanup (0.5 days)**
1. Remove old switch-based Walk implementation
2. Update documentation
3. Add examples showing how to add new node types

**Impact Assessment:**
- **Files Modified:** ~12 (all AST node definitions + visitor.go)
- **Test Changes:** Add ~20 unit tests for individual node Walk methods
- **Breaking Changes:** None if we maintain Walk(v Visitor, node Node) signature
- **Performance Impact:** Negligible (virtual method dispatch vs switch)

---

## 2. NextToken Function (lexer.go:73) - CC=45 🟡

### Current State
- **Cyclomatic Complexity:** 45 (threshold: 15)
- **Lines of Code:** 250
- **Nature:** Giant switch statement for tokenization
- **Pattern:** Hand-written lexer with state management

### Analysis
The NextToken function is complex due to:
- 30+ case statements for different token types
- Mixed concerns:
  - Indentation handling (lines 82-89)
  - Pending token queue (lines 75-79)
  - Character-by-character scanning (lines 97-319)
  - Two-character lookahead for operators
- State management via pendingToks, atLineStart, indentStack

**Why This Matters:**
- Difficult to add new token types
- Indentation logic intertwined with tokenization
- Error-prone two-character operator handling
- Hard to test individual token types in isolation

### Refactoring Strategy: State Machine + Token Handlers

**Recommended Approach:** Separate concerns into layers

```go
// 1. Token handler registry
type TokenHandler func(l *Lexer) (Token, error)

var tokenHandlers = map[rune]TokenHandler{
    '+': handlePlus,
    '-': handleMinus,
    '*': handleStar,
    // ... etc
}

// 2. Individual handlers (example)
func handlePlus(l *Lexer) (Token, error) {
    tok := Token{Line: l.line, Column: l.column}
    if l.peekChar() == '=' {
        l.readChar()
        tok.Type = PLUSEQ
        tok.Literal = "+="
    } else {
        tok.Type = PLUS
        tok.Literal = "+"
    }
    l.readChar()
    return tok, nil
}

// 3. Refactored NextToken
func (l *Lexer) NextToken() Token {
    // Layer 1: Pending tokens
    if tok, ok := l.popPending(); ok {
        return tok
    }

    // Layer 2: Indentation
    if l.atLineStart {
        if tok, ok := l.handleIndentation(); ok {
            return tok
        }
    }

    // Layer 3: Skip whitespace
    l.skipWhitespace()

    // Layer 4: Dispatch to handler
    if handler, ok := tokenHandlers[l.ch]; ok {
        tok, _ := handler(l)
        return tok
    }

    // Layer 5: Complex tokens (identifiers, numbers, strings)
    return l.handleComplexToken()
}
```

**Benefits:**
- ✅ CC reduction: 45 → ~12 (73% reduction)
- ✅ Each token type has isolated handler
- ✅ Easy to add new operators
- ✅ Testable in isolation
- ✅ Clear separation of concerns

**Alternative: Generate Lexer from Grammar**

Use `goyacc` or `participle` to generate lexer from grammar specification.

**Benefits:**
- ✅ Declarative grammar definition
- ✅ Zero complexity in hand-written code
- ✅ Proven correctness

**Risks:**
- ⚠️ Harder to customize error messages
- ⚠️ Indentation-sensitive grammar is complex
- ⚠️ May not support Python-style indentation well
- ❌ **NOT RECOMMENDED** - SLOP's indentation rules are too custom

**Recommended Implementation Plan:**

**Phase 1: Extract Token Handlers (1 day)**
1. Create TokenHandler type and registry
2. Extract 5-6 simple operators ('+', '-', '*', etc.)
3. Test that behavior is identical
4. Refactor NextToken to use handlers

**Phase 2: Separate Indentation Logic (0.5 days)**
1. Move indentation handling to separate method
2. Make pendingToks management explicit
3. Add unit tests for indentation edge cases

**Phase 3: Complete Migration (1 day)**
1. Extract remaining operators
2. Extract complex tokens (identifiers, numbers, strings)
3. Run full lexer test suite

**Impact Assessment:**
- **Files Modified:** 2 (lexer.go, potentially new lexer_handlers.go)
- **Test Changes:** Add ~30 unit tests for individual handlers
- **Breaking Changes:** None (NextToken signature unchanged)
- **Performance Impact:** Minimal (map lookup vs switch - Go optimizes both)

---

## 3. ValidateAgainstSchema (llm.go:353) - Moderate Complexity

### Current State
- **Cyclomatic Complexity:** ~15-20 (estimated)
- **Lines of Code:** 115
- **Nature:** Recursive validation function
- **Pattern:** Switch on schema type with nested validation

### Analysis
ValidateAgainstSchema has moderate complexity but poor decomposition:
- 6 schema types with complex validation logic each
- String validation includes enum check + format validation
- Number validation includes min/max constraints
- Object validation includes required field checking + recursive property validation
- Array validation includes recursive item validation
- Deeply nested error wrapping

**Why This Matters:**
- Mixing type checking with constraint validation
- Hard to extend with new validation rules
- Difficult to test edge cases in isolation
- validateFormat is buried as a helper function

### Refactoring Strategy: Validator Chain Pattern

**Recommended Approach:** Decompose into validator pipeline

```go
// 1. Validator interface
type SchemaValidator interface {
    Validate(value any, schema *Schema) error
}

// 2. Specific validators
type StringValidator struct{}
func (v *StringValidator) Validate(value any, schema *Schema) error {
    s, ok := value.(string)
    if !ok {
        return fmt.Errorf("expected string, got %T", value)
    }

    // Enum validation
    if err := validateEnum(s, schema.Enum); err != nil {
        return err
    }

    // Format validation
    if schema.Format != "" {
        return validateFormat(s, schema.Format)
    }

    return nil
}

type IntegerValidator struct{}
func (v *IntegerValidator) Validate(value any, schema *Schema) error {
    n := coerceToInt64(value)
    if n == nil {
        return fmt.Errorf("expected integer, got %T", value)
    }
    return validateNumericConstraints(*n, schema)
}

// ... similar for Number, Boolean, Array, Object

// 3. Validator registry
var validators = map[string]SchemaValidator{
    "string":  &StringValidator{},
    "integer": &IntegerValidator{},
    "number":  &NumberValidator{},
    "boolean": &BooleanValidator{},
    "array":   &ArrayValidator{},
    "object":  &ObjectValidator{},
}

// 4. Refactored main function
func ValidateAgainstSchema(value any, schema *Schema) error {
    if schema == nil {
        return nil
    }

    validator, ok := validators[schema.Type]
    if !ok {
        return nil // Unknown type, skip validation
    }

    return validator.Validate(value, schema)
}
```

**Benefits:**
- ✅ CC reduction: ~18 → ~5
- ✅ Each validator is independently testable
- ✅ Easy to add new schema types
- ✅ Clear separation between type checking and constraint validation
- ✅ Validators can be composed/reused

**Simpler Alternative: Extract Helper Functions**

```go
func ValidateAgainstSchema(value any, schema *Schema) error {
    if schema == nil {
        return nil
    }

    switch schema.Type {
    case "string":
        return validateString(value, schema)
    case "integer":
        return validateInteger(value, schema)
    case "number":
        return validateNumber(value, schema)
    case "boolean":
        return validateBoolean(value, schema)
    case "array":
        return validateArray(value, schema)
    case "object":
        return validateObject(value, schema)
    default:
        return nil
    }
}

func validateString(value any, schema *Schema) error {
    s, ok := value.(string)
    if !ok {
        return fmt.Errorf("expected string, got %T", value)
    }

    if err := validateEnum(s, schema.Enum); err != nil {
        return err
    }

    if schema.Format != "" {
        return validateFormat(s, schema.Format)
    }

    return nil
}

// ... similar for other types
```

**Benefits:**
- ✅ CC reduction: ~18 → ~8
- ✅ Simpler to implement (no interfaces)
- ✅ Each type validator is testable
- ✅ Minimal refactoring effort

**Recommended Implementation Plan:**

**Phase 1: Extract Type Validators (0.5 days)**
1. Extract validateString, validateInteger, validateNumber
2. Update ValidateAgainstSchema to call helpers
3. Run existing tests

**Phase 2: Extract Constraint Validators (0.5 days)**
1. Extract validateEnum, validateNumericConstraints
2. Add unit tests for each constraint validator
3. Validate edge cases (nil values, missing fields)

**Impact Assessment:**
- **Files Modified:** 1 (llm.go)
- **Test Changes:** Add ~15 unit tests for individual validators
- **Breaking Changes:** None (public API unchanged)
- **Performance Impact:** None (inlined by compiler)

---

## 4. Medium-Term Improvements

### Increase Functional Purity (Current: 29%)

**High-Impact Targets:**

1. **builtinHashHmac** - Global write violations
   - Issue: Uses global crypto.Hash registry
   - Fix: Create hash provider interface, inject as dependency
   - Impact: Reduces global coupling, improves testability

2. **Service implementations** - External I/O
   - Issue: Services perform I/O directly
   - Fix: Separate pure business logic from I/O effects
   - Pattern: Use Result<T, E> types for operations that can fail

**Strategy:**
```go
// Before (impure)
func builtinHashHmac(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
    // Uses global crypto registry
    hash := crypto.SHA256.New()
    // ... mutates state
}

// After (pure)
type HashProvider interface {
    New(algo string) hash.Hash
}

func builtinHashHmac(provider HashProvider) BuiltinFunc {
    return func(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
        hash := provider.New("sha256")
        // ... pure computation
    }
}
```

**Benefits:**
- Easier to test (no globals)
- Easier to mock external dependencies
- Enables parallel execution
- Reduces hidden coupling

### Add Semantic Annotations

**Current State:**
- Low usage of code navigation annotations
- Difficult to trace data flow through evaluator

**Recommended Annotations:**

```go
// @lci:critical-path High-frequency evaluation path
func (e *Evaluator) Eval(node ast.Node) (Value, error) {
    // ...
}

// @lci:side-effect Modifies global state
func builtinHashHmac(...) {
    // ...
}

// @lci:pure No side effects
func normalizeType(t string) string {
    // ...
}
```

**Benefits:**
- Better code navigation in IDEs
- Easier onboarding for new developers
- Automated detection of side effects

---

## 5. Testing Strategy

### Current State
- ✅ Excellent edge case coverage (schema validation, input validation)
- ✅ Good test organization (lexer_test.go has 19 test functions)
- ⚠️ Limited property-based testing

### Recommended Additions

**1. Property-Based Testing for Evaluator**

```go
func TestEvaluator_Properties(t *testing.T) {
    // Property: Evaluating same expression always returns same result
    quick.Check(func(expr string) bool {
        result1, _ := eval(expr)
        result2, _ := eval(expr)
        return result1.Equals(result2)
    }, nil)
}
```

**2. Fuzzing for Lexer/Parser**

```go
func FuzzLexer(f *testing.F) {
    f.Add("if x:\n    y = 1")
    f.Fuzz(func(t *testing.T, input string) {
        l := lexer.New(input)
        for {
            tok := l.NextToken()
            if tok.Type == lexer.EOF {
                break
            }
            // Shouldn't panic
        }
    })
}
```

**3. Mutation Testing**

Use `go-mutesting` to validate test quality:
```bash
go-mutesting ./internal/evaluator/...
```

---

## Implementation Roadmap

### Priority 1: Critical Issues (Week 1-2)
- [ ] Refactor Walk function (3 days)
  - [ ] Day 1: Add Walkable interface, implement on 3 nodes, validate
  - [ ] Day 2: Implement on all nodes, update Walk function
  - [ ] Day 3: Testing, cleanup, documentation

- [ ] Refactor NextToken (2 days)
  - [ ] Day 1: Extract token handlers, test simple operators
  - [ ] Day 2: Complete migration, test full suite

### Priority 2: Maintainability (Week 3)
- [ ] Decompose ValidateAgainstSchema (1 day)
  - [ ] Extract type validators
  - [ ] Add constraint validators
  - [ ] Unit tests for each validator

### Priority 3: Quality Improvements (Week 4+)
- [ ] Add property-based testing (2 days)
- [ ] Increase functional purity (3 days)
- [ ] Add semantic annotations (1 day)
- [ ] Set up mutation testing (1 day)

---

## Metrics & Success Criteria

### Before Refactoring
- Walk CC: 172
- NextToken CC: 45
- ValidateAgainstSchema CC: ~18
- Average CC: 3.42
- Functional Purity: 29%
- Test Coverage: Good (qualitative)

### After Refactoring (Target)
- Walk CC: <10 (94% reduction)
- NextToken CC: <15 (67% reduction)
- ValidateAgainstSchema CC: <8 (56% reduction)
- Average CC: 3.2 (maintain)
- Functional Purity: 50%+
- Test Coverage: Excellent (property tests + unit tests)

### Quality Gates
- ✅ All existing tests pass
- ✅ No performance regression (>5%)
- ✅ Documentation updated
- ✅ Code review approval
- ✅ Mutation testing score >80%

---

## Risk Mitigation

### Risk: Breaking Existing Code
- **Mitigation:** Maintain public API signatures
- **Validation:** Comprehensive integration tests before refactoring
- **Rollback:** Feature flags for new implementations

### Risk: Performance Regression
- **Mitigation:** Benchmark critical paths before/after
- **Validation:** CI performance tests
- **Threshold:** >5% regression requires optimization

### Risk: Increased Maintenance Burden
- **Mitigation:** Thorough documentation of new patterns
- **Validation:** Code review focuses on clarity
- **Monitoring:** Track time-to-fix for bugs in refactored code

---

## Conclusion

The SLOP codebase is in excellent shape overall (Grade A, 93.17 maintainability). The three identified high-complexity functions are **localized issues** that can be systematically addressed without major architectural changes.

**Key Takeaways:**
1. **Walk function is the highest priority** - CC=172 is dangerously high
2. **Node-based traversal pattern** is the recommended solution (not reflection)
3. **NextToken benefits from handler pattern** - separates concerns cleanly
4. **ValidateAgainstSchema is lowest priority** - moderate complexity, simple fix

**Estimated Total Effort:** 10-12 days for all refactoring + testing

**Expected Outcome:**
- 85% reduction in critical complexity hotspots
- Improved testability and maintainability
- No breaking changes to public API
- Foundation for future enhancements

The refactoring is **low-risk, high-reward** and should be prioritized in the next development cycle.
