#!/bin/bash

# SLOP Chat Application Startup Script

set -e

echo "🚀 Starting SLOP Chat Application..."
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    exit 1
fi

# Function to kill background processes on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down..."
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    exit 0
}

trap cleanup INT TERM

# Start backend
echo "📦 Starting Go backend..."
cd backend
go mod tidy
go run main.go &
BACKEND_PID=$!
cd ..

# Wait for backend to start
sleep 2

# Start frontend
echo "🎨 Starting React frontend..."
cd frontend

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📥 Installing frontend dependencies..."
    if command -v pnpm &> /dev/null; then
        pnpm install
    else
        npm install
    fi
fi

if command -v pnpm &> /dev/null; then
    pnpm dev &
else
    npm run dev &
fi

FRONTEND_PID=$!
cd ..

echo ""
echo "✅ SLOP Chat Application is running!"
echo ""
echo "🔗 Frontend: http://localhost:3000"
echo "🔗 Backend:  http://localhost:8080"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Wait for processes
wait $BACKEND_PID $FRONTEND_PID
