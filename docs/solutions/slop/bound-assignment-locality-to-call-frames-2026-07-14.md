---
written_at: 2026-07-14T15:55:06Z
source_event: task:01KT0Q0C16EDFC0XF3MAK29ME9
module: slop
category: logic-errors
confidence: high
sources:
  - task:01KT0Q0C16EDFC0XF3MAK29ME9
  - workflow:01KT0Q0C16PJRC3CY093W56TMY#attempt-01KXGN0ZK2E85JPHS9WYXPRASQ
  - git:8123a95
  - git:9455817
tags: [lexical-scope, assignment, call-frame, block-scope, evaluator]
status: steering
recurrence: 1
---

# Bound assignment locality to call frames

## Lesson

Plain identifier assignment should update an existing binding only inside the current call frame, then create a local binding rather than crossing into a closure or global parent; compound assignment may intentionally retain outward-update semantics.

## What didn't work

Walking every parent scope made plain assignment mutate captured or global variables. Binding only in the active scope fixed shadowing but broke loop accumulators because loop bodies use nested block scopes.

## Why it recurs

The scope parent chain represents both block nesting and call/closure boundaries, but assignment semantics distinguish those boundaries. A generic scope `Set` or `Update` cannot express both rules.

## Apply when

Changing assignment, function calls, closures, loop scopes, or any evaluator feature that pushes an enclosed scope.

## Prevention

Track the base scope of each evaluation frame. For plain assignment, search from the active scope through that base scope and stop there; add tests that jointly cover function shadowing, compound outer updates, and loop-body accumulation before accepting the change.
