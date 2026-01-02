---
sidebar_position: 1
---

# Introduction to SLOP

Welcome to **SLOP** (Structured Language for Orchestrating Prompts) - a domain-specific language designed to make AI agent development simple, safe, and powerful.

## What is SLOP?

SLOP is a Python-like scripting language optimized for:
- 🤖 Building AI agents and chatbots
- 🔄 Orchestrating LLM workflows
- 🛠️ Prompt engineering and testing
- 📊 Data processing for AI applications

## Why SLOP?

### Simple Python-like Syntax

```slop
# Define a simple agent
def greet(name):
    return "Hello, " + name + "! 👋"

# Use it
message = greet("World")
emit message
```

### Built-in Safety

- ⏱️ Automatic timeout protection
- 🔁 Loop iteration limits
- 📏 Rate limiting support
- 🛡️ Schema validation

### Native AI Features

- **LLM Integration**: Call language models with `llm.call()`
- **MCP Support**: Use Model Context Protocol tools
- **Streaming**: Real-time output with `emit` statements
- **Modules**: Organize agents into reusable components

## Quick Example

Here's a complete AI assistant in SLOP:

```slop
# Get user input
user_msg = user_message

# Process with LLM
response = llm.call({
    "messages": [
        {"role": "user", "content": user_msg}
    ],
    "model": "claude-3-5-sonnet"
})

# Stream the response
emit response
```

## What Makes SLOP Different?

| Feature | SLOP | Python | JavaScript |
|---------|------|--------|-----------|
| **AI-First Design** | ✅ Native LLM calls | ❌ Requires libraries | ❌ Requires libraries |
| **Safety Built-in** | ✅ Automatic limits | ⚠️ Manual | ⚠️ Manual |
| **Streaming Support** | ✅ `emit` keyword | ❌ Complex | ❌ Complex |
| **Learning Curve** | 🟢 5 minutes | 🟡 Hours | 🟡 Hours |
| **MCP Integration** | ✅ Native | ❌ Manual | ❌ Manual |

## Next Steps

<div className="button-group">
  <a className="button button--primary button--lg" href="/docs/getting-started/installation">
    Install SLOP →
  </a>
  <a className="button button--secondary button--lg" href="/docs/getting-started/first-script">
    Your First Script →
  </a>
  <a className="button button--outline button--lg" href="/docs/examples/chat-app">
    View Examples →
  </a>
</div>

## Community & Support

- 💬 [GitHub Discussions](https://github.com/standardbeagle/slop/discussions)
- 🐛 [Report Issues](https://github.com/standardbeagle/slop/issues)
- 📖 [Full Documentation](/docs/language/spec)
- 💡 [Examples](/docs/examples/chat-app)

Ready to build your first AI agent? Let's get started! 🚀
