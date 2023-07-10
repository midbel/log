package log

import (
	"bufio"
	"errors"
	"io"
)

var defaultParseFormat = map[string]string{
	"": "%t(mmm dd HH:MM:ss) %u %n[%p]: %m",
}

type Reader struct {
	inner *bufio.Scanner
	err   error

	lino  int
	keep  filterfunc
	parse parsefunc
}

func NewReader(rs io.Reader, pattern, filter string) (*Reader, error) {
	if str, ok := defaultParseFormat[pattern]; ok {
		pattern = str
	}
	var (
		r   Reader
		err error
	)
	r.inner = bufio.NewScanner(rs)

	if r.parse, err = parseFormat(pattern); err != nil {
		return nil, err
	}
	if r.keep, err = parseFilter(filter); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Reader) ReadAll() ([]Entry, error) {
	var (
		es  []Entry
		e   Entry
		err error
	)
	for {
		e, err = r.Read()
		if err != nil {
			break
		}
		es = append(es, e)
	}
	return es, err
}

func (r *Reader) Read() (Entry, error) {
	r.lino++

	e := Empty()
	if r.err != nil {
		return e, r.err
	}
	for i := 1; ; i++ {
		if !r.inner.Scan() {
			r.err = r.inner.Err()
			if r.err == nil {
				r.err = io.EOF
			}
			return e, r.err
		}
		line := r.inner.Text()
		if len(line) == 0 {
			continue
		}
		err := r.parse(&e, scan(line))
		if err != nil {
			if errors.Is(err, ErrPattern) {
				continue
			}
			r.err = err
			return e, r.err
		}
		if r.keep == nil || r.keep(e) {
			e.Line = line
			e.Lino = r.lino
			break
		}
	}
	return e, r.err
}
