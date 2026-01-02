# SLOP

**S**tructured **L**anguage for **O**rchestrating **P**rompts

A simple, safe, and powerful scripting language designed for AI agent workflows, LLM orchestration, and prompt engineering.

```slop
# Simple AI agent in SLOP
user_input = "Hello, world!"
response = llm.call(user_input)
emit response
```

[![Documentation](https://img.shields.io/badge/docs-latest-blue)](https://standardbeagle.github.io/slop/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev)

---

## 🚀 What is SLOP?

SLOP is a domain-specific language that makes it easy to build AI agents and orchestrate language model workflows. Think of it as **Python meets AI** - simple syntax with powerful built-in features for working with LLMs.

### Why SLOP?

- **🎯 Simple** - Python-like syntax you can learn in 5 minutes
- **🔒 Safe** - Built-in protections against infinite loops and resource exhaustion
- **⚡ Fast** - Lightweight Go runtime with streaming support
- **🔌 AI-Native** - Native LLM calls, MCP integration, and schema validation
- **📦 Modular** - Organize code into reusable agents and modules

## 📖 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/standardbeagle/slop.git
cd slop
go build -o slop ./cmd/slop

# Run your first script
echo 'emit "Hello, SLOP! 🚀"' > hello.slop
./slop run hello.slop
```

### Your First Agent

Create `agent.slop`:

```slop
# Define a simple greeting agent
def greet(name):
    return "Hello, " + name + "! 👋"

# Use it
message = greet("World")
emit message
```

Run it:

```bash
./slop run agent.slop
# Output: Hello, World! 👋
```

## 💡 What Can You Build?

- **🤖 AI Chatbots** - Build conversational agents with streaming responses
- **🔄 Workflow Automation** - Orchestrate complex LLM workflows
- **📊 Data Processing** - Process and validate data for AI applications
- **🛠️ Prompt Engineering** - Test and iterate on prompts quickly
- **🌐 Web Apps** - Power backends with the SLOP runtime (see [chat app example](examples/chat-app))

## 📚 Documentation

**Full documentation:** [standardbeagle.github.io/slop](https://standardbeagle.github.io/slop/)

Quick links:
- [Getting Started](https://standardbeagle.github.io/slop/docs/getting-started/installation)
- [Language Specification](https://standardbeagle.github.io/slop/docs/language/spec)
- [Built-in Functions](https://standardbeagle.github.io/slop/docs/builtins/overview)
- [Examples](https://standardbeagle.github.io/slop/docs/examples/chat-app)
- [API Reference](https://standardbeagle.github.io/slop/docs/api/runtime)

## 🎯 Key Features

### Streaming with `emit`

Stream responses in real-time:

```slop
emit "Processing step 1..."
emit "Processing step 2..."
emit "Done! ✅"
```

### Native LLM Integration

Call language models directly:

```slop
response = llm.call({
    "messages": [{"role": "user", "content": "Hello!"}],
    "model": "claude-3-5-sonnet"
})
emit response
```

### Schema Validation

Validate data automatically:

```slop
schema = {
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "age": {"type": "integer", "minimum": 0}
    },
    "required": ["name"]
}

# Validation happens automatically
validate(user_data, schema)
```

### Safety Built-in

Automatic protections for production use:

```slop
# Loops are automatically limited
for i in range(1000000):  # Safe - won't run forever
    process(i)

# Timeouts prevent hanging
with timeout("30s"):
    slow_operation()
```

## 🏗️ Architecture

SLOP is built with a clean, extensible architecture:

- **Lexer** - Tokenizes SLOP source code
- **Parser** - Builds an Abstract Syntax Tree (AST)
- **Evaluator** - Executes the AST with a Go runtime
- **Built-ins** - Rich standard library for common tasks
- **Safety** - Automatic limits and protections

All components are well-tested with 200+ unit tests.

## 📦 Example: Chat Application

A complete AI chat app with React + SLOP backend:

```bash
cd examples/chat-app
./start.sh
# Frontend: http://localhost:3000
# Backend: http://localhost:8080
```

Features:
- Real-time streaming responses
- Multiple AI agents
- Vercel AI SDK integration
- Beautiful modern UI

[View full example →](examples/chat-app)

## 🤝 Contributing

Contributions are welcome! Some ways to help:

- 🐛 Report bugs or request features
- 📖 Improve documentation
- 🔧 Submit pull requests
- 💡 Share your SLOP agents

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🔗 Links

- **Documentation**: [standardbeagle.github.io/slop](https://standardbeagle.github.io/slop/)
- **GitHub**: [github.com/standardbeagle/slop](https://github.com/standardbeagle/slop)
- **Issues**: [github.com/standardbeagle/slop/issues](https://github.com/standardbeagle/slop/issues)
- **Discussions**: [github.com/standardbeagle/slop/discussions](https://github.com/standardbeagle/slop/discussions)

---

**Built with ❤️ by the SLOP community**
