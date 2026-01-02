---
sidebar_position: 6
---

# SLOP Modules

## Composition Without Collision

---

## 1. The Problem

When you copy code from different sources:

```python
# From library A
def process(item):
    return item.upper()

# From library B - COLLISION
def process(item):
    return item.strip()

# Which one is called?
result = process(x)  # Ambiguous!
```

Worse - hidden dependencies:

```python
# Original source file had:
# ===SOURCE: utils===
# def clean(x): return x.strip()
#
# ===SOURCE: processor===
# def process(x): return utils.clean(x)

# You copy just processor, but you have a DIFFERENT utils:
def clean(x):
    return x * 2  # Totally different!

# processor silently uses wrong utils
```

---

## 2. The Solution

### 2.1 Explicit Dependencies

Every reusable block declares what it needs:

```python
===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {utils: "mycompany/utils@v1"}  # I need THIS utils
---
def process(item):
    return utils.clean(item)
```

### 2.2 Wiring at the Edge

Dependencies are resolved at import time, not inside the block:

```python
===USE: mycompany/processor===
===USE: mycompany/utils===
# Auto-wired: processor.utils → mycompany/utils (IDs match)

===MAIN===
result = processor.process(data)
```

Or with explicit remapping:

```python
===USE: mycompany/processor with {utils: other/utils}===
===USE: other/utils===
# Explicit: processor.utils → other/utils

===MAIN===
result = processor.process(data)
```

---

## 3. Block Format

### 3.1 Full Header

```python
===SOURCE: human_readable_name===
id: "namespace/name@version"
uses: {
    local_name: "required/id@version",
    other: "other/id@version"
}
provides: [func1, func2]  # Optional: default exports all
---
# Code here
def func1():
    pass

def func2():
    other.helper()  # Uses declared dependency
```

### 3.2 Self-Contained Block

No dependencies = copy anywhere:

```python
===SOURCE: string_helpers===
id: "lib/string_helpers@v1"
uses: {}  # Self-contained!
---
def clean(s):
    return s.strip().lower()

def normalize(s):
    return " ".join(s.split())
```

### 3.3 One-Shot Script (No Headers)

For LLM-generated scripts, no boilerplate needed:

```python
# Just write code - no headers required
contacts = salesforce.query("SELECT * FROM Contact")

for contact in contacts with rate(10/s):
    contact.company = clearbit.lookup(contact.email).company

emit(count: len(contacts))
```

---

## 4. Dependency Resolution

### 4.1 Auto-Wire (IDs Match)

```python
===SOURCE: utils===
id: "mycompany/utils@v1"
uses: {}
---
def clean(s):
    return s.strip()

===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {utils: "mycompany/utils@v1"}  # Wants this ID
---
def process(item):
    return utils.clean(item.name)

===USE: mycompany/processor===
===USE: mycompany/utils===
# Auto-wired: IDs match exactly

===MAIN===
processor.process(data)  # Just works
```

### 4.2 Explicit Remap

```python
===SOURCE: my_utils===
id: "my/utils@v2"  # Different ID
uses: {}
---
def clean(s):
    return s.strip().lower()  # Different implementation

===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {utils: "mycompany/utils@v1"}  # Wants original
---
def process(item):
    return utils.clean(item.name)

===USE: mycompany/processor with {utils: my/utils}===  # Remap here
===USE: my/utils===

===MAIN===
processor.process(data)  # Uses my_utils.clean
```

### 4.3 Missing Dependency (Error)

```python
===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {utils: "mycompany/utils@v1"}
---
def process(item):
    return utils.clean(item.name)

===USE: mycompany/processor===
# No utils provided!

===MAIN===
processor.process(data)

# ERROR: processor requires 'utils' ("mycompany/utils@v1")
#        No source with matching ID found
#        No explicit binding provided
#        
# Add one of:
#   ===USE: mycompany/utils===
#   ===USE: mycompany/processor with {utils: some/other}===
```

---

## 5. Namespacing

### 5.1 Qualified Access

All cross-module access is qualified:

```python
===USE: lib/strings===
===USE: lib/numbers===

===MAIN===
# Always prefixed - no ambiguity
clean_name = strings.clean(name)
rounded = numbers.round(value, 2)
```

### 5.2 Same-Name Functions

```python
===SOURCE: clearbit===
id: "enrichment/clearbit@v1"
uses: {}
---
def enrich(record):
    record.company = api.lookup(record.email)
    return record

===SOURCE: linkedin===
id: "enrichment/linkedin@v1"
uses: {}
---
def enrich(record):  # Same name - no collision!
    record.linkedin = api.find(record.name)
    return record

===USE: enrichment/clearbit===
===USE: enrichment/linkedin===

===MAIN===
record = clearbit.enrich(record)   # Clear which one
record = linkedin.enrich(record)   # Clear which one
```

### 5.3 No Global Pollution

```python
# FORBIDDEN in SLOP:
from lib import *       # No wildcard imports
import lib              # No unqualified imports

# REQUIRED:
===USE: lib/module===

===MAIN===
lib.function()          # Always qualified
```

---

## 6. File Organization

### 6.1 Single File

Everything in one file:

```python
===SOURCE: utils===
id: "myapp/utils@v1"
uses: {}
---
def helper():
    pass

===SOURCE: processor===
id: "myapp/processor@v1"
uses: {utils: "myapp/utils@v1"}
---
def process():
    return utils.helper()

===MAIN===
processor.process()
```

### 6.2 Multi-File Library

```
mylib/
├── sources/
│   ├── utils.slop
│   ├── processor.slop
│   └── validator.slop
└── index.slop
```

```python
# sources/utils.slop
===SOURCE: utils===
id: "mylib/utils@v1"
uses: {}
---
def clean(s):
    return s.strip()
```

```python
# sources/processor.slop
===SOURCE: processor===
id: "mylib/processor@v1"
uses: {utils: "mylib/utils@v1"}
---
def process(item):
    return utils.clean(item.name)
```

```python
# index.slop (re-exports)
===USE: ./sources/utils===
===USE: ./sources/processor===

===EXPORT===
utils: utils
processor: processor
```

### 6.3 Using External Libraries

```python
===USE: github.com/company/mylib@v1.2.3===

===MAIN===
mylib.processor.process(data)
```

---

## 7. Bundling

### 7.1 Fat Bundle (Inline Dependencies)

For maximum portability, inline everything:

```python
===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {}  # Now self-contained!
bundles:
  utils:
    from: "mycompany/utils@v1"
    code: |
      def clean(s):
          return s.strip().lower()
---
def process(item):
    return utils.clean(item.name)
```

### 7.2 Bundle Command

```bash
# Create fat bundle
$ slop bundle processor.slop --include-deps -o processor.bundled.slop

# Unbundle to separate files
$ slop unbundle processor.bundled.slop -o ./sources/
```

### 7.3 Dependency Tree

```bash
$ slop deps processor.slop
processor@v1
├── utils@v1 (self-contained)
├── validator@v1
│   └── utils@v1 (shared)
└── formatter@v1 (self-contained)

Total: 4 sources, 2 unique dependencies
```

---

## 8. Versioning

### 8.1 Semantic Versions

```python
===SOURCE: processor===
id: "mycompany/processor@v2.1.0"
uses: {
    utils: "mycompany/utils@v1"      # Any v1.x.x
    # utils: "mycompany/utils@v1.2"  # Any v1.2.x
    # utils: "mycompany/utils@v1.2.3" # Exact version
}
---
```

### 8.2 Version Resolution

```python
===USE: lib/processor@v2===   # Uses latest v2.x.x
===USE: lib/processor@v2.1=== # Uses latest v2.1.x
===USE: lib/processor@v2.1.3=== # Exact version
```

### 8.3 Version Conflicts

```python
===USE: lib/a===  # Requires utils@v1
===USE: lib/b===  # Requires utils@v2

# ERROR: Conflicting versions of utils
#   lib/a requires utils@v1
#   lib/b requires utils@v2
#
# Resolution options:
#   1. ===USE: lib/a with {utils: utils_v1}===
#      ===USE: lib/b with {utils: utils_v2}===
#   2. Update lib/a or lib/b to use compatible version
```

---

## 9. Validation

### 9.1 Dependency Check

```go
func ValidateDependencies(script *Script) []Error {
    errors := []Error{}
    
    for _, source := range script.Sources {
        // Check all uses are satisfied
        for localName, requiredID := range source.Uses {
            resolved := script.Resolve(requiredID)
            if resolved == nil {
                errors = append(errors, Error{
                    Source: source.Name,
                    Message: fmt.Sprintf(
                        "requires '%s' (%s) - not found",
                        localName, requiredID,
                    ),
                })
            }
        }
        
        // Check all external refs are declared
        for _, ref := range source.ExternalRefs() {
            if !source.Uses.Has(ref) {
                errors = append(errors, Error{
                    Source: source.Name,
                    Message: fmt.Sprintf(
                        "references '%s' but not declared in uses",
                        ref,
                    ),
                })
            }
        }
    }
    
    // Check for cycles
    if cycles := script.DependencyCycles(); len(cycles) > 0 {
        errors = append(errors, Error{
            Message: fmt.Sprintf("circular dependencies: %v", cycles),
        })
    }
    
    return errors
}
```

### 9.2 CLI Check

```bash
$ slop check script.slop

Dependencies:
  ✓ processor@v1 → utils@v1 (resolved)
  ✓ processor@v1 → validator@v1 (resolved)
  ✓ validator@v1 → utils@v1 (resolved, shared)
  ✓ No circular dependencies
  ✓ No missing dependencies
  ✓ No version conflicts
```

---

## 10. Summary

| Scenario | What To Do |
|----------|------------|
| LLM generates one-shot | No headers, just write code |
| Reuse self-contained block | `===USE: block===` |
| Reuse block with deps | Deps auto-wire if IDs match |
| Remap a dependency | `===USE: block with {dep: other}===` |
| Avoid all conflicts | Every access is `module.function` |
| Maximum portability | Bundle deps inline |

**The key insight:** Dependencies are declared in the block header, wired at the import site. No editing library code. No implicit resolution. No surprises.
