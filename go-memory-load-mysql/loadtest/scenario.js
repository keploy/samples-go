import http from 'k6/http';
import exec from 'k6/execution';
import { Counter, Trend } from 'k6/metrics';
import { check, sleep } from 'k6';

const isSmokeProfile = __ENV.TEST_PROFILE === 'smoke';
const MIXED_API_START_VUS = parsePositiveIntEnv('MIXED_API_START_VUS', 10);
// Default ramp lowered from [20,40,80,30] to [2,3,4,2] so local
// runs (no env override) match the keploy-CI profile validated in
// the rate-mismatch investigation: at 14+ concurrent VUs the
// recorder's mock-emit rate exceeded the host's YAML-write
// throughput by ~7x, producing either silent TCP-buffer loss or
// pipeline deadlock. 4-VU peak still spikes agent memory enough
// (combined with the unchanged LARGE_PAYLOAD ramp below) to fire
// 2-3 memory-pressure events, which is the load profile this
// sample is designed to validate.
const MIXED_API_VU_STAGE_TARGETS = parsePositiveIntListEnv(
  'MIXED_API_VU_STAGE_TARGETS',
  [2, 3, 4, 2],
  4
);
const LARGE_PAYLOAD_PREALLOCATED_VUS = parsePositiveIntEnv('LARGE_PAYLOAD_PREALLOCATED_VUS', 16);
const LARGE_PAYLOAD_MAX_VUS = parsePositiveIntEnv('LARGE_PAYLOAD_MAX_VUS', 64);
const LARGE_PAYLOAD_SIZE_MBS = (__ENV.LARGE_PAYLOAD_SIZES_MB || '1,2,4')
  .split(',')
  .map((value) => parseInt(value.trim(), 10))
  .filter((value) => Number.isFinite(value) && value > 0);
// No fallback to [1]: an explicit LARGE_PAYLOAD_SIZES_MB=0 (or any value that
// parses to ≤0) disables the large-payload cycle entirely. This is the CI
// default because MySQL LONGTEXT large-payload responses can exceed Keploy's
// in-memory mock size, causing reconstruction failures during replay.
const LARGE_PAYLOAD_SIZES = LARGE_PAYLOAD_SIZE_MBS;

const LARGE_PAYLOAD_STAGE_TARGETS = parsePositiveIntListEnv(
  'LARGE_PAYLOAD_STAGE_TARGETS',
  [2, 4, 2],
  3
);

const THRESHOLD_HTTP_FAILED_RATE = parseFloatEnv('THRESHOLD_HTTP_FAILED_RATE', 0.02);
const THRESHOLD_HTTP_P95 = parsePositiveIntEnv('THRESHOLD_HTTP_P95', 2500);
const THRESHOLD_HTTP_AVG = parsePositiveIntEnv('THRESHOLD_HTTP_AVG', 1200);
const THRESHOLD_LARGE_INSERT_P95 = parsePositiveIntEnv('THRESHOLD_LARGE_INSERT_P95', 5000);
const THRESHOLD_LARGE_GET_P95 = parsePositiveIntEnv('THRESHOLD_LARGE_GET_P95', 5000);
const THRESHOLD_LARGE_DELETE_P95 = parsePositiveIntEnv('THRESHOLD_LARGE_DELETE_P95', 3000);

// Build scenario and threshold objects conditionally so the large_payload_cycle
// is entirely absent from the k6 options when LARGE_PAYLOAD_SIZES is empty.
// k6 registers custom-metric thresholds at init time; referencing a metric
// (large_payload_*) in thresholds when its scenario never runs causes k6 to
// report a threshold-not-met error even though zero samples were collected.
const _smokeScenarios = {
  mixed_api_load: {
    executor: 'shared-iterations',
    vus: 1,
    iterations: 8,
    maxDuration: '30s',
  },
};
if (LARGE_PAYLOAD_SIZES.length > 0) {
  _smokeScenarios.large_payload_cycle = {
    executor: 'shared-iterations',
    vus: 1,
    iterations: 3,
    maxDuration: '45s',
  };
}

const _smokeThresholds = {
  http_req_failed: ['rate<0.05'],
};
if (LARGE_PAYLOAD_SIZES.length > 0) {
  _smokeThresholds.large_payload_insert_duration = ['p(95)<3000'];
  _smokeThresholds.large_payload_get_duration = ['p(95)<3000'];
  _smokeThresholds.large_payload_delete_duration = ['p(95)<2000'];
}

const _prodScenarios = {
  mixed_api_load: {
    executor: 'ramping-vus',
    startVUs: MIXED_API_START_VUS,
    stages: [
      { target: MIXED_API_VU_STAGE_TARGETS[0], duration: '15s' },
      { target: MIXED_API_VU_STAGE_TARGETS[1], duration: '30s' },
      { target: MIXED_API_VU_STAGE_TARGETS[2], duration: '45s' },
      { target: MIXED_API_VU_STAGE_TARGETS[3], duration: '15s' },
    ],
  },
};
if (LARGE_PAYLOAD_SIZES.length > 0) {
  _prodScenarios.large_payload_cycle = {
    executor: 'ramping-arrival-rate',
    startRate: 1,
    timeUnit: '1s',
    preAllocatedVUs: LARGE_PAYLOAD_PREALLOCATED_VUS,
    maxVUs: LARGE_PAYLOAD_MAX_VUS,
    stages: [
      { target: LARGE_PAYLOAD_STAGE_TARGETS[0], duration: '15s' },
      { target: LARGE_PAYLOAD_STAGE_TARGETS[1], duration: '30s' },
      { target: LARGE_PAYLOAD_STAGE_TARGETS[2], duration: '15s' },
    ],
  };
}

const _prodThresholds = {
  http_req_failed: [`rate<${THRESHOLD_HTTP_FAILED_RATE}`],
  http_req_duration: [`p(95)<${THRESHOLD_HTTP_P95}`, `avg<${THRESHOLD_HTTP_AVG}`],
};
if (LARGE_PAYLOAD_SIZES.length > 0) {
  _prodThresholds.large_payload_insert_duration = [`p(95)<${THRESHOLD_LARGE_INSERT_P95}`];
  _prodThresholds.large_payload_get_duration = [`p(95)<${THRESHOLD_LARGE_GET_P95}`];
  _prodThresholds.large_payload_delete_duration = [`p(95)<${THRESHOLD_LARGE_DELETE_P95}`];
}

export const options = isSmokeProfile
  ? { scenarios: _smokeScenarios, thresholds: _smokeThresholds }
  : { scenarios: _prodScenarios, thresholds: _prodThresholds };

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const SEGMENTS = ['startup', 'enterprise', 'retail', 'partner'];
const CATEGORIES = ['compute', 'storage', 'networking', 'security', 'analytics'];
const STATUSES = ['paid', 'paid', 'paid', 'shipped', 'pending'];
let uniqueCounter = 0;
const payloadCache = {};
const largePayloadInsertDuration = new Trend('large_payload_insert_duration', true);
const largePayloadGetDuration = new Trend('large_payload_get_duration', true);
const largePayloadDeleteDuration = new Trend('large_payload_delete_duration', true);
const largePayloadInsertedBytes = new Counter('large_payload_inserted_bytes');
const largePayloadRetrievedBytes = new Counter('large_payload_retrieved_bytes');
const largePayloadDeletedBytes = new Counter('large_payload_deleted_bytes');

function parsePositiveIntEnv(name, fallback) {
  const value = parseInt(__ENV[name] || '', 10);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function parseFloatEnv(name, fallback) {
  const value = parseFloat(__ENV[name] || '');
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function parsePositiveIntListEnv(name, fallback, expectedLength) {
  const values = (__ENV[name] || '')
    .split(',')
    .map((value) => parseInt(value.trim(), 10))
    .filter((value) => Number.isFinite(value) && value > 0);

  if (values.length === expectedLength) {
    return values;
  }

  return fallback;
}

function jsonParams() {
  return {
    headers: {
      'Content-Type': 'application/json',
    },
  };
}

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function randomItem(values) {
  return values[randomInt(0, values.length - 1)];
}

function uniqueSuffix() {
  const vu = typeof __VU === 'number' ? __VU : 0;
  uniqueCounter += 1;
  return `${vu}-${uniqueCounter}-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
}

function bytesFromMB(mb) {
  return mb * 1024 * 1024;
}

function buildLargePayload(sizeMB) {
  if (!payloadCache[sizeMB]) {
    const targetBytes = bytesFromMB(sizeMB);
    payloadCache[sizeMB] = 'X'.repeat(targetBytes);
  }

  return payloadCache[sizeMB];
}

function createCustomer(namePrefix = 'Load Customer') {
  const suffix = uniqueSuffix();
  const payload = {
    email: `customer-${suffix}@example.com`,
    full_name: `${namePrefix} ${suffix}`,
    segment: randomItem(SEGMENTS),
  };

  const response = http.post(`${BASE_URL}/customers`, JSON.stringify(payload), jsonParams());
  check(response, {
    'create customer status is 201': (r) => r.status === 201,
  });

  return response.status === 201 ? response.json() : null;
}

function createLargePayload(sizeMB) {
  const suffix = uniqueSuffix();
  const payload = buildLargePayload(sizeMB);
  const response = http.post(
    `${BASE_URL}/large-payloads`,
    JSON.stringify({
      name: `Large Payload ${suffix}`,
      content_type: 'text/plain',
      payload,
    }),
    jsonParams()
  );

  largePayloadInsertDuration.add(response.timings.duration, { size_mb: String(sizeMB) });
  largePayloadInsertedBytes.add(payload.length);

  check(response, {
    'create large payload status is 201': (r) => r.status === 201,
    'create large payload size matches': (r) =>
      r.status === 201 && r.json('payload_size_bytes') === payload.length,
  });

  return response.status === 201 ? response.json() : null;
}

function getLargePayload(id, sizeMB) {
  const response = http.get(`${BASE_URL}/large-payloads/${id}`);

  largePayloadGetDuration.add(response.timings.duration, { size_mb: String(sizeMB) });

  const expectedBytes = bytesFromMB(sizeMB);
  check(response, {
    'get large payload status is 200': (r) => r.status === 200,
    'get large payload size matches': (r) =>
      r.status === 200 &&
      r.json('payload_size_bytes') === expectedBytes &&
      r.json('payload').length === expectedBytes,
  });

  if (response.status === 200) {
    largePayloadRetrievedBytes.add(response.json('payload_size_bytes'));
  }

  return response;
}

function deleteLargePayload(id, sizeMB) {
  const response = http.del(`${BASE_URL}/large-payloads/${id}`);

  largePayloadDeleteDuration.add(response.timings.duration, { size_mb: String(sizeMB) });

  check(response, {
    'delete large payload status is 200': (r) => r.status === 200,
    'delete large payload reports deleted': (r) => r.status === 200 && r.json('deleted') === true,
  });

  if (response.status === 200) {
    largePayloadDeletedBytes.add(response.json('record.payload_size_bytes'));
  }

  return response;
}

function createProduct(namePrefix = 'Load Product') {
  const suffix = uniqueSuffix();
  const payload = {
    sku: `SKU-${suffix}`.toUpperCase(),
    name: `${namePrefix} ${suffix}`,
    category: randomItem(CATEGORIES),
    price_cents: randomInt(1200, 18000),
    inventory_count: randomInt(1200, 2500),
  };

  const response = http.post(`${BASE_URL}/products`, JSON.stringify(payload), jsonParams());
  check(response, {
    'create product status is 201': (r) => r.status === 201,
  });

  return response.status === 201 ? response.json() : null;
}

function createOrder(customerId, products) {
  const itemCount = randomInt(1, 4);
  const items = [];
  const selectedProductIDs = new Set();

  while (items.length < itemCount) {
    const product = randomItem(products);
    if (selectedProductIDs.has(product.id)) {
      continue;
    }
    selectedProductIDs.add(product.id);
    items.push({
      product_id: product.id,
      quantity: randomInt(1, 3),
    });
  }

  const payload = {
    customer_id: customerId,
    status: randomItem(STATUSES),
    items,
  };

  const response = http.post(`${BASE_URL}/orders`, JSON.stringify(payload), jsonParams());
  check(response, {
    'create order status is 201': (r) => r.status === 201,
  });

  return response.status === 201 ? response.json() : null;
}

export function setup() {
  const bootstrapCustomers = [];
  const bootstrapProducts = [];
  const bootstrapLargePayloads = [];

  for (let i = 0; i < 20; i += 1) {
    const customer = createCustomer('Bootstrap Customer');
    if (customer) {
      bootstrapCustomers.push(customer);
    }
  }

  // 150 products (up from 35) spread concurrent findOneAndUpdate operations across
  // a much larger pool. With N concurrent VUs each picking a random product,
  // P(two VUs choose the same product) ≈ N/150, which is low enough that
  // Keploy never sees two simultaneous identical SQL UPDATE+SELECT requests
  // that it cannot distinguish during mock replay.
  for (let i = 0; i < 150; i += 1) {
    const product = createProduct('Bootstrap Product');
    if (product) {
      bootstrapProducts.push(product);
    }
  }

  if (bootstrapCustomers.length === 0 || bootstrapProducts.length === 0) {
    throw new Error(`setup: bootstrap failed — customers=${bootstrapCustomers.length}, products=${bootstrapProducts.length}; cannot continue`);
  }

  const bootstrapOrders = [];
  for (let i = 0; i < 40; i += 1) {
    const customer = randomItem(bootstrapCustomers);
    const order = createOrder(customer.id, bootstrapProducts);
    if (order) {
      bootstrapOrders.push(order);
      const r = http.get(`${BASE_URL}/orders/${order.id}`);
      check(r, { 'bootstrap get order ok': (res) => res.status === 200 });
    }
  }

  for (const sizeMB of LARGE_PAYLOAD_SIZES.slice(0, 2)) {
    const record = createLargePayload(sizeMB);
    if (record) {
      bootstrapLargePayloads.push({
        id: record.id,
        sizeMB,
      });
    }
  }

  return {
    customers: bootstrapCustomers,
    products: bootstrapProducts,
    orders: bootstrapOrders,
    largePayloads: bootstrapLargePayloads,
  };
}

export default function (data) {
  if (exec.scenario.name === 'large_payload_cycle') {
    runLargePayloadCycle(data);
    return;
  }

  const roll = Math.random();
  if (!data.customers || data.customers.length === 0) {
    return; // setup produced no customers; skip iteration to avoid crash
  }
  const customer = randomItem(data.customers);

  if (roll < 0.1) {
    createCustomer();
  } else if (roll < 0.2) {
    createProduct();
  } else if (roll < 0.45) {
    createOrder(customer.id, data.products);
  } else if (roll < 0.55) {
    if (data.orders && data.orders.length > 0) {
      const bootstrapOrder = randomItem(data.orders);
      const orderResponse = http.get(`${BASE_URL}/orders/${bootstrapOrder.id}`);
      check(orderResponse, {
        'get order status is 200': (r) => r.status === 200,
        'get order returns items': (r) => r.status === 200 && r.json('items').length > 0,
      });
    }
  } else if (roll < 0.75) {
    const isolatedCustomer = createCustomer('Summary Customer');
    if (isolatedCustomer) {
      createOrder(isolatedCustomer.id, data.products);
      const summaryResponse = http.get(`${BASE_URL}/customers/${isolatedCustomer.id}/summary`);
      check(summaryResponse, {
        'customer summary status is 200': (r) => r.status === 200,
      });
    }
  } else {
    // Increased from 0.75–1.0 after order-search was moved to teardown.
    // The isolated customer+order+summary flow is self-contained: each VU
    // creates its own customer, places one order, then fetches that customer's
    // summary. Because the customer is brand-new and unique to this VU, the
    // summary mock is unambiguous — no FIFO collision possible.
    const isolatedCustomer2 = createCustomer('Summary Customer');
    if (isolatedCustomer2) {
      createOrder(isolatedCustomer2.id, data.products);
      const summaryResponse = http.get(`${BASE_URL}/customers/${isolatedCustomer2.id}/summary`);
      check(summaryResponse, {
        'customer summary status is 200': (r) => r.status === 200,
      });
    }
  }

  sleep(randomInt(1, 3) / 10);
}

// teardown runs once after all VU iterations complete, while Keploy is still
// recording. Calling top-products here produces exactly ONE recorded mock and
// ONE test case. A single mock means Keploy's MySQL matcher has no ambiguity:
// it always returns the one recorded response, which matches the one expected
// response → deterministic pass. Contrast with the VU phase where each of the
// many top-products calls returns a different accumulated-state response; the
// matcher always serves the first recorded response (early session state) for
// all subsequent calls, causing every later test case to fail.
// teardown runs once after all VU iterations complete, while Keploy is still
// recording. All stateful search endpoints live here for the same reason
// top-products does: the DB is fully settled, so each search returns a
// deterministic result — one call → one mock → unambiguous replay.
export function teardown(data) {
  // 20-second sleep: the MySQL recorder (recorder/query.go) skips mock capture
  // while memoryguard.IsRecordingPaused() is true.  After the VU phase the
  // Keploy process holds all accumulated mocks in memory; it needs time to
  // flush them and let GC reclaim enough to drop below the 60 % resume
  // threshold before these teardown queries fire.  5 seconds was too short
  // when the second memory-pressure burst overlapped the start of teardown.
  sleep(20);
  const analyticsResponse = http.get(`${BASE_URL}/analytics/top-products?days=30&limit=5`);
  check(analyticsResponse, {
    'top products status is 200': (r) => r.status === 200,
  });

  // Order search: 5 paginated status-only queries (no customer_id).
  // The original per-customer queries embedded a customer ID that was derived
  // from a random email (Date.now() + Math.random()), making the SQL args
  // differ between recording and replay even though Keploy replays the exact
  // recorded URL — the customer IDs in data.customers come from recorded mock
  // responses which ARE stable, but only when the customer-creation mocks were
  // themselves captured (not dropped by syncMock during a pressure window).
  // Using offset-based pagination avoids this dependency entirely: each query
  // has a fixed, deterministic SQL text (LIMIT 10 OFFSET N) that is identical
  // across every recording and replay run.
  for (let i = 0; i < 5; i++) {
    const searchResponse = http.get(
      `${BASE_URL}/orders?status=paid&min_total_cents=1000&limit=10&offset=${i * 10}`
    );
    check(searchResponse, {
      'order search status is 200': (r) => r.status === 200,
    });
  }
}

function runLargePayloadCycle(data) {
  const sizeMB = randomItem(LARGE_PAYLOAD_SIZES);
  const created = createLargePayload(sizeMB);
  if (!created) {
    sleep(0.2);
    return;
  }

  getLargePayload(created.id, sizeMB);
  deleteLargePayload(created.id, sizeMB);

  if (data.largePayloads.length > 0 && Math.random() < 0.35) {
    const existing = randomItem(data.largePayloads);
    getLargePayload(existing.id, existing.sizeMB);
  }

  sleep(randomInt(2, 5) / 10);
}
