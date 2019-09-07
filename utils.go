package main

import (
	"bytes"
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

type query struct {
	t    uint16
	name string
}

func mustRR(s string) dns.RR {
	rr, err := dns.NewRR(s)
	if err != nil {
		log.Fatal(err)
	}
	return rr
}

func tryAdd(reply *dns.Msg, record string) {
	if rr, err := dns.NewRR(record); err == nil {
		reply.Answer = append(reply.Answer, rr)
	}
}

func forcedV6(ip net.IP) string {
	ip = ip.To16()
	if ip == nil {
		return "::1"
	}
	v := bytes.NewBuffer(nil)
	for i, b := range ip {
		v.WriteString(fmt.Sprintf("%02x", b))
		if i%2 == 1 && i != len(ip)-1 {
			v.WriteByte(':')
		}
	}
	return v.String()
}
