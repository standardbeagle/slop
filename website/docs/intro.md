---
sidebar_position: 1
title: Introduction
description: SLOP is a sandboxed execution environment for LLM-generated code — scripts pause to editable JSON checkpoints and resume with no work lost.
---

# Introduction to SLOP

**SLOP** (Structured Language for Orchestrating Prompts) is a sandboxed execution environment for LLM-generated code.

Scripts run under hard limits on iterations, LLM calls, API calls, duration, and cost. Any script can pause mid-run — and when it does, the entire execution state is written to a plain JSON checkpoint: the source code, the pause position, every variable in every scope, the call stack, and all output emitted so far.

That state is editable. When a generated script makes a bad API call three MCP calls into a workflow, the checkpoint gives you full freedom over how to continue:

- **Skip the call** — move `position.statement_index` past the failing statement.
- **Patch the result** — make the call yourself and paste the real response into the variable the script would have written, or drop in a placeholder and keep moving.
- **Rewrite the script** — fix the call, or patch the loop's `try`/`catch` so one failure stops killing the batch. Update `script_hash` (SHA-256 of the new source) and resume.

```bash
slop run agent.slop --checkpoint-dir ./checkpoints
# Script paused. Checkpoint saved to: ./checkpoints/20260801_193635.json

# ... a later API call fails on resume; the checkpoint is still on disk ...
# ... edit the checkpoint JSON — code, position, or variables ...

slop resume ./checkpoints/patched.json
# continues from the pause point, completed work restored
```

## What is SLOP?

SLOP is a Python-like scripting language whose runtime treats execution state as data, optimized for:
- 🤖 Building AI agents and chatbots
- 🔄 Orchestrating LLM workflows
- 🔌 Chaining MCP tool calls and external services
- 🛠️ Prompt engineering and testing
- 📊 Data processing for AI applications

## Why SLOP?

### Pause, Edit, Resume

```slop
repos = github.search(query: "mcp servers")
pause("after_fetch")   # full execution state saved as editable JSON
result = llm.call(prompt: "Summarize: " + json_stringify(repos), schema: {summary: string})
emit(result.summary)
```

A `pause` writes the whole runtime — code, stack, memory, emitted output — to a checkpoint file. Edit any of it, then `slop resume` continues from that exact point.

### Sandboxed by Default

- 🧱 Pure language — no filesystem, network, shell, or imports; scripts only touch host-registered services
- ⏱️ Automatic timeout protection
- 🔁 Loop iteration limits
- 📏 LLM call, API call, cost, and call-depth limits
- 🛡️ Schema validation

### Native AI Features

- **LLM Integration**: Call language models with `llm.call()`
- **MCP Support**: Connect MCP servers and call their tools as services
- **Streaming**: Real-time output with `emit` statements
- **Modules**: Organize agents into reusable components

## Quick Example

Here's a complete AI assistant in SLOP:

```slop
# Process a question with an LLM
user_msg = "What is SLOP?"

response = llm.call(prompt: user_msg, schema: {answer: string})

# Stream the response
emit(response.answer)
```

## What Makes SLOP Different?

| Feature | SLOP | Python | JavaScript |
|---------|------|--------|-----------|
| **Pausable, Editable Execution** | ✅ JSON checkpoints + resume | ❌ | ❌ |
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
