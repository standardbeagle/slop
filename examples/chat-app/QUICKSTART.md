# Quick Start Guide

Get the SLOP Chat App running in 3 minutes!

## Option 1: Automated Start (Recommended)

```bash
cd examples/chat-app
./start.sh
```

That's it! The script will:
- ✅ Check dependencies
- ✅ Install frontend packages
- ✅ Start the Go backend
- ✅ Start the React frontend
- ✅ Open both on their respective ports

## Option 2: Manual Start

### Terminal 1 - Backend
```bash
cd examples/chat-app/backend
go mod tidy
go run main.go
```

### Terminal 2 - Frontend
```bash
cd examples/chat-app/frontend
pnpm install  # or npm install
pnpm dev      # or npm run dev
```

## Access the App

🌐 **Frontend**: http://localhost:3000
🔌 **Backend API**: http://localhost:8080

## Try It Out

1. Select an agent from the dropdown (assistant, code-helper, or math-helper)
2. Type a message:
   - "Hello!" → Get a friendly greeting
   - "Tell me a joke" → Get a programming joke
   - "How do I write a Python function?" → Get code examples
   - "Add 5 and 10" → Get math calculation

## What's Next?

- 📝 Edit `.slop` files in `slop-agents/` to customize agent behavior
- 🎨 Modify the frontend in `frontend/src/App.tsx`
- 🔧 Add new endpoints in `backend/main.go`
- 🤖 Create your own agent by adding a new `.slop` file

## Troubleshooting

**Port already in use?**
```bash
# Change backend port
export PORT=9000

# Change frontend port
cd frontend
# Edit vite.config.ts, change port: 3000 to desired port
```

**Dependencies not found?**
```bash
# Backend
cd backend && go mod tidy

# Frontend
cd frontend && rm -rf node_modules && pnpm install
```

**SLOP module not found?**
Make sure you're running from the SLOP project directory. The `go.mod` uses a relative replace directive.

## Architecture Overview

```
User Browser
    ↓
React App (Vercel AI SDK)
    ↓ HTTP POST /api/chat
Go Backend
    ↓ Loads & Executes
SLOP Agent Script (.slop)
    ↓ emit statements
Streamed back to user
```

Enjoy building with SLOP! 🚀
