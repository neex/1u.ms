package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	rebindRecord    = `.*rebind-(.*?)rr.*`
	makeRecord      = `.*make-(.*?)(rr|rebind).*`
	incRecord       = `(.*inc-)([0-9]+?)(-num.*)`
	cnameSubdomain  = ".sub.sh.je."
	multipleRecords = "-and-"
)

func NewMakeRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(makeRecord, func(q *query, match []string) ([]string, bool) {
		return convertAddrs(match[1], q)
	})
}

func NewRebindRecordHandler() DNSHandler {
	qtimes := make(map[query]time.Time)
	return NewDNSRegexpHandler(rebindRecord, func(q *query, match []string) ([]string, bool) {
		if time.Since(qtimes[*q]) > 5*time.Second {
			qtimes[*q] = time.Now()
			return nil, false
		}
		return convertAddrs(match[1], q)
	})
}

func NewIncRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(incRecord, func(q *query, match []string) ([]string, bool) {
		if q.t != dns.TypeA {
			return nil, false
		}
		val, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, false
		}
		newName := fmt.Sprintf("%s%d%s", match[1], val+1, match[3])
		return []string{makeRR(q.name, "CNAME", newName)}, true
	})
}

func convertAddrs(addr string, q *query) (rrs []string, final bool) {
	addrs := strings.Split(addr, multipleRecords)
	for _, addr := range addrs {
		vals, parsed := convertAddr(addr, q)
		final = final || parsed
		rrs = append(rrs, vals...)
	}
	return
}

func convertAddr(addr string, q *query) ([]string, bool) {
	if len(addr) > 0 && addr[len(addr)-1] == '-' {
		addr = addr[:len(addr)-1]
	}

	if strings.HasPrefix(addr, "cname-") {
		return []string{makeRR(q.name, "CNAME", makeCNAME(strings.ToLower(addr[6:])))}, true
	}

	if q.t == dns.TypeA || q.t == dns.TypeAAAA {
		ip := strings.ToLower(addr)

		if strings.HasPrefix(ip, "ip-") {
			ip = ip[3:]
			ip = strings.Replace(ip, "o", ".", -1)
			ip = strings.Replace(ip, "c", ":", -1)
		} else {
			ipDashDots := strings.Replace(ip, "-", ".", -1)
			ipDashColons := strings.Replace(ip, "-", ":", -1)
			if net.ParseIP(ipDashDots) != nil {
				ip = ipDashDots
			} else if net.ParseIP(ipDashColons) != nil {
				ip = ipDashColons
			}
		}

		forceV6 := false
		if strings.HasPrefix(ip, "v6-") {
			ip = ip[3:]
			forceV6 = true
		}

		parsed := net.ParseIP(ip)
		if parsed != nil {
			isV4 := (parsed.To4() != nil) && !forceV6
			if q.t == dns.TypeA {
				if !isV4 {
					return nil, true
				}
				return []string{makeRR(q.name, "A", parsed.To4().String())}, true
			}

			if q.t == dns.TypeAAAA {
				if isV4 {
					return nil, true
				}
				return []string{makeRR(q.name, "AAAA", forcedV6(parsed))}, true
			}
		}
	} else {
		if strings.HasPrefix(strings.ToLower(addr), "ip-") {
			return []string{}, true
		}
	}

	if strings.HasPrefix(addr, "hex-") {
		cc, err := hex.DecodeString(addr[4:])
		if err == nil {
			addr = string(cc)
		}
	}

	rr := makeRR(q.name, dns.TypeToString[q.t], addr)
	if _, err := dns.NewRR(rr); err == nil {
		return []string{rr}, true
	}

	return []string{makeRR(q.name, "CNAME", makeCNAME(addr))}, true
}

func makeCNAME(cname string) string {
	if !strings.Contains(cname, ".") {
		cname = cname + cnameSubdomain
	}
	if cname[0] == '.' {
		cname = cname[1:]
	}
	return cname
}

func makeRR(domain, qtype, val string) string {
	return fmt.Sprintf("%s 0 IN %s %s", domain, qtype, val)
}
