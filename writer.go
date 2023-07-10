package log

import (
	"bufio"
	"encoding/json"
	"io"
)

var defaultPrintFormat = map[string]string{}

type Writer interface {
	Write(Entry) error
}

type jsonWriter struct {
	encoder *json.Encoder
}

func Json(ws io.Writer, compact bool) (Writer, error) {
	e := json.NewEncoder(ws)
	if !compact {
		e.SetIndent("", "  ")
	}
	w := jsonWriter{
		encoder: e,
	}
	return &w, nil
}

func (w *jsonWriter) Write(e Entry) error {
	return w.encoder.Encode(e)
}

type textWriter struct {
	inner *bufio.Writer
	print printfunc
}

func Text(ws io.Writer, pattern string) (Writer, error) {
	if str, ok := defaultPrintFormat[pattern]; ok {
		pattern = str
	}
	print, err := parsePrint(pattern)
	if err != nil {
		return nil, err
	}
	w := textWriter{
		inner: bufio.NewWriter(ws),
		print: print,
	}
	return &w, nil
}

func (w *textWriter) Write(e Entry) error {
	w.print(e, w.inner)
	w.inner.WriteRune('\n')
	return w.inner.Flush()
}
