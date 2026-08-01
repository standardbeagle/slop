---
written_at: 2026-07-15T04:44:00Z
source_event: task:01KT0Q0RJS48BP3TSMYC8571HS
module: slop
category: best-practices
confidence: high
sources:
  - task:01KT0Q0RJS48BP3TSMYC8571HS
  - workflow:01KXJ163JFR5MGBCZVGMCDR61X
  - git:9f2711636c73d464d5f745675fc139ec32747d23
tags: [security, subprocess, environment, executable-resolution]
status: steering
recurrence: 1
---

# Isolate child environments without breaking executable resolution

## Lesson

Resolve an executable with the trusted host context, then give the child an explicit environment; lookup policy and inherited environment are separate controls.

## What didn't work

Leaving `exec.Cmd.Env` nil made command transports inherit every host variable. Treating environment isolation as equivalent to removing host-side executable lookup would also break configured command discovery.

## Why it recurs

Process APIs commonly use a nil environment as an inheritance signal while resolving the executable before launch, so one constructor can silently combine two different trust boundaries.

## Apply when

Launching configured subprocesses, plugins, MCP servers, hooks, or tools from a host process that may hold credentials.

## Prevention

Construct the command through one helper, set a non-nil environment from the explicit allow-list even when it is empty, and test both invariants: no default inheritance and successful host-side executable resolution.
