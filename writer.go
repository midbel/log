package log

import (
	"bufio"
	"io"
)

type Writer interface {
	Write([]LogField) error
}

type textWriter struct {
	inner      *bufio.Writer
	specifiers []PrintSpecifier
}

func Text(ws io.Writer, pattern string) (Writer, error) {
	if str, ok := resolvePrintFormat(pattern); ok {
		pattern = str
	}
	specs, err := ParsePrint(pattern)
	if err != nil {
		return nil, err
	}
	w := textWriter{
		inner:      bufio.NewWriter(ws),
		specifiers: specs,
	}
	return &w, nil
}

func (w *textWriter) Write(fs []LogField) error {
	for _, ps := range w.specifiers {
		ps.print(fs, w.inner)
	}
	w.inner.WriteRune('\n')
	return w.inner.Flush()
}

type structWriter struct {
	inner      *bufio.Writer
	specifiers []PrintSpecifier
}

func Structured(ws io.Writer, pattern string) (Writer, error) {
	if str, ok := resolvePrintFormat(pattern); ok {
		pattern = str
	}
	specs, err := ParsePrint(pattern)
	if err != nil {
		return nil, err
	}
	w := structWriter{
		inner:      bufio.NewWriter(ws),
		specifiers: specs,
	}
	return &w, nil
}

func (w *structWriter) Write(fs []LogField) error {
	for _, ps := range w.specifiers {
		if ps.Char != 'w' && ps.Char != 'b' {
			io.WriteString(w.inner, ps.Name)
			io.WriteString(w.inner, "=")
		}
		ps.print(fs, w.inner)
	}
	w.inner.WriteRune('\n')
	return w.inner.Flush()
}
