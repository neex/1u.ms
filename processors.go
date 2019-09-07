package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	rebindRecord   = `.*rebind-(.*?)rr.*`
	makeRecord     = `.*make-(.*?)rr.*`
	cnameSubdomain = ".sub.sh.je."
)

func NewMakeRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(makeRecord, func(q *query, match []string) []string {
		return convertAddr(match[1], q)
	})
}

func NewRebindRecordHandler() DNSHandler {
	qtimes := make(map[query]time.Time)
	return NewDNSRegexpHandler(rebindRecord, func(q *query, match []string) []string {
		if time.Since(qtimes[*q]) > 5*time.Second {
			qtimes[*q] = time.Now()
			return nil
		}
		return convertAddr(match[1], q)
	})
}

func convertAddr(addr string, q *query) []string {
	if len(addr) > 0 && addr[len(addr)-1] == '-' {
		addr = addr[:len(addr)-1]
	}

	if strings.HasPrefix(addr, "cname-") {
		return []string{makeRR(q.name, "CNAME", makeCNAME(strings.ToLower(addr[6:])))}
	}

	if q.t == dns.TypeA || q.t == dns.TypeAAAA {
		ip := strings.ToLower(addr)

		if strings.HasPrefix(ip, "ip-") {
			ip = ip[3:]
			ip = strings.Replace(ip, "o", ".", -1)
			ip = strings.Replace(ip, "c", ":", -1)
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
					return nil
				}
				return []string{makeRR(q.name, "A", parsed.To4().String())}
			}

			if q.t == dns.TypeAAAA {
				if isV4 {
					return nil
				}
				return []string{makeRR(q.name, "AAAA", forcedV6(parsed))}
			}
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
		return []string{rr}
	}

	return []string{makeRR(q.name, "CNAME", makeCNAME(addr))}
}

func makeCNAME(cname string) string {
	if len(cname) == 0 || cname[len(cname)-1] != '.' {
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
