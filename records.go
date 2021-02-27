package main

import "github.com/miekg/dns"

var records = map[query][]string{
	{t: dns.TypeSOA, name: "1u.ms."}: {
		"1u.ms. 1 IN SOA hui.sh.je. root.localhost. 12321833 0 0 0 1",
	},

	{t: dns.TypeNS, name: "1u.ms."}: {
		"1u.ms. 1 IN NS hui.sh.je.",
		"1u.ms. 1 IN NS pizda.sh.je.",
	},

	{t: dns.TypeA, name: "1u.ms."}: {
		"1u.ms. 1 IN CNAME hui.sh.je.",
	},

	{t: dns.TypeA, name: "rec.1u.ms."}: {
		"rec.1u.ms. 0 IN CNAME rec.1u.ms.",
	},
}

func init() {
	for _, rrs := range records {
		for _, rr := range rrs {
			_ = mustRR(rr)
		}
	}
}

func NewPredefinedRecordHandler() DNSHandler {
	return DNSHandlerFunc(func(q *query) (rrs []string, had bool) {
		rrs, had = records[*q]
		return
	})
}
