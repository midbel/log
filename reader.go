package log

import (
	"bufio"
	"errors"
	"io"
)

type Reader struct {
	inner *bufio.Scanner
	err   error

	lino  int
	parse []parsefunc
}

func NewReader(rs io.Reader, pattern string) (*Reader, error) {
	if str, ok := resolveParseFormat(pattern); ok {
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
	return &r, nil
}

func (r *Reader) Read() ([]string, error) {
	fs, err := r.readNext()
	if err != nil {
		return nil, err
	}
	rs := make([]string, len(fs))
	for i := range fs {
		rs[i] = fs[i].Value
	}
	return rs, nil
}

func (r *Reader) Next() ([]LogField, error) {
	return r.readNext()
}

func (r *Reader) readNext() ([]LogField, error) {
	if r.err != nil {
		return nil, r.err
	}
	for i := 1; ; i++ {
		if !r.inner.Scan() {
			r.err = r.inner.Err()
			if r.err == nil {
				r.err = io.EOF
			}
			return nil, r.err
		}
		line := r.inner.Text()
		if len(line) == 0 {
			continue
		}
		fs, err := r.readLine(line)
		if err != nil {
			if errors.Is(err, ErrPattern) {
				continue
			}
			r.err = err
			return nil, r.err
		}
		return fs, nil
	}
	return nil, r.err
}

func (r *Reader) readLine(line string) ([]LogField, error) {
	var (
		fs  = make([]LogField, 0, len(r.parse))
		str = scan(line)
	)
	for i := range r.parse {
		var lf LogField
		err := r.parse[i](&lf, str)
		if err != nil {
			return nil, err
		}
		if lf.Name != "" && lf.Value != "" {
			fs = append(fs, lf)
		}
	}
	return fs, nil
}
