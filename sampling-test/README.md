# sampling-test

Minimal HTTP server used by the Keploy `--enable-sampling` CI pipeline
(`.github/workflows/sampling-test.yml` in the keploy/keploy repo).

`GET /work` sleeps `HANDLER_DELAY_MS` (default 500ms) before responding so a
burst of concurrent requests overlaps inside the Keploy proxy. With
`--enable-sampling=K`, the proxy only captures K concurrent in-flight
requests; the rest are forwarded transparently without being recorded.

The companion `curl.sh` fires `TOTAL_REQUESTS` (default 20) curls in
parallel — each one its own TCP connection — to exceed the sampling
budget and exercise the bypass path.

After the burst, the workflow asserts:

- every curl saw a 2xx response (clients are never broken by bypass), and
- `K <= captured_test_cases < TOTAL_REQUESTS`.

## Run locally

```bash
go build -o sampling-test
./sampling-test &
TOTAL_REQUESTS=20 bash curl.sh
```
