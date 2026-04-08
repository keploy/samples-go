import http from 'k6/http';
import exec from 'k6/execution';
import { Counter, Trend } from 'k6/metrics';
import { check, sleep } from 'k6';

const isSmokeProfile = __ENV.TEST_PROFILE === 'smoke';
const LARGE_PAYLOAD_SIZE_MBS = (__ENV.LARGE_PAYLOAD_SIZES_MB || '1,2,4')
  .split(',')
  .map((value) => parseInt(value.trim(), 10))
  .filter((value) => Number.isFinite(value) && value > 0);
const LARGE_PAYLOAD_SIZES = LARGE_PAYLOAD_SIZE_MBS.length > 0 ? LARGE_PAYLOAD_SIZE_MBS : [1];

export const options = isSmokeProfile
  ? {
      scenarios: {
        mixed_api_load: {
          executor: 'shared-iterations',
          vus: 1,
          iterations: 8,
          maxDuration: '30s',
        },
        large_payload_cycle: {
          executor: 'shared-iterations',
          vus: 1,
          iterations: 3,
          maxDuration: '45s',
        },
      },
      thresholds: {
        http_req_failed: ['rate<0.05'],
        large_payload_insert_duration: ['p(95)<3000'],
        large_payload_get_duration: ['p(95)<3000'],
        large_payload_delete_duration: ['p(95)<2000'],
      },
    }
  : {
      scenarios: {
        mixed_api_load: {
          executor: 'ramping-arrival-rate',
          startRate: 5,
          timeUnit: '1s',
          preAllocatedVUs: 100,
          maxVUs: 300,
          stages: [
            { target: 15, duration: '30s' },
            { target: 30, duration: '1m' },
            { target: 60, duration: '90s' },
            { target: 20, duration: '30s' },
          ],
        },
        large_payload_cycle: {
          executor: 'ramping-arrival-rate',
          startRate: 1,
          timeUnit: '1s',
          preAllocatedVUs: 16,
          maxVUs: 64,
          stages: [
            { target: 2, duration: '30s' },
            { target: 4, duration: '1m' },
            { target: 2, duration: '30s' },
          ],
        },
      },
      thresholds: {
        http_req_failed: ['rate<0.02'],
        http_req_duration: ['p(95)<2500', 'avg<1200'],
        large_payload_insert_duration: ['p(95)<5000'],
        large_payload_get_duration: ['p(95)<5000'],
        large_payload_delete_duration: ['p(95)<3000'],
      },
    };

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

  for (let i = 0; i < 35; i += 1) {
    const product = createProduct('Bootstrap Product');
    if (product) {
      bootstrapProducts.push(product);
    }
  }

  for (let i = 0; i < 40; i += 1) {
    const customer = randomItem(bootstrapCustomers);
    createOrder(customer.id, bootstrapProducts);
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
  } else if (roll < 0.55) {
    const order = createOrder(customer.id, data.products);
    if (order) {
      const orderResponse = http.get(`${BASE_URL}/orders/${order.id}`);
      check(orderResponse, {
        'get order status is 200': (r) => r.status === 200,
        'get order returns items': (r) => r.status === 200 && r.json('items').length > 0,
      });
    }
  } else if (roll < 0.75) {
    const summaryResponse = http.get(`${BASE_URL}/customers/${customer.id}/summary`);
    check(summaryResponse, {
      'customer summary status is 200': (r) => r.status === 200,
    });
  } else if (roll < 0.9) {
    const minTotal = randomInt(1000, 10000);
    const searchResponse = http.get(
      `${BASE_URL}/orders?status=paid&customer_id=${customer.id}&min_total_cents=${minTotal}&limit=10`
    );
    check(searchResponse, {
      'order search status is 200': (r) => r.status === 200,
    });
  } else {
    const analyticsResponse = http.get(`${BASE_URL}/analytics/top-products?days=30&limit=5`);
    check(analyticsResponse, {
      'top products status is 200': (r) => r.status === 200,
    });
  }

  sleep(randomInt(1, 3) / 10);
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
