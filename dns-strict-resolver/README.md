# dns-strict-resolver

Minimal Go HTTP server that exercises the **unconnected-UDP + RFC 5452
strict-source-validation** DNS client path. Used by Keploy's e2e CI as a
regression guard for the `cgroup/recvmsg{4,6}` SNAT fix.

- Tracking issue: https://github.com/keploy/keploy/issues/4092
- Keploy fix: https://github.com/keploy/keploy/pull/4093
- eBPF fix:   https://github.com/keploy/ebpf/pull/97

## Why a raw UDP client?

`net.LookupHost` on glibc (cgo) uses connected UDP most of the time, and
connected-UDP clients are rescued by Keploy's existing
`cgroup/getpeername4` hook — so they never exposed this bug. The
production failure mode (`java.net.UnknownHostException: Temporary
failure in name resolution` / `EAI_AGAIN`) only surfaces on the
unconnected-UDP path, where the client is responsible for validating the
reply's source address itself.

This sample sends a DNS A query over an **unconnected** UDP socket,
reads replies with `ReadFromUDP`, and **discards any reply whose source
does not match the nameserver it queried** — the same check that
`dnspython`, raw `recvfrom`-based clients, and glibc's `res_send`
unconnected path perform.

## Running

```bash
go run . &
curl -sS "http://localhost:8086/resolve?domain=google.com"
```

Expected shape (post-fix):
```json
{
  "domain": "google.com",
  "nameserver": "127.0.0.11:53",
  "rcode": 0,
  "ips": ["142.250.x.x", "..."],
  "source_mismatches": 0,
  "attempts": 1,
  "elapsed_ms": 4
}
```

Under the **buggy** (pre-fix) Keploy, replies arrive from
`<agent_ip>:<keploy_dns_port>` instead of the configured nameserver, the
source check rejects them, and `/resolve` eventually returns HTTP 502
with a non-zero `source_mismatches` counter and no answers.

## Under Keploy

```bash
sudo -E env PATH=$PATH keploy record -c "./dns-strict-resolver"
# hit /resolve endpoints, then stop keploy

sudo -E env PATH=$PATH keploy test -c "./dns-strict-resolver" --delay 10
```

Both record and test must complete with `source_mismatches: 0` and a
non-empty `ips` list for the sample to pass.

## Endpoints

| Path                                          | Description                                            |
| --------------------------------------------- | ------------------------------------------------------ |
| `GET /health`                                 | Liveness probe used by the CI script.                  |
| `GET /resolve?domain=<d>&nameserver=<ip:53>`  | Strict A-record lookup. `domain` defaults to `google.com`; `nameserver` defaults to the first entry in `/etc/resolv.conf`. |
