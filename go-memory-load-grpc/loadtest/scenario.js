import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const client = new grpc.Client();
client.load(['/proto'], 'loadtest.proto');

const TARGET_ADDR = __ENV.GRPC_ADDR || 'load-test-grpc-api:50051';

const grpcReqFailed = new Counter('grpc_req_failed');

const K6_VUS      = parseInt(__ENV.K6_VUS      || '20',   10);
const K6_DURATION = __ENV.K6_DURATION || '120s';

export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-vus',
      vus:      K6_VUS,
      duration: K6_DURATION,
    },
  },
  thresholds: {
    grpc_req_duration: [
      { threshold: `p(95)<${__ENV.THRESHOLD_HTTP_P95 || 120000}`, abortOnFail: false },
      { threshold: `avg<${__ENV.THRESHOLD_HTTP_AVG || 60000}`, abortOnFail: false },
    ],
    grpc_req_failed: [
      { threshold: `rate<${__ENV.CI_MAX_HTTP_FAILURE_RATE || 0.40}`, abortOnFail: false },
    ],
  },
};

// ─── setup ───────────────────────────────────────────────────────────────────
// Seed reference data (products + customers) that VUs will share.

export function setup() {
  client.connect(TARGET_ADDR, { plaintext: true });

  const categories = ['electronics', 'clothing', 'books', 'home', 'sports'];
  const segments   = ['startup', 'enterprise', 'smb', 'consumer'];

  const products = [];
  for (let i = 0; i < 10; i++) {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateProduct', {
      sku:             `SEED-${i}-${Date.now()}`,
      name:            `Seed Product ${i}`,
      category:        categories[i % categories.length],
      price_cents:     999 + i * 100,
      inventory_count: 100000,
    });
    if (res && res.status === grpc.StatusOK) {
      products.push(res.message.id);
    }
  }

  const customers = [];
  for (let i = 0; i < 5; i++) {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateCustomer', {
      email:     `seed-${i}-${Date.now()}@example.com`,
      full_name: `Seed Customer ${i}`,
      segment:   segments[i % segments.length],
    });
    if (res && res.status === grpc.StatusOK) {
      customers.push(res.message.id);
    }
  }

  client.close();
  return { products, customers };
}

// ─── default ─────────────────────────────────────────────────────────────────

export default function (data) {
  client.connect(TARGET_ADDR, { plaintext: true });

  const customerID = data.customers[__VU % Math.max(data.customers.length, 1)] || '';
  const productID  = data.products[__VU % Math.max(data.products.length, 1)]   || '';

  // 1. Create customer
  {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateCustomer', {
      email:     `vu${__VU}-${Date.now()}@example.com`,
      full_name: `VU User ${__VU}`,
      segment:   'startup',
    });
    const ok = check(res, { 'create customer ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 2. Create product
  {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateProduct', {
      sku:             `VU-${__VU}-${Date.now()}`,
      name:            `VU Product ${__VU}`,
      category:        'electronics',
      price_cents:     1499,
      inventory_count: 99999,
    });
    const ok = check(res, { 'create product ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 3. Create order (requires seeded customer + product)
  let orderID = '';
  if (customerID && productID) {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateOrder', {
      customer_id: customerID,
      status:      'pending',
      items:       [{ product_id: productID, quantity: 1 }],
    });
    const ok = check(res, { 'create order ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) {
      grpcReqFailed.add(1);
    } else if (res.message) {
      orderID = res.message.id;
    }
  }

  // 4. Get order
  if (orderID) {
    const res = client.invoke('loadtest.v1.LoadTestService/GetOrder', { order_id: orderID });
    const ok = check(res, { 'get order ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 5. Customer summary
  if (customerID) {
    const res = client.invoke('loadtest.v1.LoadTestService/GetCustomerSummary', {
      customer_id: customerID,
    });
    const ok = check(res, { 'customer summary ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 6. Search orders
  {
    const res = client.invoke('loadtest.v1.LoadTestService/SearchOrders', {
      status: 'pending',
      limit:  10,
      offset: 0,
    });
    const ok = check(res, { 'search orders ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 7. Top products
  {
    const res = client.invoke('loadtest.v1.LoadTestService/TopProducts', { days: 30, limit: 5 });
    const ok = check(res, { 'top products ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 8. Large payload round-trip
  {
    const payload   = 'x'.repeat(1024);
    const createRes = client.invoke('loadtest.v1.LoadTestService/CreateLargePayload', {
      name:         `payload-${__VU}-${Date.now()}`,
      content_type: 'text/plain',
      payload:      payload,
    });
    const createOk = check(createRes, { 'create payload ok': (r) => r && r.status === grpc.StatusOK });
    if (!createOk) {
      grpcReqFailed.add(1);
    } else {
      const pid = createRes.message.id;

      const getRes = client.invoke('loadtest.v1.LoadTestService/GetLargePayload', { payload_id: pid });
      const getOk  = check(getRes, { 'get payload ok': (r) => r && r.status === grpc.StatusOK });
      if (!getOk) grpcReqFailed.add(1);

      const delRes = client.invoke('loadtest.v1.LoadTestService/DeleteLargePayload', { payload_id: pid });
      const delOk  = check(delRes, { 'delete payload ok': (r) => r && r.status === grpc.StatusOK });
      if (!delOk) grpcReqFailed.add(1);
    }
  }

  client.close();
  sleep(0.5);
}
