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
		inpat      = flag.String("i", "", "input pattern")
		outpat     = flag.String("o", "", "output pattern")
		filter     = flag.String("f", "", "filter expression")
		structured = flag.Bool("s", false, "use structured output")
	)
	flag.Parse()

	var (
		in  = os.Stdin
		rs  log.Reader
		err error
	)

	if flag.NArg() > 0 {
		r, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		defer r.Close()
		in = r
	}

	rs, err = log.NewReader(in, *inpat)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *filter != "" {
		rs, err = log.Filter(rs, *filter)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	err = toLog(rs, *structured, *outpat)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func toLog(rs log.Reader, structured bool, format string) error {
	var (
		ws  log.Writer
		err error
	)
	if structured {
		ws, err = log.Structured(os.Stdout, format)
	} else {
		ws, err = log.Text(os.Stdout, format)
	}
	if err != nil {
		return err
	}
	for {
		e, err := rs.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		ws.Write(e.Fields)
	}
	return nil
}
