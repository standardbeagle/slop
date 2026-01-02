# Contributing to SLOP

Thank you for your interest in contributing to SLOP! 🎉

## Ways to Contribute

- 🐛 **Report Bugs** - Found an issue? [Open a bug report](https://github.com/standardbeagle/slop/issues/new?labels=bug)
- 💡 **Request Features** - Have an idea? [Suggest a feature](https://github.com/standardbeagle/slop/issues/new?labels=enhancement)
- 📖 **Improve Docs** - Documentation can always be better
- 🔧 **Submit Code** - Fix bugs or implement features
- 💬 **Join Discussions** - Share your experience and help others

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git
- Node.js 18+ (for documentation site)

### Getting Started

1. **Fork the repository**

2. **Clone your fork**
   ```bash
   git clone https://github.com/YOUR_USERNAME/slop.git
   cd slop
   ```

3. **Create a branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Install dependencies**
   ```bash
   go mod download
   ```

5. **Run tests**
   ```bash
   go test ./...
   ```

## Code Guidelines

### Go Code

- Follow standard Go conventions (`gofmt`, `golint`)
- Write tests for new features
- Keep functions focused and small
- Document exported functions and types
- All tests must pass before submitting PR

### SLOP Scripts

- Use clear, descriptive variable names
- Add comments for complex logic
- Follow the examples in `examples/`

### Documentation

- Use clear, concise language
- Include code examples
- Update relevant docs when changing features
- Test documentation locally before submitting

## Testing

Run the full test suite:

```bash
# All tests
go test ./...

# Specific package
go test ./internal/parser/...

# With coverage
go test -cover ./...

# Verbose mode
go test -v ./...
```

## Submitting Changes

1. **Ensure all tests pass**
   ```bash
   go test ./...
   ```

2. **Format your code**
   ```bash
   gofmt -w .
   ```

3. **Commit your changes**
   ```bash
   git add .
   git commit -m "Brief description of changes"
   ```

4. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **Open a Pull Request**
   - Go to the original repository
   - Click "New Pull Request"
   - Select your branch
   - Fill out the PR template
   - Link any related issues

## Pull Request Guidelines

### Good PR

- ✅ Clear description of changes
- ✅ All tests passing
- ✅ Documentation updated
- ✅ Small, focused changes
- ✅ Follows existing code style

### PR Template

```markdown
## Description
Brief description of what this PR does

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Code refactoring

## Testing
- [ ] All existing tests pass
- [ ] Added tests for new functionality
- [ ] Manually tested changes

## Checklist
- [ ] Code follows project style guidelines
- [ ] Documentation updated
- [ ] No breaking changes (or documented if unavoidable)
```

## Documentation Site

The documentation site uses Docusaurus.

### Running Locally

```bash
cd website
npm install
npm start
```

Visit http://localhost:3000

### Building

```bash
cd website
npm run build
```

## Code of Conduct

Please be respectful and constructive in all interactions. We're here to build something great together!

## Questions?

- 💬 [GitHub Discussions](https://github.com/standardbeagle/slop/discussions)
- 🐛 [GitHub Issues](https://github.com/standardbeagle/slop/issues)

Thank you for contributing! 🚀
