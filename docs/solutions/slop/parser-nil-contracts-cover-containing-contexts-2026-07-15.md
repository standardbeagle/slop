---
written_at: 2026-07-15T04:55:52Z
source_event: task:01KT0Q0YJE7PVEJWNGFGC74QZ5
module: slop
category: test-failures
confidence: high
sources:
  - workflow:01KXJ1KYA6C8JJGJ2YCCWZRP34#attempt:01KXJ1S0TW8393K8TH78G6HTXR
  - git:8d64f9e2a4b6513aa9ef69f0f64c506d0a19a779
  - git:e1567562792d27310e776bc6ddc8aeadac5d4314
tags: [parser, nil-propagation, containing-contexts, regression-matrix, reviewer-correctness]
status: steering
recurrence: 1
---

# Parser nil contracts must cover containing contexts

## Lesson

When an expression parser can return nil after recording an error, every AST-owning boundary must reject that nil; regression cases must exercise malformed expressions both directly and inside each containing statement context.

## What didn't work

Direct malformed collection and call coverage passed build, vet, and tests, but assignment-wrapped variants still stored a nil RHS. Correctness review caught the downstream `Program.String` and AST-walk panic path.

## Why it recurs

Nested parser fixes can make the immediate construct safe while a caller still assumes that parsing always returns a node. Tests focused only on the inner grammar production do not exercise those caller contracts.

## Apply when

Changing parser error recovery, adding a grammar production, or fixing malformed input that can return a nil expression.

## Prevention

Identify every caller that owns or stores the parsed node. Add a compact regression matrix covering the malformed expression directly and embedded in assignment, argument, collection, emit, and other applicable statement/value positions; assert parse errors plus panic-free stringification and AST walking.
