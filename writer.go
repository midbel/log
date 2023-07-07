package log

import (
	"bufio"
	"io"
)

type Writer struct {
	inner *bufio.Writer
	print printfunc
}

func NewWriter(ws io.Writer, pattern string) (*Writer, error) {
	print, err := parsePrint(pattern)
	if err != nil {
		return nil, err
	}
	w := Writer{
		inner: bufio.NewWriter(ws),
		print: print,
	}
	return &w, nil
}

func (w *Writer) Write(e Entry) error {
	w.print(e, w.inner)
	w.inner.WriteRune('\n')
	return w.inner.Flush()
}
