package log

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

type printfunc func([]LogField, io.StringWriter)

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
				format, _, err := parseTimeFormat(str)
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
			case 'w':
				info.Func = printName("")
			default:
				if !isDigit(char) {
					return nil, fmt.Errorf("%w(print): unknown specifier %%%c", ErrPattern, char)
				}
				str.unread()
				n, _ := strconv.Atoi(str.readNumber())
				info.Func = printWord(n)
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

func (p printinfo) Print(fs []LogField, w io.StringWriter) {
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
	p.Func(fs, ws)
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
	return func(fs []LogField, w io.StringWriter) {
		for _, p := range pfs {
			p.Print(fs, w)
		}
	}
}

func printLiteral(str string) printfunc {
	return func(_ []LogField, w io.StringWriter) {
		w.WriteString(str)
	}
}

func printWord(i int) printfunc {
	return func(fs []LogField, w io.StringWriter) {

	}
}

func printName(name string) printfunc {
	return func(fs []LogField, w io.StringWriter) {

	}
}

func printTime(format string) printfunc {
	if format == "" {
		format = time.RFC3339
	}
	return func(fs []LogField, w io.StringWriter) {
		printField(fs, "t", w)
	}
}

func printProcess(fs []LogField, w io.StringWriter) {
	printField(fs, "n", w)
}

func printPID(fs []LogField, w io.StringWriter) {
	printField(fs, "p", w)
}

func printUser(fs []LogField, w io.StringWriter) {
	printField(fs, "u", w)
}

func printGroup(fs []LogField, w io.StringWriter) {
	printField(fs, "g", w)
}

func printHost(fs []LogField, w io.StringWriter) {
	printField(fs, "h", w)
}

func printLevel(fs []LogField, w io.StringWriter) {
	printField(fs, "l", w)
}

func printMessage(fs []LogField, w io.StringWriter) {
	printField(fs, "m", w)
}

func printField(fs []LogField, field string, w io.StringWriter) {
	ix := slices.IndexFunc(fs, func(lf LogField) bool {
		return lf.Name == field && lf.Value != ""
	})
	if ix >= 0 {
		w.WriteString(fs[ix].Value)
	}
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
