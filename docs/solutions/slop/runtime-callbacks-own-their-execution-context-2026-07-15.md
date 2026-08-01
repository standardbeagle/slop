---
written_at: 2026-07-15T02:16:44Z
source_event: task:01KT0Q0H2R1MEW1CDW06AQ6XCJ
module: slop
category: architecture
confidence: high
sources:
  - task:01KT0Q0H2R1MEW1CDW06AQ6XCJ
  - workflow:01KT0Q0H2RXS676MC61MV621MD#attempt-01KXHRHT081BZKFHZHV5DYKP90
  - git:5d39203
  - git:daa0d04
  - git:8d392cb
  - git:716ae88
tags: [runtime-isolation, callback-routing, resource-limits, registry-ownership, concurrency]
status: steering
recurrence: 1
---

# Runtime callbacks own their execution context

## Lesson

Callbacks that re-enter an evaluator must be bound to the registry or runtime that registered them; process-global callback routing can charge limits to another runtime, bypass enforcement, and become nondeterministic under concurrency.

## What didn't work

Adding iteration accounting inside the pipeline callback caller fixed one-runtime tests, but the caller itself remained a mutable package global. Constructing a second runtime replaced that global, so pipelines executed by the older runtime invoked the newer evaluator and consumed the newer context's iteration budget. Correctness review rewound the task despite the original build, vet, and test gates passing.

## Why it recurs

Registries and builtins look stateless at registration time, which makes package globals tempting for late-bound evaluator access. In reality, callback invocation carries runtime-owned state: lexical frames, limits, services, and cancellation. Any process-global bridge silently collapses otherwise independent runtime instances.

## Apply when

Registering builtins, callbacks, hooks, service adapters, or resumable execution paths that call back into an evaluator or consume runtime-scoped limits.

## Prevention

Store callback bridges on the owning registry/runtime and register bound methods. Test with at least two simultaneously live runtimes configured with different limits: execute through the older runtime, assert only its context is charged, and run concurrency-sensitive paths under the race detector when applicable. For every new iterating construct, also test that each processed element consumes the shared runtime iteration budget.
