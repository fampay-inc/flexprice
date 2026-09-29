# Nudge Load Test 1 — Flexprice API

**Date:** 2026-09-29
**Endpoint under test:** `GET /v1/subscriptions`
**DB queries per request:** 2 (SELECT subscriptions + SELECT customers)
**Infrastructure:** AWS EKS + RDS PostgreSQL, `prod-flexprice` namespace

---

## Configuration

| Parameter | Value |
|---|---|
| CPU Request | 1000m |
| CPU Limit | 1200m |
| QoS Class | Burstable |
| HPA Metric | CPU utilization |
| HPA CPU Trigger | 70% |
| HPA Trigger (absolute) | 700m |
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
| 39,788 | 8.7ms | 6/6 | 5 | 18 | 0 | 1.4% |
| 144,137 | 38ms | 7/7 | 10 | 62 | 0 | 2.4% |

**Status:** Test 4 (in progress at time of writing). No collapse at 144K RPM.

---

## Key Observations

- At 144K RPM, only 10 DB connections in-use out of 420 total capacity (7 × 60) — 2.4% pool utilization
- Zero goroutine waits at 144K RPM
- P99 38ms vs 232ms in previous config (25 max_open, Guaranteed QoS) at the same RPM
- HPA triggered correctly — scaled from 6 to 7 pods as load increased

---

## Key Learnings

### 1. max_open_conns sweet spot
- **Too low (25):** Pool saturates before DB does — waits spike fast
- **Too high (80):** Floods DB with concurrent connections, slowing queries and causing pool exhaustion via longer query times
- **60** appears better balanced for this workload

### 2. HPA triggers on the wrong metric
CPU-based HPA does not react to DB wait pressure. Pods blocked on DB have low CPU → HPA sees no problem → system collapses while HPA watches.

Recommended fix: KEDA with `go_sql_wait_count_total` as trigger (fire at ~500 waits/sec).

### 3. `go_sql_wait_count_total` is the leading indicator
Rises 30–60 seconds before P99 becomes user-visible.
- Alert threshold: **>500/sec** = warning
- Alert threshold: **>2,000/sec** = imminent collapse

```promql
sum by (job) (rate(go_sql_wait_count_total{db_name="flexprice"}[1m]))
```

### 4. The death spiral signature
```
High RPS
→ queries slow under DB load
→ connections held longer
→ pool exhausts
→ goroutines wait → go_sql_wait_count ↑
→ Go scheduler overloaded → Client:ClientRead ↑ on RDS
→ P99 explodes
→ feedback loop accelerates
```

---

## Remaining Bottleneck

Underlying driver: **2 DB queries per HTTP request at high RPS**. Fixes in priority order:

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

At 144K RPM (stable):
  144,000 / 60 = 2,400 RPS
  2,400 × 2 queries × 0.002s avg = ~10 connections
  → matches observed 10 in-use
```
