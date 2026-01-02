---
sidebar_position: 1
---

# SLOP Language Specification

## Version 2.0

---

## 1. Lexical Structure

### 1.1 Encoding

UTF-8. Indentation: 4 spaces (no tabs).

### 1.2 Comments

```python
# Single line comment

# Multi-line comments are just
# multiple single-line comments
```

### 1.3 Keywords

```
if elif else for in with match def return emit stop
and or not true false none range limit rate parallel timeout
```

### 1.4 Identifiers

```
identifier := [a-zA-Z_][a-zA-Z0-9_]*
```

### 1.5 Literals

```python
# Numbers
42              # integer
3.14            # float
1_000_000       # underscores allowed

# Strings
"hello"         # double quotes
'hello'         # single quotes
"line 1\nline 2"  # escape sequences
"{var} is {x}"  # interpolation with {}

# Collections
[1, 2, 3]       # list
{a: 1, b: 2}    # map
{a, b, c}       # set (if no colons)
```

### 1.6 Operators

```
# Arithmetic
+  -  *  /  %  **

# Comparison
==  !=  <  >  <=  >=

# Logical
and  or  not

# Membership
in  not in

# Access
.   []   ?.   ?[]

# Assignment
=  +=  -=  *=  /=

# Pipeline
|  ->
```

---

## 2. Types

### 2.1 Primitive Types

| Type | Examples | Notes |
|------|----------|-------|
| `none` | `none` | Null value |
| `bool` | `true`, `false` | |
| `int` | `42`, `-1`, `1_000` | 64-bit |
| `float` | `3.14`, `-0.5`, `1e10` | 64-bit |
| `string` | `"hello"`, `'world'` | UTF-8 |

### 2.2 Collection Types

| Type | Literal | Notes |
|------|---------|-------|
| `list` | `[1, 2, 3]` | Ordered, mutable |
| `map` | `{a: 1, b: 2}` | Key-value pairs |
| `set` | `{a, b, c}` | Unique values |

### 2.3 Special Types

```python
# Enum (inline or defined)
enum(pending, active, done)

# Schema (for LLM calls)
{
    name: string,
    age: int,
    tags: list(string),
    status: enum(active, inactive)
}

# Range
range(10)       # 0..9
range(1, 10)    # 1..9
range(0, 10, 2) # 0, 2, 4, 6, 8

# Reference
ref(some_id)    # Reference to entity
```

### 2.4 Type Annotations (Optional)

```python
def greet(name: string) -> string:
    return "Hello, {name}"

count: int = 0
items: list(string) = []
```

---

## 3. Expressions

### 3.1 Primary Expressions

```python
# Literals
42
"hello"
[1, 2, 3]
{a: 1, b: 2}

# Identifiers
foo
bar.baz
items[0]
data?.optional?.field

# Parenthesized
(a + b) * c
```

### 3.2 Operators

Precedence (highest to lowest):

1. `()` `[]` `.` `?.`
2. `**`
3. `not` `-` (unary)
4. `*` `/` `%`
5. `+` `-`
6. `<` `>` `<=` `>=`
7. `==` `!=`
8. `in` `not in`
9. `and`
10. `or`

### 3.3 Conditionals (Expression Form)

```python
result = value if condition else other

status = match code:
    200 -> "ok"
    404 -> "not found"
    500 -> "error"
    _ -> "unknown"
```

### 3.4 Collection Operations

```python
# Indexing
items[0]
items[-1]
items[1:3]
items[:5]
items[::2]

# Map access
data["key"]
data.key        # Same as data["key"]
data?.key       # None if data is none

# Comprehensions (bounded)
[x * 2 for x in items]
[x for x in items if x > 0]
{k: v * 2 for k, v in pairs}
```

### 3.5 Pipeline Expressions

```python
# Pipeline operator
items | filter(x -> x > 0) | map(x -> x * 2) | take(10)

# Arrow for lambdas in pipelines
items.filter(x -> x > 0)
items.map(x -> x.name)
items.sort(x -> x.date)
```

---

## 4. Statements

### 4.1 Assignment

```python
x = 1
x, y = get_pair()
x += 1
data.field = value
items[0] = new_value
```

### 4.2 If Statement

```python
if condition:
    do_this()

if condition:
    do_this()
else:
    do_that()

if a:
    path_a()
elif b:
    path_b()
elif c:
    path_c()
else:
    default()
```

### 4.3 For Statement

**All `for` loops iterate over bounded collections.**

```python
# Basic iteration
for item in items:
    process(item)

# With index
for i, item in enumerate(items):
    print("{i}: {item}")

# With bounds and rates
for item in items with limit(100):
    process(item)

for item in items with rate(10/s):
    process(item)

for item in items with parallel(5):
    process(item)

for item in items with limit(100), rate(10/s), parallel(5), timeout(30s):
    process(item)

# Range (always bounded)
for i in range(10):
    print(i)

for i in range(start, end):
    print(i)
```

**Modifiers:**

| Modifier | Meaning |
|----------|---------|
| `limit(N)` | Maximum N iterations |
| `rate(N/s)` | N per second |
| `rate(N/m)` | N per minute |
| `rate(N/h)` | N per hour |
| `parallel(N)` | N concurrent executions |
| `timeout(Ns)` | Timeout in seconds |
| `timeout(Nm)` | Timeout in minutes |

### 4.4 Match Statement

```python
match value:
    1 -> handle_one()
    2 -> handle_two()
    n if n > 10 -> handle_large(n)
    _ -> handle_default()

match result:
    {ok: data} -> process(data)
    {error: msg} -> log_error(msg)
    _ -> panic("unexpected")

match response.status:
    200 -> response.body
    404 -> none
    code -> error("HTTP {code}")
```

### 4.5 Return Statement

```python
def get_value():
    return 42

def maybe_value(flag):
    if flag:
        return "yes"
    return "no"
```

### 4.6 Emit Statement

Produces output from the script:

```python
emit(result)
emit(count: 10, status: "done")
emit({data: results, meta: {time: elapsed}})
```

### 4.7 Stop Statement

Terminates execution:

```python
if error_condition:
    emit(error: "something wrong")
    stop

# With rollback
stop with rollback
```

### 4.8 Expression Statement

```python
print("hello")
service.do_something()
items.append(value)
```

---

## 5. Functions

### 5.1 Definition

```python
def function_name(param1, param2):
    body
    return result

def with_defaults(a, b=10, c="default"):
    return a + b

def with_types(name: string, count: int) -> list(string):
    return [name] * count
```

### 5.2 Restrictions

**No recursion:**
```python
# ERROR: recursive call
def factorial(n):
    if n <= 1:
        return 1
    return n * factorial(n - 1)  # FORBIDDEN
```

**No nested definitions:**
```python
# ERROR: nested function
def outer():
    def inner():  # FORBIDDEN
        pass
    return inner
```

**No closures over mutable state:**
```python
# ERROR: closure
def make_counter():
    count = 0
    def increment():  # FORBIDDEN
        count += 1
        return count
    return increment
```

### 5.3 Lambdas

Single-expression only:

```python
# Arrow syntax for inline functions
double = x -> x * 2
add = (a, b) -> a + b

# Used in pipelines
items.map(x -> x.name)
items.filter(x -> x.active)
items.sort(x -> x.date, reverse: true)
```

---

## 6. Service Calls

### 6.1 MCP Services

```python
# Service calls use dot notation
result = service_name.tool_name(args)

# Examples
contacts = salesforce.query("SELECT * FROM Contact")
company = clearbit.lookup(email: "test@example.com")
results = web.search("query", limit: 10)
file_content = filesystem.read("/path/to/file")
```

### 6.2 LLM Calls

```python
result = llm.call(
    prompt: "Your prompt with {variable} interpolation",
    schema: {
        field1: string,
        field2: int,
        field3: enum(option1, option2, option3),
        field4: list(string),
        field5: {
            nested_field: string,
            nested_num: float
        }
    }
)

# Optional parameters
result = llm.call(
    prompt: "...",
    schema: {...},
    model: "claude-sonnet",
    max_tokens: 1000,
    temperature: 0.7,
    system: "You are a helpful assistant."
)

# Access result
print(result.field1)
for item in result.field4:
    process(item)
```

### 6.3 Agent Calls

Invoke a sub-agent (another SLOP script):

```python
result = agent.call(
    script: "agents/researcher.slop",
    input: {query: "topic to research"},
    limits: {max_steps: 10, timeout: 60s}
)
```

---

## 7. Built-in Functions

### 7.1 Type Functions

```python
int("42")           # Parse int
float("3.14")       # Parse float
str(42)             # To string
bool(value)         # To boolean
list(iterable)      # To list
set(iterable)       # To set
map(key_vals)       # To map
```

### 7.2 Collection Functions

```python
len(collection)     # Length
min(collection)     # Minimum
max(collection)     # Maximum
sum(collection)     # Sum
sorted(collection)  # Sorted copy
reversed(collection) # Reversed copy
enumerate(collection) # (index, item) pairs
zip(coll1, coll2)   # Pair items
range(stop)         # 0 to stop-1
range(start, stop)  # start to stop-1
range(start, stop, step)
```

### 7.3 String Functions

```python
s.upper()           # Uppercase
s.lower()           # Lowercase
s.strip()           # Trim whitespace
s.split(sep)        # Split to list
s.join(list)        # Join list
s.replace(old, new) # Replace
s.startswith(prefix)
s.endswith(suffix)
s.contains(substr)  # Or: substr in s
```

### 7.4 List Methods

```python
items.append(x)     # Add to end
items.extend(other) # Add all
items.insert(i, x)  # Insert at index
items.remove(x)     # Remove first x
items.pop()         # Remove last
items.pop(i)        # Remove at index
items.clear()       # Remove all
items.index(x)      # Find index
items.count(x)      # Count occurrences
items.sort()        # Sort in place
items.reverse()     # Reverse in place
```

### 7.5 Map Methods

```python
m.keys()            # All keys
m.values()          # All values
m.items()           # (key, value) pairs
m.get(key, default) # Get with default
m.pop(key)          # Remove and return
m.update(other)     # Merge in other
m.clear()           # Remove all
key in m            # Check key exists
```

### 7.6 Pipeline Functions

```python
items | filter(predicate)    # Keep matching
items | map(transform)       # Transform each
items | flat_map(transform)  # Transform and flatten
items | take(n)              # First n
items | drop(n)              # Skip first n
items | take_while(pred)     # Take while true
items | drop_while(pred)     # Drop while true
items | unique()             # Deduplicate
items | unique(key_fn)       # Dedupe by key
items | sort()               # Sort
items | sort(key_fn)         # Sort by key
items | group(key_fn)        # Group by key
items | chunk(n)             # Chunk into lists of n
items | flatten()            # Flatten one level
items | first()              # First item or none
items | last()               # Last item or none
```

### 7.7 Utility Functions

```python
print(values...)    # Debug output
log.debug(msg)      # Log at debug level
log.info(msg)       # Log at info level  
log.warn(msg)       # Log at warn level
log.error(msg)      # Log at error level
sleep(seconds)      # Pause execution
now()               # Current timestamp
uuid()              # Generate UUID
hash(value)         # Hash value
json.parse(string)  # Parse JSON
json.stringify(val) # To JSON string
```

---

## 8. Error Handling

### 8.1 Try/Catch

```python
try:
    result = risky_operation()
catch error:
    log.error("Failed: {error}")
    result = default_value

# Specific error types
try:
    data = api.fetch(id)
catch NotFoundError:
    data = none
catch RateLimitError as e:
    sleep(e.retry_after)
    data = api.fetch(id)
catch error:
    emit(error: str(error))
    stop
```

### 8.2 Error Propagation

```python
# Errors propagate up by default
def load_data():
    return api.fetch()  # Error bubbles up if fetch fails

# Explicit error
def validate(data):
    if not data.email:
        error("Email required")  # Raises error
    return data
```

### 8.3 Optional Chaining

```python
# Returns none if any part is none
value = data?.user?.profile?.name

# With default
value = data?.user?.name or "Anonymous"

# Safe index
first = items?[0]
```

---

## 9. Input/Output

### 9.1 Script Input

```python
# Input is available as 'input' variable
task = input.task
options = input.options or {}
limit = input.limit or 100
```

### 9.2 Script Output

```python
# Emit produces output
emit(result)
emit(status: "done", count: len(results))

# Multiple emits accumulate
for batch in batches:
    result = process(batch)
    emit(batch_result: result)  # Streamed output
```

### 9.3 Intermediate Results

```python
# Store for later steps or debugging
store("key", value)
stored_value = load("key")

# Named checkpoints
checkpoint("after_fetch", data)
```

---

## 10. Grammar Summary

```ebnf
program     = statement* ;

statement   = simple_stmt NEWLINE
            | compound_stmt ;

simple_stmt = assignment
            | expr_stmt
            | return_stmt
            | emit_stmt
            | stop_stmt ;

compound_stmt = if_stmt
              | for_stmt
              | match_stmt
              | try_stmt
              | func_def ;

assignment  = target "=" expr
            | target "+=" expr
            | target "-=" expr ;

if_stmt     = "if" expr ":" block
              ("elif" expr ":" block)*
              ("else" ":" block)? ;

for_stmt    = "for" pattern "in" expr modifiers? ":" block ;

modifiers   = "with" modifier ("," modifier)* ;
modifier    = "limit" "(" expr ")"
            | "rate" "(" rate_expr ")"
            | "parallel" "(" expr ")"
            | "timeout" "(" time_expr ")" ;

match_stmt  = "match" expr ":" NEWLINE match_arm+ ;
match_arm   = INDENT pattern guard? "->" expr NEWLINE ;
guard       = "if" expr ;

try_stmt    = "try" ":" block catch_clause+ ;
catch_clause = "catch" (NAME ("as" NAME)?)? ":" block ;

func_def    = "def" NAME "(" params? ")" return_type? ":" block ;

expr        = pipeline_expr ;
pipeline_expr = or_expr ("|" pipe_call)* ;
or_expr     = and_expr ("or" and_expr)* ;
and_expr    = not_expr ("and" not_expr)* ;
not_expr    = "not" not_expr | comparison ;
comparison  = arith_expr (comp_op arith_expr)* ;
arith_expr  = term (("+"|"-") term)* ;
term        = factor (("*"|"/"|"%") factor)* ;
factor      = ("+" | "-") factor | power ;
power       = primary "**" factor | primary ;
primary     = atom trailer* ;
trailer     = "." NAME | "[" expr "]" | "(" args? ")" | "?." NAME | "?[" expr "]" ;
atom        = NAME | NUMBER | STRING | list | map | "(" expr ")" | lambda ;

lambda      = params "->" expr ;
```

---

## Appendix A: Differences from Python

| Feature | Python | SLOP |
|---------|--------|------|
| `while` loops | Yes | **No** |
| Recursion | Yes | **No** |
| Nested functions | Yes | **No** |
| `*args`, `**kwargs` | Yes | **No** |
| `eval`, `exec` | Yes | **No** |
| `import` | Yes | **Module system** |
| Generators | Yes | **No** (use pipelines) |
| Classes | Yes | **No** (use maps) |
| Decorators | Yes | **No** |
| Global variables | Yes | **No** |
| `lambda` | Multi-expr | **Single-expr only** |

## Appendix B: Reserved for Future

```
async await yield class import from as
try catch finally raise with assert
global nonlocal del pass break continue
```
