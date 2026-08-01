---
sidebar_position: 1
title: Installation
description: Install SLOP from source with Go — clone the repository, build the binary, and verify it with a hello-world script.
---

# Installation

Get SLOP up and running in minutes!

## Prerequisites

- **Go 1.23+** - [Download Go](https://go.dev/dl/)
- **Git** (optional, for cloning)

## Install from Source

### 1. Clone the Repository

```bash
git clone https://github.com/standardbeagle/slop.git
cd slop
```

### 2. Build the Project

```bash
go mod tidy
go build -o slop ./cmd/slop
```

### 3. Verify Installation

```bash
./slop --version
```

You should see output like:
```
slop version 0.1.0
```

## Add to PATH (Optional)

### Linux/macOS

```bash
sudo mv slop /usr/local/bin/
```

### Windows

Add the `slop` directory to your PATH environment variable.

## Quick Test

Create a test file `hello.slop`:

```slop
# hello.slop
message = "Hello, SLOP! 🚀"
emit(message)
```

Run it:

```bash
slop run hello.slop
```

You should see:
```
Hello, SLOP! 🚀
```

## Next Steps

- 🎓 [Quick Start Guide](/docs/getting-started/quick-start)
- 📚 [Language Specification](/docs/language/spec)

## Troubleshooting

### `go: command not found`

Install Go from [https://go.dev/dl/](https://go.dev/dl/)

### `module not found`

Run `go mod tidy` to download dependencies.

### Permission denied

On Linux/macOS, make the binary executable:
```bash
chmod +x slop
```

Need help? [Open an issue](https://github.com/standardbeagle/slop/issues)!
