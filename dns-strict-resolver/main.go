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
	Rcode   int
	Answers []string
}

func parseReply(reply []byte) (parsed, error) {
	if len(reply) < 12 {
		return parsed{}, fmt.Errorf("reply too short")
	}
	flags := binary.BigEndian.Uint16(reply[2:4])
	qd := binary.BigEndian.Uint16(reply[4:6])
	an := binary.BigEndian.Uint16(reply[6:8])
	out := parsed{Rcode: int(flags & 0x000F)}
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
	Domain           string   `json:"domain"`
	Nameserver       string   `json:"nameserver"`
	Rcode            int      `json:"rcode"`
	IPs              []string `json:"ips,omitempty"`
	SourceMismatches int      `json:"source_mismatches"`
	Attempts         int      `json:"attempts"`
	ElapsedMS        int64    `json:"elapsed_ms"`
	Error            string   `json:"error,omitempty"`
}

// resolveStrict sends an A-record query for domain to nsAddr over
// unconnected UDP and accepts the reply only if its source matches
// nsAddr (RFC 5452 §9.1 "birthday attack" mitigation / anti-spoofing).
// Replies whose source does not match are counted in SourceMismatches
// and silently discarded, mirroring what dnspython and glibc's
// unconnected-UDP path do.
func resolveStrict(domain, nsAddr string) result {
	start := time.Now()
	r := result{Domain: domain, Nameserver: nsAddr}

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

	port := "8086"
	fmt.Printf("dns-strict-resolver listening on :%s (default nameserver=%s)\n", port, ns)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
