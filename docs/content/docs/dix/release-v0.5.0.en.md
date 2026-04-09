---
title: 'dix v0.5.0'
linkTitle: 'release v0.5.0'
description: 'Framework event logging, logger sugar, and shorter setup APIs'
weight: 42
---

`dix v0.5.0` focuses on public API ergonomics around framework logging and setup registration.

## Highlights

- Added `UseLogger0/1/Err0/Err1(...)` so framework loggers can be resolved from DI with the same typed style as the rest of `dix`.
- Added `UseEventLogger(...)` and `UseEventLogger0/1/Err0/Err1(...)` so callers can fully own dix internal build/start/stop/health/debug logging.
- Internal dix logging now routes through `EventLogger` when configured, instead of bypassing it with direct `slog` calls.
- Added shorter setup and hook helpers such as `Setup0`, `SetupContainer`, `SetupLifecycle`, `Setup1..6`, `OnStartFunc`, and `OnStopFunc`.
- Updated the docs and examples to recommend `UseLogger...` / `UseEventLogger...` as the primary logging APIs.

## Compatibility note

- Existing `WithLogger(...)` and `WithLoggerFrom...` APIs remain supported.
- `Observer` remains available for sidecar consumers such as metrics, but it is no longer the recommended primary hook for framework logging customization.

## Validation

Verified with:

```bash
go test ./dix/... ./examples/dix/basic ./examples/dix/metrics ./examples/dix/inspect ./examples/dix/override ./examples/dix/runtime_scope ./examples/dix/build_runtime ./examples/dix/build_failure ./examples/dix/advanced_do_bridge
```
