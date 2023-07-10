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
		inpat   = flag.String("i", "", "input pattern")
		outpat  = flag.String("o", "", "output pattern")
		filter  = flag.String("f", "", "filter log entry")
		jsonify = flag.Bool("j", false, "jsonify results")
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

	rs, err := log.NewReader(in, *inpat, *filter)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = toLog(rs, *outpat, *jsonify)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func toLog(rs *log.Reader, format string, jsonify bool) error {
	var (
		ws  log.Writer
		err error
	)
	if jsonify {
		ws, _ = log.Json(os.Stdout, true)
	} else {
		ws, err = log.Text(os.Stdout, format)
	}
	if err != nil {
		return err
	}
	for i := 1; ; i++ {
		e, err := rs.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err := ws.Write(e); err != nil {
			return err
		}
	}
	return nil
}
