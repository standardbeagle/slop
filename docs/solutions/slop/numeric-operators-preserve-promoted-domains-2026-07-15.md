---
written_at: 2026-07-15T06:12:36Z
source_event: task:01KT0Q0KTWGZZDJTENWMBVN14S
module: slop
category: correctness
confidence: high
sources:
  - workflow:01KT0Q0KTWQQMTCHDXNFT1YBHK
  - git:7704046ba1de815a25f6b27818f45980aa9581c6
  - git:e835f7ac570a76957801c6fdb3a1b1c990aa318f
tags: [evaluator, numeric-operators, floating-point, standard-library-semantics, regression-matrix]
status: steering
recurrence: 1
---

# Numeric operators must preserve promoted domains

## Lesson

Once numeric dispatch promotes either operand to floating point, every operator in that branch must preserve the floating-point domain and semantics. Do not silently narrow operands to integers or approximate a standard operation with a loop when the language's standard library already defines the required behavior.

## What didn't work

Float modulo converted both operands to integers before applying `%`, discarding fractional values. Exponentiation repeated multiplication `int(exponent)` times, so fractional and negative exponents silently produced incorrect results.

## Why it recurs

Arithmetic implementations often share a promoted-number branch, but operator bodies may retain integer-only algorithms. Ordinary positive whole-number examples still pass, hiding domain truncation and incomplete semantics.

## Apply when

Adding or changing evaluator arithmetic, numeric promotion, operator overloading, or a hand-written implementation of a standard mathematical operation.

## Prevention

Delegate promoted operations to the language standard library where its semantics match the interpreter contract. Add a compact regression matrix across integer/float operand order and semantic boundaries: fractional operands, fractional and negative exponents, and explicit zero-divisor errors where the interpreter promises them.
