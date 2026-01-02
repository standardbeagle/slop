---
sidebar_position: 1
---

# SLOP Chat Application

A full-stack AI chat application built with:
- **Backend**: Go with SLOP script execution engine
- **Frontend**: React + TypeScript with Vercel AI SDK
- **Agents**: SLOP scripts for customizable AI behavior

## Architecture

```
┌─────────────────┐      HTTP/SSE      ┌──────────────────┐
│   React App     │ ◄──────────────────► │   Go Backend     │
│ (Vercel AI SDK) │                     │  (SLOP Runtime)  │
└─────────────────┘                     └──────────────────┘
                                                 │
                                                 │ Executes
                                                 ▼
                                        ┌──────────────────┐
                                        │  SLOP Agents     │
                                        │  (.slop files)   │
                                        └──────────────────┘
```

## Features

✨ **Real-time Streaming**: Responses stream from SLOP agents using Server-Sent Events (SSE)
🔄 **Multiple Agents**: Switch between different SLOP-powered agents
🎨 **Modern UI**: Clean, responsive chat interface with Vercel AI SDK
🚀 **Fast**: Go backend with efficient SLOP script execution
🔧 **Extensible**: Add new agents by creating SLOP scripts

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- pnpm (or npm/yarn)

### 1. Start the Backend

\`\`\`bash
cd backend
go mod init slop-chat-backend
go mod tidy
go run main.go
\`\`\`

The backend will start on \`http://localhost:8080\`

### 2. Start the Frontend

\`\`\`bash
cd frontend
pnpm install
pnpm dev
\`\`\`

The frontend will start on \`http://localhost:3000\`

### 3. Open Your Browser

Navigate to \`http://localhost:3000\` and start chatting!

## Project Structure

\`\`\`
chat-app/
├── backend/
│   └── main.go                 # Go server with SLOP integration
├── frontend/
│   ├── src/
│   │   ├── App.tsx            # Main React component
│   │   ├── App.css            # Styling
│   │   ├── main.tsx           # Entry point
│   │   └── index.css          # Global styles
│   ├── index.html             # HTML template
│   ├── package.json           # Dependencies
│   ├── tsconfig.json          # TypeScript config
│   └── vite.config.ts         # Vite config
├── slop-agents/
│   ├── assistant.slop         # General assistant
│   ├── code-helper.slop       # Programming help
│   └── math-helper.slop       # Math operations
└── README.md
\`\`\`

## Creating Custom Agents

Create a new \`.slop\` file in the \`slop-agents/\` directory:

\`\`\`slop
# my-agent.slop - Description of your agent

# Access user input
user_msg = user_message

# Process the message
response = "You said: " + user_msg

# Stream response chunks with emit
emit "Processing your request...\n\n"
emit response

# Or return a final value
response
\`\`\`

**Key Variables:**
- \`user_message\`: The latest user message (string)
- \`messages\`: Full conversation history (JSON string)

**Streaming with \`emit\`:**
\`\`\`slop
emit "First chunk\n"
emit "Second chunk\n"
emit "Final chunk"
\`\`\`

## API Endpoints

### POST \`/api/chat\`

Chat with an agent (streaming)

**Request:**
\`\`\`json
{
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "agentId": "assistant"
}
\`\`\`

**Response:** Server-Sent Events (SSE) stream
\`\`\`
data: {"content":"Hello! 👋","done":false}
data: {"content":" How can I help?","done":false}
data: {"content":"","done":true}
\`\`\`

### GET \`/api/agents\`

List available agents

**Response:**
\`\`\`json
[
  {
    "id": "assistant",
    "name": "assistant",
    "description": "General AI Assistant"
  }
]
\`\`\`

## How It Works

### Frontend (Vercel AI SDK)

The \`useChat\` hook from \`ai/react\` handles message state, streaming, and input.

### Backend (Go + SLOP)

1. **Parse Request**: Extract messages and agent ID
2. **Load SLOP Script**: Read the appropriate \`.slop\` file
3. **Execute**: Run the SLOP script with user context
4. **Stream**: Send \`emit\` statements as SSE chunks
5. **Complete**: Send final result when done

## License

Part of the SLOP project. See main repository for license.
