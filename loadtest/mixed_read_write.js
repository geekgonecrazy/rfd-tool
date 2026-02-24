// K6 load test for rfd-tool.
//
// This script hammers the API with balanced concurrent reads and writes to
// demonstrate the throughput and reliability difference when using separate
// read/write SQLite connection pools (with WAL) versus a single shared pool.
//
// The read and write VU counts are intentionally balanced (10 + 10) so that
// both pools are under heavy simultaneous pressure — this is where the
// separated-pool architecture shines, because WAL-mode reads never block on
// the dedicated write connection.
//
// Environment variables:
//   TOKEN      – JWT session token (from cmd/generate-test-jwt)
//   API_SECRET – api-token header value (default: test-api-secret)
//   BASE_URL   – server base URL         (default: http://localhost:8877)
//
// Usage:
//   k6 run loadtest/mixed_read_write.js

import http from "k6/http";
import { check } from "k6";
import { Counter, Trend } from "k6/metrics";

// ── custom metrics ──────────────────────────────────────────────────────────
const readListLatency = new Trend("read_list_latency", true);
const readByIDLatency = new Trend("read_by_id_latency", true);
const writeLatency    = new Trend("write_latency", true);
const readListOK      = new Counter("read_list_ok");
const readByIDOK      = new Counter("read_by_id_ok");
const writeOK         = new Counter("write_ok");

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
    // List-all reads – balanced with writers
    list_readers: {
      executor: "constant-vus",
      vus: 5,
      duration: "30s",
      exec: "readRFDList",
    },
    // Per-ID reads – reads individual RFDs that writers are actively mutating
    id_readers: {
      executor: "constant-vus",
      vus: 5,
      duration: "30s",
      exec: "readRFDByID",
    },
    // Writers – equal total VU count to reads (5+5 read = 10 write)
    writers: {
      executor: "constant-vus",
      vus: 10,
      duration: "30s",
      exec: "writeRFD",
    },
  },
  thresholds: {
    http_req_failed:     ["rate<0.05"],   // <5 % errors
    read_list_latency:   ["p(95)<500"],   // p95 list reads under 500ms
    read_by_id_latency:  ["p(95)<500"],   // p95 single reads under 500ms
    write_latency:       ["p(95)<1000"],  // p95 writes under 1s
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

// readRFDList fetches the full list (GET /api/v1/rfds).
export function readRFDList() {
  const res = http.get(`${BASE_URL}/api/v1/rfds`, authHeaders());
  readListLatency.add(res.timings.duration);
  const ok = check(res, { "list read 200": (r) => r.status === 200 });
  if (ok) readListOK.add(1);
}

// readRFDByID reads a specific RFD that writers are actively creating/updating,
// exercising concurrent read + write contention on the same rows.
export function readRFDByID() {
  // Pick an ID from the writer space so we hit rows being mutated.
  const id = rfdID(__VU, __ITER);
  const params = Object.assign({}, authHeaders(), {
    // Tell K6 that 404 (not yet created) is an expected response so it does
    // not count against http_req_failed.
    responseCallback: http.expectedStatuses(200, 404),
  });
  const res = http.get(`${BASE_URL}/api/v1/rfds/${id}`, params);
  readByIDLatency.add(res.timings.duration);
  // Accept 200 (found) or 404 (not yet written) – both are correct.
  const ok = check(res, {
    "by-id read 200 or 404": (r) => r.status === 200 || r.status === 404,
  });
  if (ok) readByIDOK.add(1);
}

// writeRFD creates/updates an RFD via the api-token endpoint
// (POST /api/v1/rfds/:id) which uses requireAPISecret, not session auth.
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
