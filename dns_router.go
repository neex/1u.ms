package main

import (
	"regexp"
)

type DNSHandler interface {
	Handle(q *query) (rrs []string, final bool)
}

type DNSHandlerFunc func(q *query) ([]string, bool)

func (f DNSHandlerFunc) Handle(q *query) ([]string, bool) {
	return f(q)
}

type DNSHandlers []DNSHandler

func (t DNSHandlers) Handle(q *query) ([]string, bool) {
	for i := range t {
		if ans, final := t[i].Handle(q); final {
			return ans, final
		}
	}
	return nil, false
}

type ParsedRegexpHandler func(q *query, match []string) ([]string, bool)

func NewDNSRegexpHandler(expr string, handler ParsedRegexpHandler) DNSHandler {
	r := regexp.MustCompile(expr)
	return DNSHandlerFunc(func(q *query) ([]string, bool) {
		if m := r.FindStringSubmatch(q.name); m != nil {
			return handler(q, m)
		}
		return nil, false
	})
}
