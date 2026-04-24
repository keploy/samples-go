// Package main is a minimal HTTP server that exercises the RFC 5452
// "strict source address validation" DNS client path used by dnspython,
// raw recvfrom-based clients, and glibc res_send on its unconnected UDP
// path. It is the smallest self-contained reproducer of the failure
// mode fixed by https://github.com/keploy/keploy/pull/4093 /
// https://github.com/keploy/ebpf/pull/97 (tracking issue
// https://github.com/keploy/keploy/issues/4092).
//
// Why we do raw UDP here instead of net.LookupHost:
//   - net.LookupHost on glibc (cgo) uses connected UDP most of the time.
//     Connected-UDP clients are rescued by Keploy's existing
//     cgroup/getpeername4 hook and therefore never exposed the bug.
//   - The production symptom ("Temporary failure in name resolution" /
//     EAI_AGAIN) only surfaces on the unconnected UDP path, where the
//     client validates the reply's source address itself.
//
// With the buggy version of Keploy, /resolve returns with a non-zero
// "source_mismatches" counter and eventually HTTP 502. After the fix,
// the reply's source is rewritten back to the nameserver the client
// queried, the source check passes, and /resolve returns the A records.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var fixtureIPs = map[string]string{
	"alpha.keploy.test": "10.42.0.11",
	"beta.keploy.test":  "10.42.0.12",
	"gamma.keploy.test": "10.42.0.13",
}

func buildQuery(domain string, txid uint16) ([]byte, error) {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, txid)
	binary.Write(&b, binary.BigEndian, uint16(0x0100)) // RD=1
	binary.Write(&b, binary.BigEndian, uint16(1))      // QDCOUNT
	binary.Write(&b, binary.BigEndian, uint16(0))      // ANCOUNT
	binary.Write(&b, binary.BigEndian, uint16(0))      // NSCOUNT
	binary.Write(&b, binary.BigEndian, uint16(0))      // ARCOUNT
	for _, label := range strings.Split(strings.TrimSuffix(domain, "."), ".") {
		if label == "" {
			continue
		}
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %q", label)
		}
		b.WriteByte(byte(len(label)))
		b.WriteString(label)
	}
	b.WriteByte(0)
	binary.Write(&b, binary.BigEndian, uint16(1)) // QTYPE A
	binary.Write(&b, binary.BigEndian, uint16(1)) // QCLASS IN
	return b.Bytes(), nil
}

// skipName walks past a DNS name at offset i, respecting compression
// pointers, and returns the byte index just past the name.
func skipName(buf []byte, i int) int {
	for i < len(buf) {
		l := buf[i]
		if l == 0 {
			return i + 1
		}
		if l&0xc0 == 0xc0 {
			return i + 2
		}
		i += 1 + int(l)
	}
	return i
}

type parsed struct {
	TxID    uint16
	Rcode   int
	Answers []string
}

func parseReply(reply []byte) (parsed, error) {
	if len(reply) < 12 {
		return parsed{}, fmt.Errorf("reply too short")
	}
	txid := binary.BigEndian.Uint16(reply[0:2])
	flags := binary.BigEndian.Uint16(reply[2:4])
	qd := binary.BigEndian.Uint16(reply[4:6])
	an := binary.BigEndian.Uint16(reply[6:8])
	out := parsed{TxID: txid, Rcode: int(flags & 0x000F)}
	off := 12
	for q := uint16(0); q < qd && off < len(reply); q++ {
		off = skipName(reply, off)
		off += 4
	}
	for a := uint16(0); a < an && off+10 <= len(reply); a++ {
		off = skipName(reply, off)
		if off+10 > len(reply) {
			break
		}
		atype := binary.BigEndian.Uint16(reply[off : off+2])
		rdlen := int(binary.BigEndian.Uint16(reply[off+8 : off+10]))
		off += 10
		if atype == 1 && rdlen == 4 && off+rdlen <= len(reply) {
			out.Answers = append(out.Answers,
				net.IPv4(reply[off], reply[off+1], reply[off+2], reply[off+3]).String())
		}
		off += rdlen
	}
	return out, nil
}

type result struct {
	Mode             string   `json:"mode"`
	Domain           string   `json:"domain"`
	Nameserver       string   `json:"nameserver"`
	Rcode            int      `json:"rcode"`
	IPs              []string `json:"ips,omitempty"`
	SourceMismatches int      `json:"source_mismatches"`
	TxidMismatches   int      `json:"txid_mismatches"`
	Attempts         int      `json:"attempts"`
	ElapsedMS        int64    `json:"elapsed_ms"`
	Error            string   `json:"error,omitempty"`
}

type suiteCheck struct {
	Name string `json:"name"`
	// Passed reflects whether this individual check met its assertions.
	Passed bool `json:"passed"`
	// Informational checks are exercised but excluded from the top-level
	// Passed aggregation. Used for BPF behaviours that are documented
	// trade-offs rather than regressions (e.g. the same-socket-to-multiple-
	// upstreams case, which the cookie-keyed orig_dst map in
	// keploy/ebpf#97 is not designed to cover: the second sendmsg4 on a
	// reused socket overwrites the first's stored dst, and recvmsg4 SNATs
	// every reply back to the latest destination).
	Informational bool   `json:"informational,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Result        result `json:"result"`
}

type suiteResult struct {
	Nameserver          string       `json:"nameserver"`
	SecondaryNameserver string       `json:"secondary_nameserver,omitempty"`
	Fixture             bool         `json:"fixture"`
	Passed              bool         `json:"passed"`
	Checks              []suiteCheck `json:"checks"`
	ElapsedMS           int64        `json:"elapsed_ms"`
}

// resolveStrict sends an A-record query for domain to nsAddr over
// unconnected UDP and accepts the reply only if its source matches
// nsAddr (RFC 5452 §9.1 "birthday attack" mitigation / anti-spoofing).
// Replies whose source does not match are counted in SourceMismatches
// and silently discarded, mirroring what dnspython and glibc's
// unconnected-UDP path do.
func resolveStrict(domain, nsAddr string) result {
	start := time.Now()
	r := result{Mode: "unconnected_udp_strict", Domain: domain, Nameserver: nsAddr}

	ns, err := net.ResolveUDPAddr("udp", nsAddr)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		r.Error = err.Error()
		return r
	}
	defer conn.Close()

	query, err := buildQuery(domain, 0x4242)
	if err != nil {
		r.Error = err.Error()
		return r
	}

	deadline := time.Now().Add(3 * time.Second)
	for attempt := 1; attempt <= 3 && time.Now().Before(deadline); attempt++ {
		r.Attempts = attempt
		if _, err := conn.WriteToUDP(query, ns); err != nil {
			r.Error = err.Error()
			r.ElapsedMS = time.Since(start).Milliseconds()
			return r
		}
		for time.Now().Before(deadline) {
			_ = conn.SetReadDeadline(time.Now().Add(800 * time.Millisecond))
			buf := make([]byte, 1500)
			n, src, rerr := conn.ReadFromUDP(buf)
			if rerr != nil {
				break
			}
			if !src.IP.Equal(ns.IP) || src.Port != ns.Port {
				r.SourceMismatches++
				continue
			}
			p, perr := parseReply(buf[:n])
			if perr != nil {
				r.Error = perr.Error()
				r.ElapsedMS = time.Since(start).Milliseconds()
				return r
			}
			if p.TxID != 0x4242 {
				r.TxidMismatches++
				continue
			}
			r.Rcode = p.Rcode
			r.IPs = p.Answers
			r.ElapsedMS = time.Since(start).Milliseconds()
			return r
		}
	}
	r.Error = fmt.Sprintf("no accepted reply from %s after %d attempts", nsAddr, r.Attempts)
	r.ElapsedMS = time.Since(start).Milliseconds()
	return r
}

func resolveConnected(domain, nsAddr string) result {
	start := time.Now()
	r := result{Mode: "connected_udp_control", Domain: domain, Nameserver: nsAddr}

	ns, err := net.ResolveUDPAddr("udp", nsAddr)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	conn, err := net.DialUDP("udp", nil, ns)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	defer conn.Close()

	query, err := buildQuery(domain, 0x4343)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	r.Attempts = 1
	if _, err := conn.Write(query); err != nil {
		r.Error = err.Error()
		r.ElapsedMS = time.Since(start).Milliseconds()
		return r
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		r.Error = err.Error()
		r.ElapsedMS = time.Since(start).Milliseconds()
		return r
	}
	p, err := parseReply(buf[:n])
	if err != nil {
		r.Error = err.Error()
		r.ElapsedMS = time.Since(start).Milliseconds()
		return r
	}
	if p.TxID != 0x4343 {
		r.TxidMismatches++
		r.Error = fmt.Sprintf("reply txid 0x%x did not match query txid 0x4343", p.TxID)
		r.ElapsedMS = time.Since(start).Milliseconds()
		return r
	}
	r.Rcode = p.Rcode
	r.IPs = p.Answers
	r.ElapsedMS = time.Since(start).Milliseconds()
	return r
}

func resolveConcurrentStrict(primaryDomain, primaryNS, secondaryDomain, secondaryNS string) []result {
	start := time.Now()
	results := []result{
		{Mode: "same_socket_multi_upstream_strict", Domain: primaryDomain, Nameserver: primaryNS},
		{Mode: "same_socket_multi_upstream_strict", Domain: secondaryDomain, Nameserver: secondaryNS},
	}

	primaryAddr, err := net.ResolveUDPAddr("udp", primaryNS)
	if err != nil {
		results[0].Error = err.Error()
		return results
	}
	secondaryAddr, err := net.ResolveUDPAddr("udp", secondaryNS)
	if err != nil {
		results[1].Error = err.Error()
		return results
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		results[0].Error = err.Error()
		results[1].Error = err.Error()
		return results
	}
	defer conn.Close()

	queries := []struct {
		txid uint16
		addr *net.UDPAddr
		idx  int
	}{
		{txid: 0x5101, addr: primaryAddr, idx: 0},
		{txid: 0x5102, addr: secondaryAddr, idx: 1},
	}
	for _, q := range queries {
		query, err := buildQuery(results[q.idx].Domain, q.txid)
		if err != nil {
			results[q.idx].Error = err.Error()
			continue
		}
		results[q.idx].Attempts = 1
		if _, err := conn.WriteToUDP(query, q.addr); err != nil {
			results[q.idx].Error = err.Error()
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(results[0].IPs) > 0 && len(results[1].IPs) > 0 {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		buf := make([]byte, 1500)
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		p, err := parseReply(buf[:n])
		if err != nil {
			continue
		}
		idx := -1
		expected := primaryAddr
		if p.TxID == 0x5101 {
			idx = 0
			expected = primaryAddr
		}
		if p.TxID == 0x5102 {
			idx = 1
			expected = secondaryAddr
		}
		if idx == -1 {
			results[0].TxidMismatches++
			results[1].TxidMismatches++
			continue
		}
		if !src.IP.Equal(expected.IP) || src.Port != expected.Port {
			results[idx].SourceMismatches++
			continue
		}
		results[idx].Rcode = p.Rcode
		results[idx].IPs = p.Answers
		results[idx].ElapsedMS = time.Since(start).Milliseconds()
	}

	for i := range results {
		if len(results[i].IPs) == 0 && results[i].Error == "" {
			results[i].Error = fmt.Sprintf("no accepted reply from %s", results[i].Nameserver)
		}
		results[i].ElapsedMS = time.Since(start).Milliseconds()
	}
	return results
}

func validateResult(r result, fixture bool) (bool, string) {
	if r.Error != "" {
		return false, r.Error
	}
	if r.SourceMismatches != 0 {
		return false, fmt.Sprintf("source_mismatches=%d", r.SourceMismatches)
	}
	if r.TxidMismatches != 0 {
		return false, fmt.Sprintf("txid_mismatches=%d", r.TxidMismatches)
	}
	if len(r.IPs) == 0 {
		return false, "no A records returned"
	}
	if fixture {
		want := fixtureIPs[strings.TrimSuffix(r.Domain, ".")]
		if want != "" && !contains(r.IPs, want) {
			return false, fmt.Sprintf("fixture answer mismatch: want %s got %v", want, r.IPs)
		}
	}
	return true, ""
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func runSuite(ns, secondaryNS string, fixture bool) suiteResult {
	start := time.Now()
	out := suiteResult{
		Nameserver:          ns,
		SecondaryNameserver: secondaryNS,
		Fixture:             fixture,
		Passed:              true,
	}

	add := func(name string, r result, informational bool) {
		passed, reason := validateResult(r, fixture)
		if !passed && !informational {
			out.Passed = false
		}
		out.Checks = append(out.Checks, suiteCheck{
			Name:          name,
			Passed:        passed,
			Informational: informational,
			Reason:        reason,
			Result:        r,
		})
	}

	add("strict_unconnected_alpha", resolveStrict("alpha.keploy.test", ns), false)
	add("strict_unconnected_beta", resolveStrict("beta.keploy.test", ns), false)
	add("strict_unconnected_gamma", resolveStrict("gamma.keploy.test", ns), false)
	add("connected_udp_control", resolveConnected("alpha.keploy.test", ns), false)

	if secondaryNS != "" {
		// same_socket_multi_upstream_* is exercised on purpose (Keploy's
		// cookie-keyed orig_dst_by_cookie map lets only one destination
		// be active per socket at a time — a documented limitation of
		// the recvmsg4 SNAT approach, tracked in keploy/ebpf#97 review
		// threads). We run the probe so regressions in either direction
		// surface in the result JSON, but we don't let it gate the
		// suite. A future per-(cookie, dst, txid) tracker in the BPF
		// would let us flip these to hard gates.
		for i, r := range resolveConcurrentStrict("alpha.keploy.test", ns, "beta.keploy.test", secondaryNS) {
			name := "same_socket_multi_upstream_primary"
			if i == 1 {
				name = "same_socket_multi_upstream_secondary"
			}
			add(name, r, true)
		}
	}

	out.ElapsedMS = time.Since(start).Milliseconds()
	return out
}

func defaultNameserver() string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return "8.8.8.8:53"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver ") {
			return net.JoinHostPort(strings.TrimPrefix(line, "nameserver "), "53")
		}
	}
	return "8.8.8.8:53"
}

func main() {
	ns := defaultNameserver()

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/resolve", func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == "" {
			domain = "google.com"
		}
		server := r.URL.Query().Get("nameserver")
		if server == "" {
			server = ns
		}
		res := resolveStrict(domain, server)
		w.Header().Set("Content-Type", "application/json")
		if res.Error != "" {
			w.WriteHeader(http.StatusBadGateway)
		}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		}
	})

	http.HandleFunc("/suite", func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("nameserver")
		if server == "" {
			server = ns
		}
		secondary := r.URL.Query().Get("secondary_nameserver")
		fixture := r.URL.Query().Get("fixture") == "1" || r.URL.Query().Get("fixture") == "true"
		res := runSuite(server, secondary, fixture)
		w.Header().Set("Content-Type", "application/json")
		if !res.Passed {
			w.WriteHeader(http.StatusBadGateway)
		}
		if err := json.NewEncoder(w).Encode(res); err != nil {
			fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		}
	})

	port := "8086"
	fmt.Printf("dns-strict-resolver listening on :%s (default nameserver=%s)\n", port, ns)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
