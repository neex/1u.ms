package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	rebindRecord         = `.*?rebind-(.*?)rr.*`
	rebindForRecord      = `.*?rebindfor([^-]*)-(.*?)rr.*`
	rebindForTimesRecord = `.*?rebindfor([^-]*)after([0-9]*)times-(.*?)rr.*`
	makeRecord           = `.*?make-(.*?)(rr|rebind).*`
	incRecord            = `(.*?inc-)([0-9]+?)(-num.*)`
	multipleRecords      = "-and-"
	setTTLForRecord      = "set-([0-9]+)-ttl"
	fakeRecord           = `.*?fake-(.*?)-rr.*`
	delayRecord          = `.*?delay-(.*?)-(.*?)-only.*`
	servfailRecord       = `.*?make-servfail.*`
)

func NewMakeRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(makeRecord, func(q *query, match []string) *DNSHandlerResponse {
		return convertAddrs(match[1], q)
	})
}

func NewRebindRecordHandler() DNSHandler {
	deadline := make(map[query]time.Time)
	var m sync.Mutex
	return NewDNSRegexpHandler(rebindRecord, func(q *query, match []string) *DNSHandlerResponse {
		m.Lock()
		defer m.Unlock()
		if time.Now().After(deadline[*q]) {
			deadline[*q] = time.Now().Add(5 * time.Second)
			return nil
		}
		return convertAddrs(match[1], q)
	})
}

func NewRebindForRecordHandler() DNSHandler {
	deadline := make(map[query]time.Time)
	var m sync.Mutex
	return NewDNSRegexpHandler(rebindForRecord, func(q *query, match []string) *DNSHandlerResponse {
		m.Lock()
		defer m.Unlock()
		duration, err := time.ParseDuration(match[1])
		if err != nil {
			return nil
		}

		if time.Now().After(deadline[*q]) {
			deadline[*q] = time.Now().Add(duration)
			return nil
		}
		return convertAddrs(match[2], q)
	})
}

func NewRebindForTimesRecordHandler() DNSHandler {
	deadline := make(map[query]time.Time)
	times := make(map[query]int)
	var m sync.Mutex
	return NewDNSRegexpHandler(rebindForTimesRecord, func(q *query, match []string) *DNSHandlerResponse {
		m.Lock()
		defer m.Unlock()
		duration, err := time.ParseDuration(match[1])
		if err != nil {
			return nil
		}
		timesAfter, err := strconv.Atoi(match[2])
		if err != nil {
			return nil
		}

		if time.Now().After(deadline[*q]) {
			deadline[*q] = time.Now().Add(duration)
			times[*q] = 1
			return nil
		}
		if times[*q] < timesAfter {
			times[*q] += 1
			return nil
		}
		return convertAddrs(match[3], q)
	})
}

func NewIncRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(incRecord, func(q *query, match []string) *DNSHandlerResponse {
		if q.t != dns.TypeA {
			return nil
		}
		val, err := strconv.Atoi(match[2])
		if err != nil {
			return nil
		}
		newName := fmt.Sprintf("%s%d%s", match[1], val+1, match[3])
		return &DNSHandlerResponse{
			RRs: []string{makeRR(q.name, "CNAME", newName)},
		}
	})
}

func NewPredefinedRecordHandler(records map[query][]string) DNSHandler {
	return DNSHandlerFunc(func(q *query) *DNSHandlerResponse {
		if rrs, ok := records[*q]; ok {
			return &DNSHandlerResponse{
				RRs: rrs,
			}
		}
		return nil
	})
}

func NewFakeRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(fakeRecord, func(q *query, match []string) *DNSHandlerResponse {
		q.nameForReply = match[1]
		return nil
	})
}

func NewNoHTTPSRecordHandler() DNSHandler {
	return DNSHandlerFunc(func(q *query) *DNSHandlerResponse {
		if q.t == dns.TypeHTTPS {
			return &DNSHandlerResponse{}
		}
		return nil
	})
}

func NewDelayRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(delayRecord, func(q *query, match []string) *DNSHandlerResponse {
		duration, err := time.ParseDuration(match[1])
		if err != nil {
			return nil
		}
		reqType := match[2]
		if strings.ToUpper(reqType) == dns.TypeToString[q.t] {
			time.Sleep(duration)
		}
		return nil
	})
}

func NewServfailRecordHandler() DNSHandler {
	return NewDNSRegexpHandler(servfailRecord, func(q *query, match []string) *DNSHandlerResponse {
		return &DNSHandlerResponse{
			ReturnServfail: true,
		}
	})
}

func convertAddrs(addr string, q *query) *DNSHandlerResponse {
	addrs := strings.Split(addr, multipleRecords)
	found := true
	var rrs []string
	for _, addr := range addrs {
		vals, parsed := convertAddr(addr, q)
		found = found || parsed
		rrs = append(rrs, vals...)
	}
	if found {
		return &DNSHandlerResponse{
			RRs: rrs,
		}
	}
	return nil
}

func convertAddr(addr string, q *query) ([]string, bool) {
	if len(addr) > 0 && addr[len(addr)-1] == '-' {
		addr = addr[:len(addr)-1]
	}

	if strings.HasPrefix(addr, "cname-") {
		return []string{makeRR(q.nameForReply, "CNAME", makeCNAME(strings.ToLower(addr[6:])))}, true
	}

	if q.t == dns.TypeA || q.t == dns.TypeAAAA {
		ip := strings.ToLower(addr)

		if strings.HasPrefix(ip, "ip-") {
			ip = ip[3:]
			ip = strings.Replace(ip, "o", ".", -1)
			ip = strings.Replace(ip, "l", ":", -1)
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
				return []string{makeRR(q.nameForReply, "A", parsed.To4().String())}, true
			}

			if q.t == dns.TypeAAAA {
				if isV4 {
					return nil, true
				}
				return []string{makeRR(q.nameForReply, "AAAA", forcedV6(parsed))}, true
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

	rr := makeRR(q.nameForReply, dns.TypeToString[q.t], addr)
	if _, err := dns.NewRR(rr); err == nil {
		return []string{rr}, true
	}

	return []string{}, true
}

func makeCNAME(cname string) string {
	if len(cname) > 0 && cname[0] == '.' {
		cname = cname[1:]
	}
	return cname
}

var ttlRe = regexp.MustCompile(setTTLForRecord)

func makeRR(domain, qtype, val string) string {
	ttl := 0
	m := ttlRe.FindStringSubmatch(domain)
	if m != nil {
		ttl, _ = strconv.Atoi(m[1])
	}
	return fmt.Sprintf("%s %d IN %s %s", domain, ttl, qtype, val)
}
