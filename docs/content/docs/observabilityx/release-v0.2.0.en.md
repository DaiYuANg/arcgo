---
title: 'observabilityx v0.2.0'
linkTitle: 'release v0.2.0'
description: 'Breaking API update with declared metric specs and richer metric instruments'
weight: 41
---

`observabilityx v0.2.0` is a breaking release. The package now centers metric recording around declared specs and typed instruments instead of ad-hoc `AddCounter` / `RecordHistogram` calls.

## Highlights

- Reworked the core facade to use declared metric specs:
  - `obs.Counter(spec).Add(...)`
  - `obs.UpDownCounter(spec).Add(...)`
  - `obs.Histogram(spec).Record(...)`
  - `obs.Gauge(spec).Set(...)`
- Added first-class metric spec types:
  - `CounterSpec`
  - `UpDownCounterSpec`
  - `HistogramSpec`
  - `GaugeSpec`
- Added shared metric declaration options:
  - `WithDescription(...)`
  - `WithUnit(...)`
  - `WithLabelKeys(...)`
- Prometheus and OTel adapters now both enforce declared metric label schemas through the same facade contract.
- Updated internal integrations in `configx`, `eventx`, `clientx`, and `dix/metrics` to the new declared-instrument model.

## Breaking Changes

- Removed direct metric emission methods from `observabilityx.Observability`:
  - `AddCounter(...)`
  - `RecordHistogram(...)`
- Existing custom `Observability` implementations need to implement:
  - `Counter(...)`
  - `UpDownCounter(...)`
  - `Histogram(...)`
  - `Gauge(...)`
- Metric declaration is now explicit, so call sites should define metric names, units, descriptions, and label keys up front.

## Why This Change

- Declared specs make Prometheus label schemas predictable and easier to validate.
- Richer metric types cover common operational signals such as inflight counts and queue depth.
- The API is now closer to how production observability code is actually structured: declare once, record many times.

## Validation

Verified with:

```bash
go test ./observabilityx/... ./configx/... ./eventx/... ./clientx/... ./dix/... ./examples/observabilityx/...
```
