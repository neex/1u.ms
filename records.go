package main

import "github.com/miekg/dns"

var records = map[query][]string{
	{t: dns.TypeSOA, name: "airbnb-photos1s37amazonaws.com."}: {
		"airbnb-photos1s37amazonaws.com. 0 IN SOA hui.sh.je. root.hui.sh.je. 12321838 0 0 0 1",
	},

	{t: dns.TypeNS, name: "airbnb-photos1s37amazonaws.com."}: {
		"airbnb-photos1s37amazonaws.com. 0 IN NS hui.sh.je.",
		"airbnb-photos1s37amazonaws.com. 0 IN NS hui.sub.sh.je.",
	},

	{t: dns.TypeA, name: "airbnb-photos1s37amazonaws.com."}: {
		"airbnb-photos1s37amazonaws.com. 0 IN CNAME zalupa.sub.sh.je.",
	},

	{t: dns.TypeSOA, name: "sub.sh.je."}: {
		"sub.sh.je. 0 IN SOA hui.sh.je. root.hui.sh.je. 12321839 0 0 0 1",
	},

	{t: dns.TypeNS, name: "sub.sh.je."}: {
		"sub.sh.je. 0 IN NS hui.sh.je.",
	},

	{t: dns.TypeA, name: "hui.sub.sh.je."}: {
		"hui.sub.sh.je. 0 IN A 206.81.28.151",
	},

	{t: dns.TypeA, name: "local.sub.sh.je."}: {
		"local.sub.sh.je. 0 IN A 127.0.0.1",
	},

	{t: dns.TypeA, name: "meta.sub.sh.je."}: {
		"meta.sub.sh.je. 0 IN A 169.254.169.254",
	},

	{t: dns.TypeA, name: "this.sub.sh.je."}: {
		"this.sub.sh.je. 0 IN A 206.81.28.151",
	},

	{t: dns.TypeSOA, name: "1u.ms."}: {
		"1u.ms. 0 IN SOA a.make-this.rr.sub.sh.je. root.localhost. 12321833 0 0 0 1",
	},

	{t: dns.TypeNS, name: "1u.ms."}: {
		"1u.ms. 0 IN NS a.make-this.rr.sub.sh.je.",
		"1u.ms. 0 IN NS b.make-this.rr.sub.sh.je.",
	},

	{t: dns.TypeA, name: "1u.ms."}: {
		"1u.ms. 0 IN CNAME this.sub.sh.je.",
	},

	{t: dns.TypeA, name: "rec.sub.sh.je."}: {
		"rec.sub.sh.je. 0 IN CNAME rec.sub.sh.je.",
	},
}

func init() {
	for _, rrs := range records {
		for _, rr := range rrs {
			_ = mustRR(rr)
		}
	}
}

func NewPredifinedRecordHandler() DNSHandler {
	return DNSHandlerFunc(func(q *query) (rrs []string, had bool) {
		rrs, had = records[*q]
		return
	})
}
