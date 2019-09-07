package main

import (
	"regexp"
)

type DNSHandler interface {
	Handle(q *query) []string
}

type DNSHandlerFunc func(q *query) []string

func (f DNSHandlerFunc) Handle(q *query) []string {
	return f(q)
}

type DNSHandlers []DNSHandler

func (t DNSHandlers) Handle(q *query) []string {
	for i := range t {
		if ans := t[i].Handle(q); ans != nil {
			return ans
		}
	}
	return nil
}

type ParsedRegexpHandler func(q *query, match []string) []string

func NewDNSRegexpHandler(expr string, handler ParsedRegexpHandler) DNSHandler {
	r := regexp.MustCompile(expr)
	return DNSHandlerFunc(func(q *query) []string {
		if m := r.FindStringSubmatch(q.name); m != nil {
			return handler(q, m)
		}
		return nil
	})
}
