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
// Seed ALL reference data (products, customers, orders, large payloads)
// sequentially so mock time-windows are clean and isolated.

export function setup() {
  client.connect(TARGET_ADDR, { plaintext: true });

  const categories = ['electronics', 'clothing', 'books', 'home', 'sports'];
  const segments   = ['startup', 'enterprise', 'smb', 'consumer'];

  // ── Create products ──
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

  // ── Create customers ──
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

  // ── Create orders (one per customer, using first product) ──
  const orders = [];
  for (let i = 0; i < customers.length; i++) {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateOrder', {
      customer_id: customers[i],
      status:      'pending',
      items:       [{ product_id: products[i % products.length], quantity: 1 }],
    });
    if (res && res.status === grpc.StatusOK && res.message) {
      orders.push(res.message.id);
    }
  }

  // ── Create large payloads (one per VU slot) ──
  const payloads = [];
  for (let i = 0; i < K6_VUS; i++) {
    const res = client.invoke('loadtest.v1.LoadTestService/CreateLargePayload', {
      name:         `setup-payload-${i}-${Date.now()}`,
      content_type: 'text/plain',
      payload:      'x'.repeat(1024),
    });
    if (res && res.status === grpc.StatusOK && res.message) {
      payloads.push(res.message.id);
    }
  }

  // Small sleep to let data settle
  sleep(1);

  client.close();
  return { products, customers, orders, payloads };
}

// ─── default (100% read-only VU phase) ───────────────────────────────────────
// VUs only read settled bootstrap data. No writes during the VU phase
// ensures deterministic, unique query-to-mock mapping during replay.

export default function (data) {
  client.connect(TARGET_ADDR, { plaintext: true });

  const custIdx    = __VU % Math.max(data.customers.length, 1);
  const customerID = data.customers[custIdx] || '';
  const productID  = data.products[__VU % Math.max(data.products.length, 1)] || '';
  const orderID    = data.orders[custIdx % Math.max(data.orders.length, 1)] || '';
  const payloadID  = data.payloads[__VU % Math.max(data.payloads.length, 1)] || '';

  // 1. Get order (read-only — uses settled bootstrap order)
  if (orderID) {
    const res = client.invoke('loadtest.v1.LoadTestService/GetOrder', { order_id: orderID });
    const ok = check(res, { 'get order ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 2. Customer summary (read-only — uses settled bootstrap customer)
  if (customerID) {
    const res = client.invoke('loadtest.v1.LoadTestService/GetCustomerSummary', {
      customer_id: customerID,
    });
    const ok = check(res, { 'customer summary ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 3. Search orders (read-only — queries settled bootstrap data)
  {
    const res = client.invoke('loadtest.v1.LoadTestService/SearchOrders', {
      status: 'pending',
      limit:  10,
      offset: 0,
    });
    const ok = check(res, { 'search orders ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 4. Top products (read-only — queries settled bootstrap data)
  {
    const res = client.invoke('loadtest.v1.LoadTestService/TopProducts', { days: 30, limit: 5 });
    const ok = check(res, { 'top products ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  // 5. Get large payload (read-only — uses settled bootstrap payload)
  if (payloadID) {
    const res = client.invoke('loadtest.v1.LoadTestService/GetLargePayload', { payload_id: payloadID });
    const ok = check(res, { 'get payload ok': (r) => r && r.status === grpc.StatusOK });
    if (!ok) grpcReqFailed.add(1);
  }

  client.close();
  sleep(0.5);
}
