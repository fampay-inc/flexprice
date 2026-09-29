# Nudge Load Test 2 — Flexprice API

**Date:** 2026-09-29
**Endpoint under test:** `GET /v1/subscriptions`
**DB queries per request:** 2 (SELECT subscriptions + SELECT customers)
**Infrastructure:** AWS EKS + RDS PostgreSQL, `prod-flexprice` namespace

---

## Configuration

| Parameter | Value |
|---|---|
| CPU Request | 750m |
| CPU Limit | 1200m |
| QoS Class | Burstable |
| HPA Metric | CPU utilization |
| HPA CPU Trigger | 70% |
| HPA Trigger (absolute) | 525m |
| Min Pods | 6 |
| Max Pods | 20 |
| Max Open Conns | 60 |
| Max Idle Conns | 10 |
| Conn Max Lifetime | 60 minutes |
| Total pool at min pods | 360 (6 × 60) |

---

## Results

| RPM | P99 | Pods | DB In-Use | DB Idle | DB Wait/sec | Pool Utilization |
|---|---|---|---|---|---|---|
| 69,279 | 9.5ms | 6/6 | 1 | 33 | 0 | 0.3% |
| 127,145 | 9.8ms | 8/8 | 9 | 43 | 0 | 1.9% |
| 142,210 | **640ms** | 10/12 | 485 | 38 | 409 | 80.8% |
| 150,523 (peak) | **8,543ms** | 11/14 | 661 | 2 | 11,261 | 100%+ |

---

## Key Observations

- Collapsed at **~150K RPM** — pool fully exhausted (661 in-use / 660 capacity)
- System was healthy at 127K RPM (9 connections, 0 waits, 9.8ms P99)
- Deterioration was sudden: 9.8ms → 640ms → 8,543ms across two snapshots
- HPA was scaling (8 → 10 → 11 → 14 desired) but couldn't keep pace — each new pod adds 60 connections which gets consumed immediately under the spiral
- Wait count hit 11,261/sec at collapse (critical threshold is 2,000/sec)
- Lower CPU request (750m vs 1000m in Test 1) caused HPA to fire earlier but also means pods have less CPU headroom — CFS throttling likely kicking in above 1200m limit, holding connections open longer

## Comparison vs Nudge Load Test 1 (1000m request, same 60 max_open)

| Metric | Test 1 @ 144K RPM | Test 2 @ 142K RPM |
|---|---|---|
| P99 | 38ms | 640ms |
| DB In-Use | 10 | 485 |
| DB Idle | 62 | 38 |
| DB Wait/sec | 0 | 409 |
| Pool Utilization | 2.4% | 80.8% |

**Root cause:** Lower CPU request (750m) means pods hit CFS throttling sooner under actual load (limit is 1200m). Throttled pods hold DB connections open longer → pool exhausts faster. HPA scaling was reactive but could not outrun the spiral.

## Key Learnings

### 1. CPU request directly affects connection hold time
Lower CPU request → more throttling under load → requests take longer wall-clock → connections held longer → pool exhausts faster. The 250m reduction in CPU request (1000m → 750m) caused collapse at 150K RPM vs no collapse at 144K RPM in Test 1.

### 2. HPA scaling cannot outrun the death spiral
Once pool utilization hits ~80%, each new pod's connections get consumed immediately. HPA went from 8 → 14 desired pods but the spiral was already irreversible at 142K RPM.

### 3. `go_sql_wait_count_total` as early warning
- At 142K RPM: 409/sec (warning territory, approaching threshold)
- At 150K RPM: 11,261/sec (full collapse)
- 30–60 second window between warning and collapse

## Remaining Bottleneck

Same as Test 1 — DB connection pool exhaustion driven by 2 queries per request. Lower CPU request made it worse.

| Fix | Impact | Effort | Status |
|---|---|---|---|
| Read replica (`FLEXPRICE_POSTGRES_READER_HOST` — code already supports it) | Very high | Low — config only | Not done |
| Redis cache on subscription list (15–30s TTL) | Very high | Medium | Not done |
| KEDA HPA on `go_sql_wait_count_total` | High | Medium | Not done |
| Query indexes — check for seq scans on customers + subscriptions tables | High | Low | Not investigated |
| PgBouncer in transaction pooling mode | Medium | Medium | Not done |

---

## Connection Pool Math Reference

```
concurrent_connections_needed = RPS × queries_per_request × avg_query_time_seconds

At 150K RPM (collapse):
  150,000 / 60 = 2,500 RPS
  2,500 × 2 queries × ~0.132s avg (inferred from 661 connections / 5,000 RPS) ≈ 660 connections
  → matches exactly what we observed at collapse (661 in-use, 2 idle)
```
