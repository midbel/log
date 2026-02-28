package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/log"
)

func main() {
	var (
		inpat  = flag.String("i", "", "input pattern")
		outpat = flag.String("o", "", "output pattern")
	)
	flag.Parse()

	var in = os.Stdin

	if flag.NArg() > 0 {
		r, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer r.Close()
		in = r
	}

	rs, err := log.NewReader(in, *inpat)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = toLog(rs, *outpat)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func toLog(rs *log.Reader, format string) error {
	ws, err := log.Text(os.Stdout, format)
	if err != nil {
		return err
	}
	for {
		fs, err := rs.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		ws.Write(fs)
	}
	return nil
}
