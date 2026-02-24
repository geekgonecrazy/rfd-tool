// K6 load test for rfd-tool.
//
// This script hammers the API with concurrent reads and writes to demonstrate
// the throughput difference when using separate read/write SQLite connection
// pools (with WAL) versus a single shared pool.
//
// Environment variables:
//   TOKEN     – JWT session token (from cmd/generate-test-jwt)
//   API_SECRET – api-token header value (default: test-api-secret)
//   BASE_URL  – server base URL         (default: http://localhost:8877)
//
// Usage:
//   k6 run loadtest/mixed_read_write.js

import http from "k6/http";
import { check } from "k6";
import { Counter, Trend } from "k6/metrics";

// ── custom metrics ──────────────────────────────────────────────────────────
const readLatency  = new Trend("read_latency",  true);
const writeLatency = new Trend("write_latency", true);
const readOK       = new Counter("read_ok");
const writeOK      = new Counter("write_ok");

// ── configuration ───────────────────────────────────────────────────────────
const BASE_URL   = __ENV.BASE_URL   || "http://localhost:8877";
const TOKEN      = __ENV.TOKEN      || "";
const API_SECRET = __ENV.API_SECRET || "test-api-secret";

// Each VU writes its own unique RFD IDs (offset by VU id) to avoid conflicts.
// We pad to 4 digits to match the rfd-tool ID format.
function rfdID(vu, iter) {
  // Start IDs at 1000 + VU offset so we stay out of the way of real data.
  return String(1000 + vu * 100 + (iter % 100)).padStart(4, "0");
}

export const options = {
  scenarios: {
    // Heavy concurrent reads – no sleep, maximum throughput
    readers: {
      executor: "constant-vus",
      vus: 20,
      duration: "30s",
      exec: "readRFDs",
    },
    // Steady stream of writes – no sleep, maximum throughput
    writers: {
      executor: "constant-vus",
      vus: 5,
      duration: "30s",
      exec: "writeRFD",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],      // <5 % errors
    read_latency:    ["p(95)<500"],       // p95 reads under 500ms
    write_latency:   ["p(95)<1000"],      // p95 writes under 1s
  },
};

// ── helpers ─────────────────────────────────────────────────────────────────
function authHeaders() {
  return {
    headers: {
      Authorization: TOKEN,
      "Content-Type": "application/json",
    },
  };
}

function apiSecretHeaders() {
  return {
    headers: {
      "api-token": API_SECRET,
      "Content-Type": "application/json",
    },
  };
}

// ── scenarios ───────────────────────────────────────────────────────────────

// readRFDs fetches the list of RFDs (GET /api/v1/rfds) — no sleep for max throughput
export function readRFDs() {
  const res = http.get(`${BASE_URL}/api/v1/rfds`, authHeaders());
  readLatency.add(res.timings.duration);
  const ok = check(res, { "read status 200": (r) => r.status === 200 });
  if (ok) readOK.add(1);
}

// writeRFD creates/updates an RFD via the api-token endpoint
// (POST /api/v1/rfds/:id) which uses requireAPISecret, not session auth.
// No sleep for max throughput.
export function writeRFD() {
  const id = rfdID(__VU, __ITER);
  const body = JSON.stringify({
    id:        id,
    title:     `Load Test RFD ${id} iter ${__ITER}`,
    authors:   ["loadtest@example.com"],
    state:     "draft",
    tags:      ["loadtest"],
    content:   `Load test content for RFD ${id}. Iteration ${__ITER}. VU ${__VU}. Timestamp ${Date.now()}.`,
    contentMD: `# RFD ${id}\n\nLoad test body.\n`,
  });

  const res = http.post(
    `${BASE_URL}/api/v1/rfds/${id}?skip_discussion=true`,
    body,
    apiSecretHeaders(),
  );
  writeLatency.add(res.timings.duration);
  const ok = check(res, { "write status 200": (r) => r.status === 200 });
  if (ok) writeOK.add(1);
}
