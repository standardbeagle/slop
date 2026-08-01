---
written_at: 2026-07-14T15:46:46Z
source_event: task:01KT0Q060F5D6WEYZM3G9Y6E7Y
module: slop
category: logic-errors
confidence: high
sources:
  - task:01KT0Q060F5D6WEYZM3G9Y6E7Y
  - git:cd00781
  - git:f704272
tags: [unsupported-semantics, fail-loudly, evaluator, execution-model]
status: steering
recurrence: 1
---

# Unsupported execution modifiers must fail loudly

## Lesson

When syntax promises a distinct execution model, an unsupported implementation must return an explicit error instead of accepting the syntax and running with ordinary semantics.

## What didn't work

`parallel(N)` was parsed and accepted while the evaluator ignored the modifier and ran the loop serially. Successful execution therefore concealed a semantic mismatch.

## Why it recurs

Parser support and placeholder evaluator branches can make a feature appear complete even when its defining runtime behavior is absent.

## Apply when

Adding or reviewing modifiers, options, or keywords whose meaning changes ordering, concurrency, isolation, retries, timing, or resource limits.

## Prevention

Add a negative evaluator test for every recognized-but-unsupported semantic option. Require a stable, explicit error until the promised behavior is implemented; replace that test only when runtime semantics have dedicated behavioral coverage.
