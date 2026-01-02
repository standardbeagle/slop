---
sidebar_position: 2
---

# Quick Start

Get started with SLOP in 5 minutes! 

## Your First SLOP Script

Create a file called `hello.slop`:

```slop
# A simple greeting
name = "World"
greeting = "Hello, " + name + "!"

emit greeting
```

Run it:

```bash
slop run hello.slop
```

Output:
```
Hello, World!
```

## Working with Variables

SLOP supports common data types:

```slop
# Numbers
age = 25
price = 99.99

# Strings
name = "Alice"
message = 'Single quotes work too'

# Booleans
is_active = true
is_valid = false

# None (null value)
result = none

# Lists
fruits = ["apple", "banana", "orange"]

# Maps (dictionaries)
person = {
    "name": "Bob",
    "age": 30,
    "city": "NYC"
}
```

## Control Flow

### If Statements

```slop
age = 20

if age >= 18:
    emit "You can vote! ✅"
else:
    emit "Too young to vote ❌"
```

### For Loops

```slop
# Iterate over a list
fruits = ["🍎", "🍌", "🍊"]

for fruit in fruits:
    emit fruit

# Iterate with index
for i, fruit in fruits:
    emit str(i) + ": " + fruit

# Range
for i in range(5):
    emit "Count: " + str(i)
```

## Functions

Define reusable logic:

```slop
def greet(name):
    return "Hello, " + name + "! 👋"

def add(a, b):
    return a + b

# Use functions
message = greet("Alice")
sum = add(10, 20)

emit message
emit "Sum: " + str(sum)
```

## Streaming with emit

The `emit` statement sends output in real-time:

```slop
emit "Processing..."
emit "Step 1 complete ✓"
emit "Step 2 complete ✓"
emit "Done! 🎉"
```

This is perfect for streaming AI responses!

## Built-in Functions

SLOP comes with many useful built-ins:

```slop
# String functions
upper_text = upper("hello")  # "HELLO"
lower_text = lower("WORLD")  # "world"
parts = split("a,b,c", ",")  # ["a", "b", "c"]

# Math functions
absolute = abs(-42)  # 42
maximum = max([1, 5, 3])  # 5
total = sum([1, 2, 3])  # 6

# List functions
first_item = first([1, 2, 3])  # 1
last_item = last([1, 2, 3])  # 3
length = len([1, 2, 3])  # 3
```

See the [Built-in Functions](/docs/builtins/overview) reference for more!

## Error Handling

Handle errors gracefully:

```slop
try:
    result = risky_operation()
    emit "Success: " + str(result)
catch error:
    emit "Error occurred: " + error
```

## Next Steps

Now that you know the basics:

- 🎓 [Learn the Language](/docs/language/syntax) - Deep dive into SLOP syntax
- 🔧 [Built-in Functions](/docs/builtins/overview) - Explore all available functions
- 💡 [Examples](/docs/examples/chat-app) - See SLOP in action
- 🚀 [Build an AI Agent](/docs/advanced/agents) - Create your first agent

Happy coding! 🎉
