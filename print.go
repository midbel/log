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
// format %[w][[fg:bg]]c
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
		pfs []printinfo
	)
	for {
		char := str.read()
		if str.done() {
			break
		}
		if k := str.peek(); char == '%' && k != char {
			char = str.read()
			if buf.Len() > 0 {
				fn := printLiteral(buf.String())
				pfs = append(pfs, infoFromFunc(fn))
				buf.Reset()
			}
			var info printinfo
			if isDigit(char) {
				str.unread()
				info.Width, _ = strconv.Atoi(str.readNumber())
				char = str.read()
			}
			if char == '[' {
				info.Fore = str.readUntil(func(r rune) bool {
					return r != ',' && r != ']'
				})
				if str.current() == ',' {
					info.Back = str.readUntil(func(r rune) bool { return r != ']' })
				}
				if str.current() != ']' {
					return nil, fmt.Errorf("missing closing ']")
				}
				char = str.read()
			}
			switch char {
			case 't':
				format, err := parseTimeFormat(str)
				if err != nil {
					return nil, err
				}
				info.Func = printTime(format)
			case 'n':
				info.Func = printProcess
			case 'p':
				info.Func = printPID
			case 'u':
				info.Func = printUser
			case 'g':
				info.Func = printGroup
			case 'h':
				info.Func = printHost
			case 'l':
				info.Func = printLevel
			case 'm':
				info.Func = printMessage
			case '#':
				info.Func = printLine
			case 'w':
				info.Func = printName("")
			default:
				return nil, fmt.Errorf("%w(print): unknown specifier %%%c", ErrPattern, char)
			}
			pfs = append(pfs, info)
		} else {
			if char == '%' && k == char {
				str.read()
			}
			buf.WriteRune(char)
		}
	}
	if buf.Len() > 0 {
		fn := printLiteral(buf.String())
		pfs = append(pfs, infoFromFunc(fn))
	}
	return mergePrint(pfs), nil
}

type printinfo struct {
	Width int
	Back  string
	Fore  string
	Func  printfunc
}

func infoFromFunc(fn printfunc) printinfo {
	return printinfo{
		Func: fn,
	}
}

func (p printinfo) Print(e Entry, w io.StringWriter) {
	p.Func(e, w)
}

func mergePrint(pfs []printinfo) printfunc {
	if len(pfs) == 1 {
		return pfs[0].Print
	}
	return func(e Entry, w io.StringWriter) {
		for _, p := range pfs {
			p.Print(e, w)
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

func printName(name string) printfunc {
	return func(e Entry, w io.StringWriter) {
		if name == "" {
			return
		}
		printString(e.Named[name], w)
	}
}

func printTime(format string) printfunc {
	if format == "" {
		format = time.RFC3339
	}
	return func(e Entry, w io.StringWriter) {
		var str string
		if !e.When.IsZero() {
			str = e.When.Format(format)
		}
		printString(str, w)
	}
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
