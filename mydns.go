package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/miekg/dns"
)

type HandlerWrapper struct {
	DNSHandler
}

func (h *HandlerWrapper) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := &dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	q := &query{msg.Question[0].Qtype, strings.ToLower(msg.Question[0].Name)}

	replies := h.DNSHandler.Handle(q)
	if q.name != "hui.sub.sh.je." {
		_, _ = fmt.Fprintf(os.Stderr, "dns %v: %v %v -> %#v\n", w.RemoteAddr(), dns.TypeToString[q.t], q.name, replies)
	}

	for _, s := range replies {
		tryAdd(msg, s)
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Fatal(err)
	}
}

func main() {
	handlers := DNSHandlers{
		NewRebindRecordHandler(),
		NewMakeRecordHandler(),
		NewIncRecordHandler(),
		NewPredifinedRecordHandler(),
	}

	if len(os.Args) == 2 && os.Args[1] == "--no-rebind" {
		handlers = handlers[1:]
	}

	srv := &dns.Server{
		Addr:    ":53",
		Net:     "udp",
		Handler: &HandlerWrapper{handlers},
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
