package main

import (
	"bytes"
	"io"
	"regexp"
)

type lineGrep struct {
	w   io.Writer
	re  *regexp.Regexp
	buf *bytes.Buffer
}

func newLineGrep(re *regexp.Regexp, w io.Writer) *lineGrep {
	return &lineGrep{
		w:   w,
		re:  re,
		buf: bytes.NewBuffer(nil),
	}
}

func (l *lineGrep) Write(b []byte) (int, error) {
	l.buf.Write(b)
	for bytes.ContainsAny(l.buf.Bytes(), "\n") {
		line, _ := l.buf.ReadBytes('\n')
		if l.re.Match(line) {
			if _, err := l.w.Write(line); err != nil {
				return 0, err
			}
		}
	}
	return len(b), nil
}
