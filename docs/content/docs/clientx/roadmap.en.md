---
title: 'roadmap'
linkTitle: 'roadmap'
description: 'clientx roadmap'
weight: 90
---

## clientx Roadmap (2026-03)

## Positioning

`clientx` is a protocol-oriented client layer, not a heavyweight RPC framework.

- Keep protocol-specific APIs explicit (`http` request/response, `tcp` stream, `udp` packet).
- Unify engineering conventions instead of call shapes (timeouts, retry policies, error model, observability hooks).

## Current State

- `http`, `tcp`, and `udp` packages provide baseline capability.
- Shared retry/TLS configuration primitives are available.
- Main gaps: cross-protocol observability hooks and consistent resilience knobs.

## Version Plan (Suggested)

- `v0.3`: stabilize first-wave protocol baseline (`http/tcp/udp`) and shared engineering conventions
- `v0.4`: typed error model + observability hooks
- `v0.5`: pluggable resilience policies (backoff/jitter/circuit-breaker)

## Priority Suggestions

### P0 (Now)

- Harden `udp` behavior and tests to production-like quality.
- Define shared conventions (timeout, retry, error categories, hook lifecycle), not a unified call API.
- Fix protocol-level lifecycle edge cases and ensure tests for connection behavior.

### P1 (Next)

- Complete typed error coverage and compatibility tests across protocols.
- Add observability hooks aligned with `observabilityx`.
- Harmonize configuration semantics across protocols.

### P2 (Later)

- Add pluggable resilience strategies (backoff/jitter/circuit-breaker).
- Introduce transport extension points while keeping core lightweight.

## Non-Goals

- No one-size-fits-all abstraction that hides protocol semantics.
- No replacement of mature full-feature protocol SDKs.
- No forced dependency on one telemetry/retry implementation.

## Migration Source

- Consolidated from ArcGo global roadmap draft and current package status.
- This page is the canonical roadmap maintained in docs.
