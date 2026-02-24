# Load Testing – Read/Write Pool Performance

This directory contains a [K6](https://grafana.com/docs/k6/) load test that
demonstrates the benefits of using **separate read and write SQLite connection
pools** (with WAL mode) versus a single shared pool.

## Prerequisites

| Tool | Install |
|------|---------|
| **Go** ≥ 1.24 | <https://go.dev/dl/> |
| **K6** | `brew install k6` / `go install go.k6.io/k6@latest` / <https://grafana.com/docs/k6/latest/set-up/install-k6/> |

## Quick Start

### 1. Generate test credentials

```bash
go run ./cmd/generate-test-jwt -out /tmp/loadtest
```

This creates:

* `/tmp/loadtest/config-loadtest.yaml` – a minimal config with generated RSA keys
* `/tmp/loadtest/test-token.txt` – a signed JWT session token (valid 24 h)

### 2. Start the server

```bash
go run ./cmd/rfd-server -configFile /tmp/loadtest/config-loadtest.yaml
```

The API will be available at `http://localhost:8877`.

### 3. Run the load test

```bash
export TOKEN=$(cat /tmp/loadtest/test-token.txt)

k6 run loadtest/mixed_read_write.js
```

The test runs **10 reader VUs** (5 list + 5 by-ID) and **10 writer VUs** for
30 seconds with zero think-time. The balanced 1:1 read/write ratio creates
heavy contention on the database — exactly where separated pools shine.

K6 will print a summary with custom metrics:
`read_list_latency`, `read_by_id_latency`, `write_latency`,
`read_list_ok`, `read_by_id_ok`, `write_ok`.

## Why a Balanced Ratio?

With a read-heavy skew (e.g. 20 reads : 5 writes), the write pressure is too
low to create meaningful contention. The difference between a single pool and
dual pools only becomes clear when writes are frequent enough to actually block
readers. A 1:1 ratio ensures:

* Writes happen continuously, keeping the write lock active.
* Reads compete directly with writes for the same connection pool (single-pool)
  or proceed unblocked via a separate read-only pool (dual-pool / WAL).
* The by-ID reader scenario hits the **same rows** writers are mutating,
  maximising row-level contention.

## Comparing Before & After

### Before (single connection pool)

```bash
git stash   # or checkout the commit before the read/write pool refactor
go run ./cmd/rfd-server -configFile /tmp/loadtest/config-loadtest.yaml &

export TOKEN=$(cat /tmp/loadtest/test-token.txt)
k6 run loadtest/mixed_read_write.js 2>&1 | tee /tmp/loadtest/results-before.txt

# Stop the server
kill %1
rm -rf /tmp/loadtest/data   # clean DB for fair comparison
```

### After (separate read/write pools)

```bash
git stash pop   # or checkout the read/write pool branch
go run ./cmd/rfd-server -configFile /tmp/loadtest/config-loadtest.yaml &

export TOKEN=$(cat /tmp/loadtest/test-token.txt)
k6 run loadtest/mixed_read_write.js 2>&1 | tee /tmp/loadtest/results-after.txt

kill %1
```

### What to look for

| Metric | Expected with dual pools |
|--------|--------------------------|
| `http_req_failed` | **0 %** — no "database is locked" errors |
| `read_list_latency` p95 | Lower — reads never blocked by writes |
| `read_by_id_latency` p95 | Lower — same-row reads proceed concurrently |
| `write_latency` p95 | Similar or slightly lower |
| Total `http_reqs` | Higher — no head-of-line blocking |

With a single pool, every read and write competes for the same connection(s),
creating head-of-line blocking. With separate pools, WAL-mode SQLite allows
reads to proceed **concurrently** with writes, improving throughput and
eliminating write failures under balanced load.
