import http from 'k6/http';
import exec from 'k6/execution';
import { Counter, Trend } from 'k6/metrics';
import { check, sleep } from 'k6';

const isSmokeProfile = __ENV.TEST_PROFILE === 'smoke';
const MIXED_API_START_VUS = parsePositiveIntEnv('MIXED_API_START_VUS', 10);
const MIXED_API_VU_STAGE_TARGETS = parsePositiveIntListEnv(
  'MIXED_API_VU_STAGE_TARGETS',
  [20, 40, 80, 30],
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
    // Extends from 0.75 to 1.0 (was 0.75–0.90 before top-products was moved
    // to teardown). top-products is excluded from the VU phase because its
    // SQL — SELECT … LIMIT 5 — carries no unique parameter that changes
    // across calls. Keploy's MySQL mock matcher returns the first recorded
    // response for any matching SQL pattern; with many VU calls each
    // returning a different accumulated state, every replay gets the same
    // early-session mock. Moving the call to teardown (one invocation,
    // one mock) makes the match unambiguous and the test deterministic.
    const minTotal = randomInt(1000, 10000);
    const searchResponse = http.get(
      `${BASE_URL}/orders?status=paid&customer_id=${customer.id}&min_total_cents=${minTotal}&limit=10`
    );
    check(searchResponse, {
      'order search status is 200': (r) => r.status === 200,
    });
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
export function teardown(_data) {
  const analyticsResponse = http.get(`${BASE_URL}/analytics/top-products?days=30&limit=5`);
  check(analyticsResponse, {
    'top products status is 200': (r) => r.status === 200,
  });
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
