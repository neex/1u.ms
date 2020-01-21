package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type LogRecord struct {
	Time          time.Time
	RemoteAddr    string
	RequestType   string
	RequestDomain string
	Replies       []string
}

func (lr *LogRecord) String() string {
	return fmt.Sprintf("[%v] %s %s %#v -> %v", lr.Time.Format("2006-01-02 15:04:05.999"),
		lr.RemoteAddr, lr.RequestType, lr.RequestDomain, lr.Replies)
}

type LogViewer struct {
	m         sync.Mutex
	listeners map[chan<- *LogRecord]struct{}
	last      []*LogRecord
}

func NewLogViewer() *LogViewer {
	return &LogViewer{
		listeners: make(map[chan<- *LogRecord]struct{}),
	}
}

func (lv *LogViewer) RegisterHandlers(mux *http.ServeMux) {
	mux.Handle("/log", http.HandlerFunc(lv.handleLog))
	mux.Handle("/last", http.HandlerFunc(lv.handleLast))
}

func (lv *LogViewer) Push(lr *LogRecord) {
	lv.m.Lock()
	defer lv.m.Unlock()

	lv.last = append(lv.last, lr)
	if len(lv.last) > 100 {
		lv.last = lv.last[1:]
	}

	for ch := range lv.listeners {
		select {
		case ch <- lr:
		default:
			close(ch)
			delete(lv.listeners, ch)
		}
	}
}

func (lv *LogViewer) handleLast(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	_, _ = w.Write(lv.getLastRepr())
}

func (lv *LogViewer) handleLog(w http.ResponseWriter, r *http.Request) {
	snapshot, ch := lv.registerListener()
	defer lv.unregisterListener(ch)
	flush := func() {
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	if _, err := w.Write(snapshot); err != nil {
		return
	}
	flush()

	for {
		select {
		case <-r.Context().Done():
			return

		case val, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "%s\n", val); err != nil {
				return
			}
			flush()
		}
	}
}

func (lv *LogViewer) getLastRepr() []byte {
	lv.m.Lock()
	defer lv.m.Unlock()
	return lv.getLastReprNB()
}

func (lv *LogViewer) getLastReprNB() []byte {
	b := bytes.NewBuffer(nil)

	for _, lr := range lv.last {
		_, _ = fmt.Fprintf(b, "%s\n", lr)
	}

	return b.Bytes()
}

func (lv *LogViewer) registerListener() (snapshot []byte, ch chan *LogRecord) {
	lv.m.Lock()
	defer lv.m.Unlock()

	snapshot = lv.getLastReprNB()

	c := make(chan *LogRecord, 100)
	ch = c
	lv.listeners[c] = struct{}{}
	return
}

func (lv *LogViewer) unregisterListener(ch chan *LogRecord) {
	lv.m.Lock()
	defer lv.m.Unlock()

	if _, ok := lv.listeners[ch]; !ok {
		return
	}

	close(ch)
	delete(lv.listeners, ch)
}
