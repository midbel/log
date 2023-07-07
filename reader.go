package log

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type Reader struct {
	inner *bufio.Scanner
	err   error

	keep  filterfunc
	parse parsefunc
}

func NewReader(rs io.Reader, pattern, filter string) (*Reader, error) {
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
	var e Entry
	if r.err != nil {
		return e, r.err
	}
	for {
		if !r.inner.Scan() {
			r.err = r.inner.Err()
			if r.err == nil {
				r.err = io.EOF
			}
			return e, r.err
		}
		line := r.inner.Bytes()
		if len(line) == 0 {
			continue
		}
		err := r.parse(&e, bytes.NewReader(line))
		if err != nil {
			if errors.Is(err, ErrPattern) {
				continue
			}
			r.err = err
			return e, r.err
		}
		if r.keep == nil || r.keep(e) {
			e.Line = r.inner.Text()
			break
		}
	}
	return e, r.err
}
