package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type HandlerWrapper struct {
	dh DNSHandler
	lv *LogViewer
}

func (w *HandlerWrapper) ServeDNS(wr dns.ResponseWriter, r *dns.Msg) {
	msg := &dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	q := &query{msg.Question[0].Qtype, strings.ToLower(msg.Question[0].Name)}

	replies, _ := w.dh.Handle(q)
	if q.name != "hui.sub.sh.je." {
		_, _ = fmt.Fprintf(os.Stderr, "dns %v: %v %v -> %#v\n", wr.RemoteAddr(), dns.TypeToString[q.t], q.name, replies)
	}

	for _, s := range replies {
		tryAdd(msg, s)
	}

	if strings.HasSuffix(q.name, "1u.ms.") && q.name != "1u.ms." {
		lr := &LogRecord{
			Time:          time.Now(),
			RemoteAddr:    wr.RemoteAddr().String(),
			RequestType:   dns.TypeToString[q.t],
			RequestDomain: q.name,
			Replies:       replies,
		}

		w.lv.Push(lr)
	}

	if err := wr.WriteMsg(msg); err != nil {
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

	lv := NewLogViewer()
	mux := http.NewServeMux()
	lv.RegisterHandlers(mux)
	go func() {
		err := http.ListenAndServe(":8080", mux)
		if err != nil {
			log.Fatal(err)
		}
	}()

	srv := &dns.Server{
		Addr: ":53",
		Net:  "udp",
		Handler: &HandlerWrapper{
			dh: handlers,
			lv: lv,
		},
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
