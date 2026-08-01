---
written_at: 2026-07-15T02:03:48Z
source_event: task:01KT0Q0EBABWSH35SDBCY93634
module: slop
category: logic-errors
confidence: high
sources:
  - task:01KT0Q0EBABWSH35SDBCY93634
  - workflow:01KT0Q0EBAE3SXABH6A77NQ9Q7#attempt-01KXHR4RF3MTNSN7SQRG1A7W86
  - git:d09c860
  - git:dc6b297
tags: [lexical-scope, closure-capture, loop-binding, evaluator]
status: steering
recurrence: 1
---

# Loop closures require iteration-local bindings

## Lesson

Each loop iteration must bind its index and value in a fresh scope so closures retain that iteration's binding instead of observing later rebinding in a shared loop scope.

## What didn't work

Creating one scope for the whole loop and overwriting its variables each iteration made every closure created in the body resolve the final values.

## Why it recurs

Loop-local visibility and binding identity are different concerns: one outer loop scope hides variables after the loop, but it does not give captured variables a stable identity per iteration.

## Apply when

Changing loop evaluation, closure construction, lexical environments, or scope lifetime.

## Prevention

Push a fresh binding scope before assigning each iteration's index and value, pop it after evaluating the body, and test a list of closures invoked only after the loop has completed.
