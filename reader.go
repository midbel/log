package log

import (
	"bufio"
	"errors"
	"io"
)

type Reader interface {
	Next() (LogEntry, error)
}

type LogFilter struct {
	inner  Reader
	filter filterfunc
}

func Filter(rs Reader, filter string) (Reader, error) {
	var (
		r   LogFilter
		err error
	)
	r.inner = rs
	if r.filter, err = ParseFilter(filter); err != nil {
		return nil, err
	}
	return &r, err
}
func (r *LogFilter) Next() (LogEntry, error) {
	for {
		e, err := r.inner.Next()
		if err != nil {
			return e, err
		}
		ok := r.filter(e)
		if ok {
			return e, nil
		}
	}
}

type LogReader struct {
	inner *bufio.Scanner
	err   error

	lino       int
	specifiers []Specifier
}

func NewReader(rs io.Reader, pattern string) (*LogReader, error) {
	if str, ok := resolveParseFormat(pattern); ok {
		pattern = str
	}
	var (
		r   LogReader
		err error
	)
	r.inner = bufio.NewScanner(rs)

	if r.specifiers, err = ParseFormat(pattern); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *LogReader) Attach(spec Specifier) {
	r.specifiers = append(r.specifiers, spec)
}

func (r *LogReader) Read() ([]string, error) {
	es, err := r.readNext()
	if err != nil {
		return nil, err
	}
	rs := make([]string, len(es.Fields))
	for i := range es.Fields {
		rs[i] = es.Fields[i].Value
	}
	return rs, nil
}

func (r *LogReader) Next() (LogEntry, error) {
	e, err := r.readNext()
	return e, err
}

func (r *LogReader) readNext() (LogEntry, error) {
	var es LogEntry
	if r.err != nil {
		return es, r.err
	}
	r.lino++
	for i := 1; ; i++ {
		if !r.inner.Scan() {
			r.err = r.inner.Err()
			if r.err == nil {
				r.err = io.EOF
			}
			return es, r.err
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
			return es, r.err
		}
		es = LogEntry{
			Lino:   r.lino,
			Line:   line,
			Fields: fs,
		}
		break
	}
	return es, r.err
}

func (r *LogReader) readLine(line string) ([]LogField, error) {
	var (
		fs  = make([]LogField, 0, len(r.specifiers))
		str = scan(line)
	)
	for i := range r.specifiers {
		var lf LogField
		err := r.specifiers[i].parse(&lf, str)
		if err != nil {
			return nil, err
		}
		if lf.Name != "" && lf.Value != "" {
			fs = append(fs, lf)
		}
	}
	return fs, nil
}
