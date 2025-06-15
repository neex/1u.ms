package main

import (
	"regexp"
)

type DNSHandlerResponse struct {
	RRs            []string
	ReturnServfail bool
}

type DNSHandler interface {
	Handle(q *query) *DNSHandlerResponse
}

type DNSHandlerFunc func(q *query) *DNSHandlerResponse

func (f DNSHandlerFunc) Handle(q *query) *DNSHandlerResponse {
	return f(q)
}

type DNSHandlers []DNSHandler

func (t DNSHandlers) Handle(q *query) *DNSHandlerResponse {
	for i := range t {
		if ans := t[i].Handle(q); ans != nil {
			return ans
		}
	}
	return nil
}

type ParsedRegexpHandler func(q *query, match []string) *DNSHandlerResponse

func NewDNSRegexpHandler(expr string, handler ParsedRegexpHandler) DNSHandler {
	r := regexp.MustCompile(expr)
	return DNSHandlerFunc(func(q *query) *DNSHandlerResponse {
		if m := r.FindStringSubmatch(q.name); m != nil {
			return handler(q, m)
		} else {
			return nil
		}
	})
}
