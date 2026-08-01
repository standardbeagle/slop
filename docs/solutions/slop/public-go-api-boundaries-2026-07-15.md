---
written_at: 2026-07-15T05:02:50Z
source_event: task:01KT0Q15G83EA5E06M3WPWVKYD
module: slop
category: best-practices
confidence: high
sources:
  - task:01KT0Q15G83EA5E06M3WPWVKYD
  - workflow:01KT0Q15G8RNJCCX0A5SSHPPA8
  - git:3305d5aa49a1ccc7108b52cf7402f71d42debeaa
tags: [go, public-api, internal-packages, external-package-tests]
status: steering
recurrence: 1
---

# Keep exported Go APIs self-contained

## Lesson

Every type named by an exported signature must be reachable through the public package; expose intentional aliases or public wrappers instead of requiring consumers to name `internal/*` types.

## What didn't work

The runtime facade was exported, but constructors, methods, arguments, results, and collection element types still named `internal/ast` and `internal/evaluator`. In-package tests compiled because they shared implementation visibility, leaving an API that external embedders could not express using only `pkg/slop`.

## Why it recurs

Go permits an exported declaration to mention a type from an internal package, and tests in the implementation package inherit the same blind spot. A facade can therefore look public while its type graph remains coupled to implementation packages.

## Apply when

Adding or changing an exported constructor, method, callback, interface, return value, collection, or parsed/evaluated representation in a library package.

## Prevention

- Audit the complete exported signature graph, not only top-level type names.
- Re-export identity-preserving aliases when the internal representation is intentionally part of the contract; use wrappers when representation must remain encapsulated.
- Add a `package <name>_test` compile-and-use test that imports only the public package and exercises constructors, parse/eval values, callbacks/services, contexts, and returned collections.
- Keep normal build, vet, and full test gates; the external-package test is the consumer-boundary check those gates otherwise lack.
