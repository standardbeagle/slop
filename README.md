# SLOP

**S**tructured **L**anguage for **O**rchestrating **P**rompts

A sandboxed execution environment for LLM-generated code. Scripts run under hard limits on iterations, LLM calls, API calls, duration, and cost — and any script can pause mid-run. When it does, the entire execution state (source code, call stack, variables, emitted output) is written to a plain JSON checkpoint. Edit the code, change a variable, fix a bad value, then resume from the exact pause point with no completed work lost.

```slop
# Chain MCP tool calls and shell commands, pause between stages
repos = github.search(query: "mcp servers")
pause("after_fetch")   # full execution state saved as editable JSON
result = llm.call(prompt: "Summarize: " + json_stringify(repos), schema: {summary: string})
emit(result.summary)
```

[![Documentation](https://img.shields.io/badge/docs-latest-blue)](https://dev.standardbeagle.com/slop/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev)

---

## 🚀 What is SLOP?

SLOP is a scripting language and runtime built for code that LLMs write and run. The language is pure: no filesystem, no network, no shell, no imports. A script can only reach what the host explicitly hands it — LLM services, MCP connections, and custom Go services registered with the runtime. The runtime treats execution state as data: a paused script is a JSON file containing its source, its position, every variable in every scope, the call stack, and everything it has emitted so far. That makes failed runs recoverable — when a generated script breaks three MCP calls in, you patch the checkpoint and resume instead of starting over.

### Why SLOP?

- **⏸️ Pausable** - `pause("name")` snapshots the whole runtime to JSON; `slop resume` continues from that exact point
- **✏️ Editable** - Rewrite the code, change variables, or adjust the stack in the checkpoint file, then resume
- **🔒 Sandboxed** - Pure language: no filesystem, network, shell, or imports — only host-granted services. Hard limits on iterations, LLM calls, API calls, duration, cost, and call depth
- **🔌 AI-Native** - Native LLM calls, MCP server integration, and schema validation
- **🎯 Simple** - Python-like syntax an LLM (or a human) can write correctly on the first try
- **📦 Modular** - Organize code into reusable agents and modules

## ⏸️ Pause, Edit, Resume

This is the core workflow. An LLM writes a script that enriches a list of repos, with a pause before the risky stage:

```slop
repos = ["slop", "agnt", "worktrack"]
summaries = []
pause("before_enrich")
for repo in repos:
    info = github.get_repo(nme: repo)   # typo: bad kwarg, call will fail
    summaries = summaries + [info]
emit(json_stringify(summaries))
```

```bash
slop run enrich.slop --checkpoint-dir ./checkpoints
# Script paused. Checkpoint saved to: ./checkpoints/20260801_193635.json

slop resume ./checkpoints/20260801_193635.json
# Error resuming: the API call fails
```

The checkpoint is still on disk, and every part of it is plain JSON you can edit: the script source, the resume position, every variable in every scope, the control-flow stack, all emitted output. How you continue is your call:

- **Skip the call** — move `position.statement_index` past the failing statement and resume from the next one.
- **Patch the result** — make the API call yourself and paste the real response into `context.scopes[].variables` under `summaries`. A placeholder value works too, if the rest of the script can live with it.
- **Rewrite the script** — fix the kwarg, or wrap the loop body in `try`/`catch` so one bad record stops killing the batch. Set `script_hash` to the SHA-256 of the new source.

```bash
slop resume ./checkpoints/patched.json
# ["slop: execution env","agnt: browser toolkit","worktrack: task store"]
```

Execution continues from the pause point with everything before it restored — fetched data, partial results, emitted output — so a long chain of MCP calls and CLI commands survives any single failure. One constraint: resume restarts at the top-level statement after the pause, so edit statements after the pause point freely but keep the ones before it in place.

## 📖 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/standardbeagle/slop.git
cd slop
go build -o slop ./cmd/slop

# Run your first script
echo 'emit("Hello, SLOP! 🚀")' > hello.slop
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
emit(message)
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

**Full documentation:** [dev.standardbeagle.com/slop](https://dev.standardbeagle.com/slop/)

Quick links:
- [Getting Started](https://dev.standardbeagle.com/slop/docs/getting-started/installation)
- [Language Specification](https://dev.standardbeagle.com/slop/docs/language/spec)
- [Built-in Functions](https://dev.standardbeagle.com/slop/docs/builtins/overview)
- [Examples](https://dev.standardbeagle.com/slop/docs/examples/chat-app)
- [API Reference](https://dev.standardbeagle.com/slop/docs/api/runtime)

## 🎯 Key Features

### Streaming with `emit`

Stream responses in real-time:

```slop
emit("Processing step 1...")
emit("Processing step 2...")
emit("Done! ✅")
```

### Native LLM Integration

Call language models directly — output is validated against your schema:

```slop
result = llm.call(prompt: "What is the capital of France?", schema: {answer: string})
emit(result.answer)
```

### External Service Integration

Create custom services accessible from SLOP scripts:

```go
// Go code
type MemoryService struct{}

func (m *MemoryService) Call(method string, args []slop.Value, kwargs map[string]slop.Value) (slop.Value, error) {
    switch method {
    case "read":
        // Handle read
        return slop.NewStringValue("stored value"), nil
    default:
        return nil, fmt.Errorf("unknown method: %s", method)
    }
}

// Register with runtime
rt := slop.NewRuntime()
rt.RegisterExternalService("memory", &MemoryService{})
```

```slop
# SLOP script
data = memory.read(key: "my_key")
emit(data)
```

### Schema Validation

LLM output is validated against the schema you pass to `llm.call`:

```slop
user = llm.call(prompt: "Extract name and age from: Alice is 30", schema: {name: string, age: int})
emit(user.name)   # "Alice"
emit(user.age)    # 30
```

Built-in validators cover common formats: `validate_json(s)`, `validate_email(s)`, `validate_url(s)`, `validate_uuid(s)`.

### Safety Built-in

The language has no filesystem, network, shell, module-import, or environment-variable access — generated code can only call services the host registered. On top of that, loops run under explicit limits and the CLI enforces global caps:

```slop
# Bounded loop - at most 100 iterations
for item in items with limit(100):
    process(item)
```

```bash
# Hard caps on the whole run
slop run script.slop --max-iterations 10000 --max-llm-calls 20
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

- **Documentation**: [dev.standardbeagle.com/slop](https://dev.standardbeagle.com/slop/)
- **GitHub**: [github.com/standardbeagle/slop](https://github.com/standardbeagle/slop)
- **Issues**: [github.com/standardbeagle/slop/issues](https://github.com/standardbeagle/slop/issues)
- **Discussions**: [github.com/standardbeagle/slop/discussions](https://github.com/standardbeagle/slop/discussions)

---

**Built with ❤️ by the SLOP community**
