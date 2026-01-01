# SLOP v2

**Script Language for Orchestrating Protocols**

*A bounded scripting language where LLMs are tools, not controllers.*

---

## What is SLOP?

SLOP is a Python-like scripting language designed for:

1. **LLM-generated scripts** — One-shot task execution
2. **Agent workflows** — LLM as a service call, not the control plane
3. **Service composition** — MCP orchestration with rate limits and bounds
4. **Safe execution** — Provably terminates, auditable, rollback-able

```python
# Simple and readable - LLM generates this, humans can audit it
task = input.task

# LLM is just another service call with structured output
plan = llm.call(
    prompt: "Break into subtasks: {task}",
    schema: {subtasks: list(string)}
)

# Bounded execution - guaranteed to terminate
for subtask in plan.subtasks with limit(10), rate(5/s):
    result = tools.execute(subtask)
    
    if result.needs_help:
        guidance = llm.call(
            prompt: "How to handle: {result.error}",
            schema: {action: enum(retry, skip, abort)}
        )
        # Code decides what to do, not LLM
        match guidance.action:
            retry -> tools.execute(subtask)
            skip  -> continue
            abort -> break

emit(results)
```

---

## Why SLOP?

### vs. LangGraph / Agent Frameworks

| Aspect | LangGraph | SLOP |
|--------|-----------|------|
| Control flow | LLM decides (implicit) | Code decides (explicit) |
| LLM calls | Unbounded loops | Explicit limits |
| State | Hidden in graph | Visible in variables |
| Debugging | "Why did it take that path?" | Read the code |
| Cost control | Hope it stops | `limit(10)` - guaranteed |
| Verification | Can't | Provable bounds |

### vs. Raw Python

| Aspect | Python | SLOP |
|--------|--------|------|
| Termination | Undecidable | Guaranteed (no while/recursion) |
| Side effects | Anything | Only declared services |
| Validation | Runtime errors | Pre-execution verification |
| Rollback | DIY | Built-in transaction log |
| LLM generation | Error-prone | Designed for it |

---

## Design Principles

### 1. Flat, Not Nested

```python
# YES - flat, readable
for item in items with limit(100):
    result = process(item)
    output.append(result)

# NO - deep nesting (Lisp-style)
(map (fn [item] (process item)) (take 100 items))
```

### 2. Bounded, Not Unbounded

```python
# YES - explicit bound
for i in range(100):
    do_work()

# NO - unbounded
while not done:  # SYNTAX ERROR - while not allowed
    do_work()
```

### 3. LLM as Service, Not Controller

```python
# YES - code controls, LLM advises
action = llm.call(prompt: "what next?", schema: {action: enum(a,b,c)})
match action:
    a -> do_a()
    b -> do_b()
    c -> do_c()

# NO - LLM controls flow (LangGraph-style)
# graph.add_conditional_edges(llm_decides_next_node)
```

### 4. Explicit Dependencies

```python
# YES - declared at the edge
===USE: lib/processor with {utils: my/utils}===

# NO - implicit resolution
from lib import *  # What did I just import?
```

---

## Two Modes

### Mode 1: One-Shot Scripts (90% of usage)

No boilerplate. LLM generates, runtime executes:

```python
contacts = salesforce.query("SELECT * FROM Contact LIMIT 100")

for contact in contacts with rate(10/s):
    contact.company = clearbit.lookup(contact.email).company
    salesforce.update(contact.id, contact)

emit(count: len(contacts))
```

### Mode 2: Composition (when reusing code)

Import existing blocks, wire at the edges:

```python
===USE: recipes/enrichment===
===USE: recipes/dedup===
===USE: mycompany/crm with {config: my/config}===

===MAIN===
contacts = crm.get_contacts()
contacts = dedup.by_email(contacts)
enrichment.process_all(contacts)
```

---

## Quick Reference

### Bounds and Rates

```python
for x in items with limit(100):           # Max 100 iterations
for x in items with rate(10/s):           # 10 per second
for x in items with rate(100/m):          # 100 per minute
for x in items with parallel(5):          # 5 concurrent
for x in items with timeout(30s):         # 30 second max
for x in items with limit(100), rate(10/s), parallel(3):  # Combined
```

### LLM Calls

```python
result = llm.call(
    prompt: "Your prompt here with {variables}",
    schema: {                              # Required - structured output
        field1: string,
        field2: int,
        field3: enum(a, b, c),
        field4: list(string),
        field5: {nested: string}
    },
    model: "claude-sonnet",               # Optional
    max_tokens: 1000,                     # Optional
    temperature: 0.7                      # Optional
)
```

### MCP Service Calls

```python
result = service.tool(arg1, arg2, named: value)

# Examples
contacts = salesforce.query("SELECT * FROM Contact")
company = clearbit.lookup(email)
results = web.search(query, limit: 10)
```

### Control Flow

```python
if condition:
    do_this()
elif other:
    do_that()
else:
    do_default()

for item in collection with limit(N):
    process(item)

match value:
    pattern1 -> result1
    pattern2 -> result2
    _ -> default
```

### Functions (No Recursion)

```python
def process(item):
    cleaned = item.strip()
    validated = validate(cleaned)
    return validated

# Functions cannot call themselves or form cycles
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [SPEC.md](SPEC.md) | Language specification |
| [SAFETY.md](SAFETY.md) | Verification, bounds, rollback |
| [AGENTS.md](AGENTS.md) | Agent patterns and examples |
| [MODULES.md](MODULES.md) | Composition and dependencies |
| [EXAMPLES.md](EXAMPLES.md) | Comprehensive examples |
| [BUILTINS.md](BUILTINS.md) | Built-in functions and services |

---

## Installation

```bash
# CLI
go install github.com/anthropics/slop/cmd/slop@latest

# Run a script
slop run script.slop

# Validate without running
slop check script.slop

# Show execution plan
slop plan script.slop
```

---

## Example: Research Agent

```python
query = input.query

# Step 1: Expand query (one LLM call)
expansion = llm.call(
    prompt: "Generate search queries for: {query}",
    schema: {queries: list(string), max: 5}
)

# Step 2: Search (bounded, parallel)
documents = []
for q in expansion.queries with parallel(3), rate(5/s):
    results = web.search(q, limit: 10)
    documents.extend(results)

documents = dedup(documents, by: url)[:50]  # Cap at 50

# Step 3: Analyze with possible refinement (bounded)
for iteration in range(3):
    analysis = llm.call(
        prompt: "Analyze for '{query}':\n{documents}",
        schema: {
            answer: string,
            confidence: float,
            gaps: list(string)
        }
    )
    
    if analysis.confidence > 0.8 or len(analysis.gaps) == 0:
        break
    
    # Fill gaps (bounded)
    for gap in analysis.gaps with limit(3):
        more = web.search(gap, limit: 5)
        documents.extend(more)

emit(
    answer: analysis.answer,
    confidence: analysis.confidence,
    sources: documents.map(d -> d.url)
)
```

**32 lines. Readable. Bounded. Auditable. Provably terminates.**

---

## Project Status

- [x] Language design
- [x] Specification
- [ ] Go runtime
- [ ] MCP integration
- [ ] CLI tools
- [ ] VS Code extension

---

## License

Apache 2.0
