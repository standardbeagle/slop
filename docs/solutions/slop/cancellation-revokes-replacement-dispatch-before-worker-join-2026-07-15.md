---
written_at: 2026-07-15T05:18:19Z
source_event: task:01KXHZ3VHMHCP3YWP06FZKTR9K
module: slop
category: concurrency
confidence: high
sources:
  - task:01KXHZ3VHMHCP3YWP06FZKTR9K
  - workflow:01KXHZ3VHMBWGDGPPYNW4FF3ZH#attempt-01KXJ3118VWBGYNQQ5WD3FJ16W
  - git:300ebb9
  - git:f32da23
tags: [cancellation, worker-pool, dispatch, cleanup, deterministic-testing]
status: steering
recurrence: 1
---

# Cancellation revokes replacement dispatch before worker join

## Lesson

In a bounded worker orchestrator, cancellation must make replacement work ineligible before the coordinator can dispatch again. A ready job-channel send can win a `select` even when cancellation is also ready, so checking cancellation only in another `select` case does not establish the terminal boundary.

## What didn't work

The first bounded MCP dial orchestrator correctly limited workers and synchronously joined them during cleanup, but result handling could re-enable dispatch while the parent context was already canceled. The coordinator could enqueue a replacement dial before selecting `ctx.Done`; cleanup then waited indefinitely when that replacement worker blocked. Build, vet, ordinary tests, and the race detector all passed before correctness review forced the cancellation/result interleaving and rewound the task.

## Why it recurs

Worker pools commonly couple capacity return with immediate replacement dispatch. Cancellation and result delivery are independent events, and Go `select` chooses among ready cases without prioritizing cancellation. Synchronous cleanup makes the latent ordering bug visible as a deadlock because every mistakenly admitted unit of work becomes part of the join obligation.

## Apply when

Coordinating bounded goroutine pools, retries, hedged requests, staggered network attempts, or any scheduler that both admits replacement work and waits for owned workers on shutdown.

## Prevention

Treat cancellation as a state transition that permanently disables the dispatch case. Check terminal state before entering scheduling selection and again immediately after receiving a result, before restoring capacity or readiness. Cleanup must cancel workers, close admission, join all owned workers, and drain or close late resources.

Use channel-controlled tests to make the race deterministic: fill the worker budget, cancel while a result is released, allow workers to observe cancellation, then assert no replacement attempt starts and the coordinator returns only after all workers exit. Do not use sleeps as evidence of ordering, and do not assume `go test -race` proves cancellation linearizability.
