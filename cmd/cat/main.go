package main

import (
	"errors"
	"encoding/json"
	"flag"
	"io"
	"fmt"
	"os"

	"github.com/midbel/log"
)

var (
	input  = "[%t] [%h(%4:%p)]%b%u:%g:%n [%p:%l]:%b%m"
	output = "%t %n[%p]: %m"
)

func main() {
	var (
		in     = flag.String("i", "", "input pattern")
		out    = flag.String("o", "", "output pattern")
		filter = flag.String("f", "", "filter log entry")
		jsonify = flag.Bool("j", false, "jsonify results")
	)
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	rs, err := log.NewReader(r, *in, *filter)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *jsonify {
		err = toJson(rs)
	} else {
		err = toLog(rs, *out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func toJson(rs *log.Reader) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("  ", "  ")
	for {
		e, err := rs.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}

func toLog(rs *log.Reader, format string) error {
	ws, err := log.NewWriter(os.Stdout, format)
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