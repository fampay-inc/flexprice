# Nudge Load Test N — Flexprice API

**Date:** YYYY-MM-DD
**Endpoint under test:** `GET /v1/subscriptions`
**DB queries per request:** 2 (SELECT subscriptions + SELECT customers)
**Infrastructure:** AWS EKS + RDS PostgreSQL, `prod-flexprice` namespace

---

## Configurations

### Fixed across all tests

| Parameter | Value |
|---|---|
| Cluster | AWS EKS, `prod-flexprice` namespace |
| Database | RDS PostgreSQL (writer endpoint) |
| Service | `flexprice-api` Deployment |
| Max Idle Conns | 10 |
| Conn Max Lifetime | 60 minutes |
| HPA metric | CPU utilization (CPU-based) |
| HPA scale-down cooldown | default (5 min) |

### Per-test configuration

| Parameter | Test 1 | Test 2 | Test 3 |
|---|---|---|---|
| CPU Request | | | |
| CPU Limit | | | |
| QoS Class | | | |
| HPA Threshold | | | |
| HPA Trigger (absolute) | | | |
| Min Pods | | | |
| Max Open Conns | | | |
| Max Idle Conns | | | |
| Total pool at min pods | | | |

---

## Test 1 Results
**Config: Xm/Xm CPU · X% HPA · X max_open · min X pods**

| RPM | P99 | Pods | DB In-Use | DB Idle | DB Wait/sec |
|---|---|---|---|---|---|
| | | | | | |

**Outcome:**

**Root cause:**

---

## Test 2 Results
**Config: Xm/Xm CPU · X% HPA · X max_open · min X pods**

| RPM | P99 | Pods | DB In-Use | DB Idle | DB Wait/sec |
|---|---|---|---|---|---|
| | | | | | |

**Outcome:**

**Root cause:**

---

## Progression Comparison at ~XXX K RPM

| Metric | Test 1 | Test 2 | Test 3 |
|---|---|---|---|
| P99 | | | |
| DB in-use | | | |
| DB idle | | | |
| DB wait/sec | | | |
| Pool utilization | | | |

---

## Key Learnings

### 1.

### 2.

---

## Remaining Bottleneck

| Fix | Impact | Effort | Status |
|---|---|---|---|
| | | | |

---

## Connection Pool Math Reference

```
concurrent_connections_needed = RPS × queries_per_request × avg_query_time_seconds

Example at XXX K RPM:
  XXX,000 / 60 = X,XXX RPS
  X,XXX × 2 queries × X.XXXs avg = ~XXX connections
```
