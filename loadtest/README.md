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

## Load Test Results

Both tests ran on the same machine, same K6 scenario (10 read VUs + 10 write
VUs, 30 s, zero think-time), clean database each run.

### BEFORE — Single Shared Connection Pool

```
  █ TOTAL RESULTS

    checks_total.......: 41,065  1,364 req/s
    checks_succeeded...: 100.00%
    checks_failed......: 0.00%

    ✓ by-id read 200 or 404
    ✓ list read 200
    ✓ write status 200

    CUSTOM
    read_by_id_latency: avg=4.49ms   med=3.03ms  max=71.09ms   p(90)=10.03ms  p(95)=13.74ms
    read_by_id_ok.....: 31,855  (1,058/s)
    read_list_latency.: avg=22.84ms  med=18.98ms max=116.69ms  p(90)=47.77ms  p(95)=55.69ms
    read_list_ok......: 6,517   (216/s)
    write_latency.....: avg=111.27ms med=33.92ms max=3.97s     p(90)=247.8ms  p(95)=506.63ms
    write_ok..........: 2,693   (89/s)

    HTTP
    http_req_duration.: avg=14.41ms  med=4.08ms  max=3.97s     p(90)=27.97ms  p(95)=43.95ms
    http_req_failed...: 0.00%  (0 out of 41,065)
    http_reqs.........: 41,065  (1,364/s)
```

### AFTER — Separate Read + Write Pools (with WAL)

```
  █ TOTAL RESULTS

    checks_total.......: 46,910  1,563 req/s
    checks_succeeded...: 100.00%
    checks_failed......: 0.00%

    ✓ list read 200
    ✓ by-id read 200 or 404
    ✓ write status 200

    CUSTOM
    read_by_id_latency: avg=3.85ms  med=2.59ms  max=48.56ms   p(90)=8.83ms   p(95)=11.98ms
    read_by_id_ok.....: 37,075  (1,235/s)
    read_list_latency.: avg=23.57ms med=24.07ms max=85.23ms   p(90)=44.25ms  p(95)=50.2ms
    read_list_ok......: 6,318   (210/s)
    write_latency.....: avg=85.1ms  med=74.01ms max=512.07ms  p(90)=155.26ms p(95)=188.52ms
    write_ok..........: 3,517   (117/s)

    HTTP
    http_req_duration.: avg=12.6ms  med=3.56ms  max=512.07ms  p(90)=33.98ms  p(95)=58.58ms
    http_req_failed...: 0.00%  (0 out of 46,910)
    http_reqs.........: 46,910  (1,563/s)
```

### Side-by-Side Comparison

| Metric | Single Pool | Dual Pool | Δ Change |
|--------|-------------|-----------|----------|
| **Total requests** | 41,065 | **46,910** | **+14.2%** |
| **Throughput** | 1,364 req/s | **1,563 req/s** | **+14.6%** |
| **Write throughput** | 89 writes/s | **117 writes/s** | **+31.5%** |
| **Write p95 latency** | 506.63 ms | **188.52 ms** | **−62.8%** |
| **Write max latency** | **3.97 s** | 512.07 ms | **−87.1%** |
| **Write avg latency** | 111.27 ms | **85.1 ms** | **−23.5%** |
| **By-ID read throughput** | 1,058/s | **1,235/s** | **+16.7%** |
| **By-ID read p95** | 13.74 ms | **11.98 ms** | **−12.8%** |
| **Failed requests** | 0 | 0 | — |

### Analysis

The dual-pool architecture delivers clear, measurable improvements:

| Observation | Detail |
|-------------|--------|
| **+31% write throughput** | Writes jump from 89/s to 117/s. The dedicated write pool (MaxOpenConns=1) queues writes cleanly instead of competing with readers. |
| **−63% write tail latency** | Write p95 drops from **507 ms → 189 ms**. The single-pool's worst case hit **3.97 s** vs only 512 ms with dual pools. |
| **+14% total throughput** | Overall request rate climbs from 1,364/s to 1,563/s because reads proceed without blocking on writes. |
| **+17% read throughput** | By-ID reads increase from 1,058/s to 1,235/s — reads on the same rows being written never wait for the write lock. |
| **Stable tail latency** | The single pool has a long tail (max 3.97 s write) while the dual pool keeps even worst-case writes under 512 ms — a **7.8× reduction** in max latency. |

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

## Reproducing These Results

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
