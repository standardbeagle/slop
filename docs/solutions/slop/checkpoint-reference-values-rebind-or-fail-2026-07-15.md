---
written_at: 2026-07-15T06:08:07Z
source_event: task:01KT0PZZW6SQQ8K16X6FNSACEH
module: slop
category: runtime-contracts
confidence: high
sources:
  - task:01KT0PZZW6SQQ8K16X6FNSACEH
  - workflow:01KT0PZZW6A6H19Z7XRQR1QM8T
  - git:cc47377
  - git:e4fbf99
tags: [checkpoint, serialization, runtime-references, fail-loudly]
status: steering
recurrence: 1
---

# Checkpoint reference values must rebind or fail

## Lesson

Checkpoint data may persist the stable identity of a runtime-backed value, but restore must resolve that identity to a live dependency before returning the value. If resolution is impossible—or a runtime value has no stable restore contract—the checkpoint boundary must return an explicit error instead of constructing a nil-backed placeholder that fails later during use.

## Why it recurs

Serialization can preserve enough metadata to reconstruct a value's shape while losing the external object that makes it executable. A structurally valid placeholder then defers the real checkpoint failure into an unrelated method call, obscuring both cause and recovery.

## Apply when

Adding checkpoint support for services, bound methods, modules, handles, callbacks, open resources, or any value whose behavior depends on process-local runtime state.

## Prevention

Define each runtime-backed type's checkpoint contract explicitly: serialize a stable reference and require successful rebinding on restore, or reject it as unsupported during serialization. Test successful rebinding and missing-reference failure together, including derived values such as bound methods that carry the same dependency indirectly.
