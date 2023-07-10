package log

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type printfunc func(Entry, io.StringWriter)

// line specifiers (writing)
// format %[w][[fg:bg]]c
// %d
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
			case 'd':
				info.Func = printLino
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
	Left  bool
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
	if code := foregroundAnsiCodes[p.Fore]; code != "" {
		w.WriteString(code)
	}
	if code := backgroundAnsiCodes[p.Back]; code != "" {
		w.WriteString(code)
	}
	var (
		ws  = w
		tmp bytes.Buffer
	)
	if p.Width > 0 {
		ws = &tmp
	}
	p.Func(e, ws)
	if p.Width > 0 {
		diff := p.Width - tmp.Len()
		if diff > 0 {
			tmp.WriteString(strings.Repeat(" ", diff))
		} else if diff < 0 {
			tmp.Truncate(tmp.Len() + diff)
		}
		w.WriteString(tmp.String())
	}
	if p.Fore != "" || p.Back != "" {
		w.WriteString(resetAnsiCode)
	}
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

func printLino(e Entry, w io.StringWriter) {
	w.WriteString(strconv.Itoa(e.Lino))
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

var resetAnsiCode = "\033[0m"

var backgroundAnsiCodes = map[string]string{
	"black":         "\033[40m",
	"red":           "\033[41m",
	"green":         "\033[42m",
	"yellow":        "\033[43m",
	"blue":          "\033[44m",
	"magenta":       "\033[45m",
	"cyan":          "\033[46m",
	"white":         "\033[47m",
	"brightblack":   "\033[100m",
	"brightred":     "\033[101m",
	"brightgreen":   "\033[102m",
	"brightyellow":  "\033[103m",
	"brightblue":    "\033[104m",
	"brightmagenta": "\033[105m",
	"brightcyan":    "\033[106m",
	"brightwhite":   "\033[107m",
}

var foregroundAnsiCodes = map[string]string{
	"black":         "\033[30m",
	"red":           "\033[31m",
	"green":         "\033[32m",
	"yellow":        "\033[33m",
	"blue":          "\033[34m",
	"magenta":       "\033[35m",
	"cyan":          "\033[36m",
	"white":         "\033[37m",
	"brightblack":   "\033[90m",
	"brightred":     "\033[91m",
	"brightgreen":   "\033[92m",
	"brightyellow":  "\033[93m",
	"brightblue":    "\033[94m",
	"brightmagenta": "\033[95m",
	"brightcyan":    "\033[96m",
	"brightwhite":   "\033[97m",
}
