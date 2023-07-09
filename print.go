package log

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"
)

type printfunc func(Entry, io.StringWriter)

// line specifiers (writing)
// %t: time
// %n: process
// %p: pid
// %u: user
// %g: group
// %h: host
// %l: level
// %m: message
// %#: line
// %[digit]: word
// %%: a percent sign
// c : any character(s)
func parsePrint(pattern string) (printfunc, error) {
	if pattern == "" {
		return nil, fmt.Errorf("%w: empty pattern not allowed", ErrSyntax)
	}
	var (
		str = scan(pattern)
		buf bytes.Buffer
		pfs []printfunc
	)
	for {
		char := str.read()
		if str.done() {
			break
		}
		if k := str.peek(); char == '%' && k != char {
			char = str.read()
			if buf.Len() > 0 {
				pfs = append(pfs, printLiteral(buf.String()))
				buf.Reset()
			}
			switch char {
			case 't':
				pfs = append(pfs, printTime)
			case 'n':
				pfs = append(pfs, printProcess)
			case 'p':
				pfs = append(pfs, printPID)
			case 'u':
				pfs = append(pfs, printUser)
			case 'g':
				pfs = append(pfs, printGroup)
			case 'h':
				pfs = append(pfs, printHost)
			case 'l':
				pfs = append(pfs, printLevel)
			case 'm':
				pfs = append(pfs, printMessage)
			case '#':
				pfs = append(pfs, printLine)
			default:
				return nil, fmt.Errorf("%w(print): unknown specifier %%%c", ErrPattern, char)
			}
		} else {
			if char == '%' && k == char {
				str.read()
			}
			buf.WriteRune(char)
		}
	}
	if buf.Len() > 0 {
		pfs = append(pfs, printLiteral(buf.String()))
	}
	return mergePrint(pfs), nil
}

func mergePrint(pfs []printfunc) printfunc {
	return func(e Entry, w io.StringWriter) {
		for _, p := range pfs {
			p(e, w)
		}
	}
}

func printLiteral(str string) printfunc {
	return func(_ Entry, w io.StringWriter) {
		printString(str, w)
	}
}

func printWord(i int) printfunc {
	return func(e Entry, w io.StringWriter) {
		var str string
		if i >= 0 && i < len(e.Words) {
			str = e.Words[i]
		}
		printString(str, w)
	}
}

func printTime(e Entry, w io.StringWriter) {
	var str string
	if !e.When.IsZero() {
		str = e.When.Format(time.RFC3339)
	}
	printString(str, w)
}

func printProcess(e Entry, w io.StringWriter) {
	printString(e.Process, w)
}

func printPID(e Entry, w io.StringWriter) {
	var str string
	if e.Pid > 0 {
		str = strconv.Itoa(e.Pid)
	}
	printString(str, w)
}

func printUser(e Entry, w io.StringWriter) {
	printString(e.User, w)
}

func printGroup(e Entry, w io.StringWriter) {
	printString(e.Group, w)
}

func printHost(e Entry, w io.StringWriter) {
	printString(e.Host, w)
}

func printLevel(e Entry, w io.StringWriter) {
	printString(e.Level, w)
}

func printMessage(e Entry, w io.StringWriter) {
	printString(e.Message, w)
}

func printLine(e Entry, w io.StringWriter) {
	printString(e.Line, w)
}

func printString(str string, w io.StringWriter) {
	if str == "" {
		return
	}
	w.WriteString(str)
}
