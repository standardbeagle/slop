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
dict(x)             # Convert pairs to dict
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
s.format(args...)   # Format string
s.pad_left(n, char?)
s.pad_right(n, char?)
s.slice(start, end?)
s.repeat(n)         # Repeat n times
s.reverse()         # Reverse string
s.lines()           # Split by newlines
s.words()           # Split by whitespace
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
list.sort(key?)     # Sort in place (mutates)
list.reverse()      # Reverse in place (mutates)
list.copy()         # Shallow copy

# Non-mutating versions
sorted(list, key?)  # Return sorted copy
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
set.symmetric_difference(other)  # a ^ b
set.issubset(other)
set.issuperset(other)
```

---

## 2. Collection Pipeline

All pipeline functions work with `|` or as methods:

```python
# These are equivalent:
items | filter(x -> x > 0) | map(x -> x * 2)
items.filter(x -> x > 0).map(x -> x * 2)
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
sort()              # Sort ascending
sort(key_fn)        # Sort by key
sort(key_fn, reverse: true)  # Sort descending
reverse()           # Reverse order
shuffle()           # Random order (seeded)
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
    model: string,           # "claude-sonnet", "claude-opus", etc.
    system: string,          # System prompt
    max_tokens: int,         # Max output tokens
    temperature: float,      # 0.0 - 1.0
    stop: list(string),      # Stop sequences
    
    # Cost control
    cache: bool,             # Cache identical requests
    timeout: duration        # Request timeout
)
```

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
random.seed(value)       # Set seed for reproducibility
random.int(min, max)     # Random integer in range
random.float(min, max)   # Random float in range
random.choice(list)      # Random element
random.choices(list, n)  # n random elements
random.shuffle(list)     # Shuffled copy
random.chance(p)         # True with probability p
random.weighted(map)     # Weighted random choice
# e.g., random.weighted({common: 0.7, rare: 0.2, epic: 0.1})

random.uuid()            # Random UUID
random.hex(n)            # Random hex string of length n
```

### 5.2 Generators

```python
gen.name()               # Random person name
gen.name(gender: "female")
gen.first_name()
gen.last_name()

gen.email()              # Random email
gen.phone()              # Random phone number
gen.address()            # Random address

gen.company()            # Company name
gen.domain()             # Domain name
gen.url()                # URL

gen.word()               # Random word
gen.words(n)             # n random words
gen.sentence()           # Random sentence
gen.paragraph()          # Random paragraph
gen.lorem(words: 100)    # Lorem ipsum text

gen.date(start?, end?)   # Random date
gen.datetime(start?, end?)
gen.time()

gen.color()              # Random color (hex)
gen.rgb()                # RGB tuple
```

### 5.3 Custom Generators

For world building:

```python
gen.fantasy_name()        # Fantasy character name
gen.tavern_name()         # "The Prancing Pony"
gen.town_name()           # "Riverdale"
gen.region_name(type)     # Based on terrain type
gen.quest_name()          # "The Lost Artifact"

gen.item(type, rarity?)   # Generate item
gen.enemy(difficulty)     # Generate enemy stats
gen.loot(level, count)    # Generate loot table
gen.dialogue(npc_type)    # Dialogue snippets
```

---

## 6. Utility Functions

### 6.1 Time

```python
now()                    # Current timestamp
today()                  # Current date
time.parse(string, format?)
time.format(timestamp, format)
time.add(timestamp, duration)
time.diff(t1, t2)        # Difference in seconds
sleep(seconds)           # Pause execution
```

### 6.2 JSON

```python
json.parse(string)       # Parse JSON
json.stringify(value)    # To JSON string
json.stringify(value, indent: 2)  # Pretty print
```

### 6.3 Encoding

```python
base64.encode(string)
base64.decode(string)
url.encode(string)
url.decode(string)
html.escape(string)
html.unescape(string)
```

### 6.4 Hashing

```python
hash.md5(string)
hash.sha256(string)
hash.sha512(string)
hash.hmac(string, key, algorithm?)
```

### 6.5 Regex

```python
regex.match(pattern, string)    # First match or none
regex.find_all(pattern, string) # All matches
regex.replace(pattern, string, replacement)
regex.split(pattern, string)
regex.test(pattern, string)     # Returns bool
```

### 6.6 Validation

```python
validate.email(string)   # Returns bool
validate.url(string)
validate.uuid(string)
validate.json(string)
validate.schema(data, schema)  # Returns {valid, errors}
```

---

## 7. Control

### 7.1 Logging

```python
log.debug(message, data?)
log.info(message, data?)
log.warn(message, data?)
log.error(message, data?)
```

### 7.2 Storage

```python
store.get(key)           # Get value or none
store.set(key, value)    # Set value
store.delete(key)        # Remove value
store.exists(key)        # Check exists
store.keys(prefix?)      # List keys
```

### 7.3 Environment

```python
env.get(name)            # Get env var or none
env.get(name, default)   # Get with default
env.mode                 # "production", "test", "development"
env.debug                # Debug mode enabled
```

### 7.4 Assertions

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

```python
checkpoint(name)         # Save state
checkpoint(name, data)   # Save with extra data

# Resume from checkpoint on restart
```

---

## 9. Agent Helpers

### 9.1 Sub-Agents

```python
result = agent.run(
    script: "path/to/agent.slop",
    input: {...},
    limits: {
        max_steps: 10,
        max_llm_calls: 20,
        timeout: 60s,
        max_cost: 5.00
    }
)
```

### 9.2 Tools

```python
tools.list()             # List available tools
tools.describe(name)     # Get tool description
tools.call(name, args)   # Dynamic tool call
```

---

## 10. Extension Points

### 10.1 Custom Functions

In host application:

```go
runtime.RegisterFunction("my_func", func(args []Value) Value {
    // Implementation
    return result
})
```

Usage in SLOP:

```python
result = my_func(arg1, arg2)
```

### 10.2 Custom Services

In host application:

```go
runtime.RegisterService("my_service", MyServiceHandler{})
```

Usage in SLOP:

```python
result = my_service.method(args)
```
