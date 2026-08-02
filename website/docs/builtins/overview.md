---
sidebar_position: 1
title: Built-in Functions
description: Reference for SLOP's built-in string, math, list, map, and utility functions, with usage examples.
---

# SLOP Built-ins

## Functions, Services, and Generators

---

## 1. Core Functions

### 1.1 Type Conversion

```python
int(x)              # Convert to integer
float(x)            # Convert to float
str(x)              # Convert to string
bool(x)             # Convert to boolean
list(x)             # Convert iterable to list
set(x)              # Convert iterable to set
dict(x)             # Convert a list of [key, value] pairs to a map —
                    # the resulting map's key order is currently unreliable
                    # for keys()/emit(); json_stringify(x) shows it correctly
```

### 1.2 Type Checking

```python
type(x)             # Returns type name as string
is_none(x)          # x is none
is_bool(x)          # x is boolean
is_int(x)           # x is integer
is_float(x)         # x is float
is_number(x)        # x is int or float
is_string(x)        # x is string
is_list(x)          # x is list
is_map(x)           # x is map
is_set(x)           # x is set
is_callable(x)      # x is function
```

### 1.3 Math

```python
abs(x)              # Absolute value
min(a, b, ...)      # Minimum value
max(a, b, ...)      # Maximum value
sum(list)           # Sum of list
round(x, n?)        # Round to n decimals
floor(x)            # Round down
ceil(x)             # Round up
pow(base, exp)      # Exponentiation
sqrt(x)             # Square root
log(x, base?)       # Logarithm
sin(x), cos(x), tan(x)  # Trigonometry
```

### 1.4 String Functions

```python
len(s)              # Length
s.upper()           # Uppercase
s.lower()           # Lowercase
s.strip()           # Trim whitespace
s.lstrip()          # Trim left
s.rstrip()          # Trim right
s.split(sep?)       # Split to list (default: whitespace)
s.join(list)        # Join list with s
s.replace(old, new) # Replace all occurrences
s.startswith(prefix)
s.endswith(suffix)
s.contains(substr)  # Or: substr in s
s.find(substr)      # Index or -1
s.count(substr)     # Count occurrences
s.repeat(n)         # Repeat n times
s.reverse()         # Reverse string
s.lines()           # Split by newlines
s.words()           # Split by whitespace

# Free functions, not string methods:
format(fmt, args...) # Format string
pad_left(s, n, char?)
pad_right(s, n, char?)
slice(s, start, end?)
```

### 1.5 List Functions

```python
len(list)           # Length
list.append(x)      # Add to end (mutates)
list.extend(other)  # Add all (mutates)
list.insert(i, x)   # Insert at index (mutates)
list.remove(x)      # Remove first x (mutates)
list.pop(i?)        # Remove and return (mutates)
list.clear()        # Remove all (mutates)
list.index(x)       # Find index or error
list.count(x)       # Count occurrences
list.sort()         # Sort in place, ascending only (mutates)
list.reverse()      # Reverse in place (mutates)
list.copy()         # Shallow copy

# Non-mutating, and the only forms that take a key function or reverse:
sorted(list, key_fn?, reverse: bool?)  # Return sorted copy
reversed(list)      # Return reversed copy
```

### 1.6 Map Functions

```python
len(map)            # Number of keys
map.keys()          # List of keys
map.values()        # List of values
map.items()         # List of [key, value] pairs
map.get(key, default?) # Get with default
map.pop(key)        # Remove and return (mutates)
map.update(other)   # Merge other into map (mutates)
map.clear()         # Remove all (mutates)
map.copy()          # Shallow copy
key in map          # Check key exists
map.merge(other)    # Return merged copy
```

### 1.7 Set Functions

```python
len(set)            # Size
set.add(x)          # Add element (mutates)
set.remove(x)       # Remove element (mutates)
set.discard(x)      # Remove if exists (mutates)
set.pop()           # Remove arbitrary (mutates)
set.clear()         # Remove all (mutates)
x in set            # Check membership
set.union(other)    # a | b
set.intersection(other)  # a & b
set.difference(other)    # a - b
set.issubset(other)
set.issuperset(other)
symmetric_difference(a, b)  # a ^ b — free function, not a set method
```

---

## 2. Collection Pipeline

The functions below are free functions, chained with the `|` pipe operator.
They are not list methods — `items.filter(...)` does not work; list
methods are the fixed set in section 1.5 above.

```python
items | filter(x -> x > 0) | map(x -> x * 2)
```

### 2.1 Transformation

```python
map(fn)             # Transform each element
flat_map(fn)        # Transform and flatten
flatten()           # Flatten one level
flatten(depth)      # Flatten n levels
```

### 2.2 Filtering

```python
filter(predicate)   # Keep matching
reject(predicate)   # Remove matching
compact()           # Remove none/false values
unique()            # Remove duplicates
unique(key_fn)      # Remove duplicates by key
```

### 2.3 Selection

```python
take(n)             # First n elements
drop(n)             # Skip first n
take_while(pred)    # Take while true
drop_while(pred)    # Skip while true
first()             # First element or none
last()              # Last element or none
nth(n)              # Element at index n
slice(start, end?)  # Slice of list
```

### 2.4 Ordering

```python
sorted(items)                        # Sort ascending
sorted(items, key_fn)                # Sort by key
sorted(items, key_fn, reverse: true) # Sort descending
reversed(items)                      # Reverse order
random_shuffle(items)                # Random order (seed with random_seed(n))
```

### 2.5 Grouping

```python
group(key_fn)       # Group into map
partition(pred)     # Split into [matches, non-matches]
chunk(n)            # Split into chunks of size n
window(n)           # Sliding window of size n
```

### 2.6 Aggregation

```python
reduce(fn, init)    # Fold into single value
count()             # Number of elements
count(pred)         # Count matching
sum()               # Sum numbers
sum(key_fn)         # Sum by key
avg()               # Average
avg(key_fn)         # Average by key
min()               # Minimum
min(key_fn)         # Minimum by key
max()               # Maximum
max(key_fn)         # Maximum by key
any(pred)           # Any match?
all(pred)           # All match?
none(pred)          # None match?
find(pred)          # First match or none
find_index(pred)    # Index of first match or -1
```

### 2.7 Combination

```python
concat(other)       # Append another list
zip(other)          # Pair with another list
zip_with(other, fn) # Pair and transform
interleave(other)   # Alternate elements
```

---

## 3. LLM Service

### 3.1 Basic Call

```python
result = llm.call(
    prompt: string,          # Required
    schema: object           # Required - output structure
)

# Access result
result.field_name
```

### 3.2 Full Options

```python
result = llm.call(
    # Required
    prompt: string,
    schema: {
        field: type,
        ...
    },
    
    # Optional
    model: string,           # Defaults to "claude-sonnet"
    system: string,          # System prompt
    max_tokens: int,         # Defaults to 4096
    temperature: float       # Defaults to 0.0
)
```

`stop`, `cache`, and `timeout` kwargs are not read by the LLM service —
only `prompt`, `schema`, `model`, `system`, `max_tokens`, and `temperature`
are.

### 3.3 Schema Types

```python
schema: {
    # Primitives
    name: string,
    age: int,
    score: float,
    active: bool,
    
    # Enums
    status: enum(pending, active, done),
    
    # Lists
    tags: list(string),
    items: list({name: string, price: float}),
    
    # Optional fields
    nickname: string?,
    metadata: object?,
    
    # Nested objects
    address: {
        street: string,
        city: string,
        zip: string
    },
    
    # Constrained
    rating: int(min: 1, max: 5),
    email: string(format: email),
    url: string(format: url)
}
```

---

## 4. MCP Services

Services are registered by the host application (`Runtime.RegisterService`,
see [Extension Points](#9-extension-points)), not built into the SLOP
runtime itself. The only service registered by default is `llm` (section 3).
`web`, `filesystem`, `db`, and `git` below are illustrative names for
services a host might register — they ship with nothing out of the box.

### 4.1 Service Call Syntax

```python
result = service_name.tool_name(args...)
result = service_name.tool_name(named: value, ...)
result = service_name.tool_name(positional, named: value)
```

### 4.2 Common Services

#### Web

```python
# Search
results = web.search(query, limit: 10)
# Returns: [{url, title, snippet}, ...]

# Fetch page
page = web.fetch(url)
# Returns: {url, status, headers, text, html}

# HTTP methods
response = http.get(url, headers: {})
response = http.post(url, body: {}, headers: {})
response = http.put(url, body: {}, headers: {})
response = http.delete(url, headers: {})
# Returns: {status, headers, body, json}
```

#### Filesystem

```python
# Read
content = filesystem.read(path)
lines = filesystem.read_lines(path)
data = filesystem.read_json(path)

# Write
filesystem.write(path, content)
filesystem.write_json(path, data)
filesystem.append(path, content)

# Operations
filesystem.exists(path)
filesystem.delete(path)
filesystem.copy(src, dest)
filesystem.move(src, dest)
filesystem.mkdir(path)
filesystem.list(path)
filesystem.glob(pattern)

# Info
info = filesystem.stat(path)
# Returns: {size, created, modified, is_dir}
```

#### Database

```python
# Query
rows = db.query(sql, params: [])
row = db.query_one(sql, params: [])

# Execute
result = db.execute(sql, params: [])
# Returns: {rows_affected, last_insert_id}

# Transaction
db.begin()
db.commit()
db.rollback()
```

#### Git

```python
git.clone(url, path)
git.pull(path)
git.push(path)
git.commit(path, message)
git.branch(path, name)
git.checkout(path, branch)
git.status(path)
git.diff(path)
git.log(path, limit: 10)
```

---

## 5. Generators

For procedural generation (world building, etc.):

### 5.1 Random

```python
random_seed(value)       # Set seed for reproducibility
random_int(min, max)     # Random integer in range
random_float(min, max)   # Random float in range
random_choice(list)      # Random element
random_choices(list, n)  # n random elements
random_shuffle(list)     # Shuffled copy
random_chance(p)         # True with probability p
random_weighted(map)     # Weighted random choice
# e.g., random_weighted({common: 0.7, rare: 0.2, epic: 0.1})

random_uuid()            # Random UUID
random_hex(n)            # Random hex string of length n
```

### 5.2 Generators

```python
gen_name()                # Random person name — gender kwarg is accepted
                           # but currently ignored
gen_first_name()
gen_last_name()

gen_email()                # Random email
gen_phone()                # Random phone number

gen_word()                  # Random word
gen_words(n)                 # n random words
gen_sentence()               # Random sentence
gen_paragraph()               # Random paragraph
gen_lorem(words: 100)         # Lorem ipsum text

gen_color()                  # Random color (hex)
gen_rgb()                    # RGB tuple
```

There is no `gen_address`, `gen_company`, `gen_domain`, `gen_url`,
`gen_date`, `gen_datetime`, or `gen_time` generator, and there are no
world-building generators (names, items, enemies, loot) — those are not
implemented.

---

## 6. Utility Functions

### 6.1 Time

```python
now()                    # Current timestamp
today()                  # Current date
time_parse(string, format?)
time_format(timestamp, format)
time_add(timestamp, duration)
time_diff(t1, t2)        # Difference in seconds
sleep(seconds)           # Pause execution
```

### 6.2 JSON

```python
json_parse(string)       # Parse JSON
json_stringify(value)    # To JSON string
json_stringify(value, indent: 2)  # Pretty print
```

### 6.3 Encoding

```python
base64_encode(string)
base64_decode(string)
url_encode(string)
url_decode(string)
html_escape(string)
html_unescape(string)
```

### 6.4 Hashing

```python
hash_md5(string)
hash_sha256(string)
hash_sha512(string)
hash_hmac(string, key, algorithm?)
```

### 6.5 Regex

```python
regex_match(pattern, string)    # First match or none
regex_find_all(pattern, string) # All matches
regex_replace(pattern, string, replacement)
regex_split(pattern, string)
regex_test(pattern, string)     # Returns bool
```

### 6.6 Validation

```python
validate_email(string)   # Returns bool
validate_url(string)
validate_uuid(string)
validate_json(string)
```

---

## 7. Control

### 7.1 Logging

```python
log_debug(message, data?)
log_info(message, data?)
log_warn(message, data?)
log_error(message, data?)
```

### 7.2 Storage

```python
store_get(key)           # Get value or none
store_set(key, value)    # Set value
store_delete(key)        # Remove value
store_exists(key)        # Check exists
store_keys(prefix?)      # List keys
```

### 7.3 Assertions

```python
assert(condition)        # Fail if false
assert(condition, message)
assert_eq(a, b)          # Fail if a != b
assert_ne(a, b)          # Fail if a == b
assert_true(x)
assert_false(x)
assert_none(x)
assert_not_none(x)
```

---

## 8. Script Control

### 8.1 Output

```python
emit(value)              # Primary output
emit(key: value, ...)    # Named outputs
emit({...})              # Map output

print(value)             # Debug output (not captured)
```

### 8.2 Flow Control

```python
stop                     # End execution normally
stop with rollback       # End and rollback changes

error(message)           # Raise error
error(message, data)     # Raise with data
```

### 8.3 Checkpoints

There is no `checkpoint()` built-in. Script checkpointing and resume are a
CLI feature: run with `--checkpoint-dir <dir>`, then resume with
`slop resume <checkpoint>` — see [Safety & Resource Limits](/docs/advanced/safety).

---

---

## 9. Extension Points

### 9.1 Custom Functions

In host application (`pkg/slop.Runtime`):

```go
runtime.RegisterBuiltin("my_func", func(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
    // Implementation
    return result, nil
})
```

Usage in SLOP:

```python
result = my_func(arg1, arg2)
```

### 9.2 Custom Services

In host application:

```go
runtime.RegisterService("my_service", MyServiceHandler{})
```

Usage in SLOP:

```python
result = my_service.method(args)
```
