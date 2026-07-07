# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-01-15

### Added

- **External Service API**: New public types for implementing custom services:
  - `Service` interface for creating services callable from SLOP scripts
  - `RegisterExternalService()` method on Runtime for registering custom services
  - `Value` type aliases for all SLOP value types (StringValue, NumberValue, etc.)
  - `ValueToGo()` and `GoToValue()` conversion functions
  - Constructor functions: `NewStringValue()`, `NewNumberValue()`, `NewListValue()`, etc.

This enables external packages to create custom services that integrate with the SLOP runtime,
allowing SLOP scripts to call external functionality via `service_name.method(args...)` syntax.

Example:
```go
type MyService struct{}

func (s *MyService) Call(method string, args []slop.Value, kwargs map[string]slop.Value) (slop.Value, error) {
    // Handle method calls
    return slop.NewStringValue("result"), nil
}

rt := slop.NewRuntime()
rt.RegisterExternalService("myservice", &MyService{})
```

```slop
result = myservice.method(arg: "value")
emit result
```

## [0.1.2] - 2026-01-14

### Added

- Checkpoint/resume functionality for long-running SLOP scripts
- JSON schema for checkpoint format

## [0.1.1] - 2026-01-10

### Fixed

- Include cmd and pkg directories in repository

### Changed

- Renamed module to github.com/standardbeagle/slop

## [0.1.0] - 2026-01-09

### Added

- Initial release of SLOP language
- Core language features: variables, functions, loops, conditionals
- Built-in functions for string, list, and map operations
- LLM integration via `llm.call()`
- MCP server integration via `ConnectMCP()`
- Safety features: execution limits, timeouts
- Testing framework with script-based tests
- Chat application example

[0.2.0]: https://github.com/standardbeagle/slop/releases/tag/v0.2.0
[0.1.2]: https://github.com/standardbeagle/slop/releases/tag/v0.1.2
[0.1.1]: https://github.com/standardbeagle/slop/releases/tag/v0.1.1
[0.1.0]: https://github.com/standardbeagle/slop/releases/tag/v0.1.0
