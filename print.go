package log

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"
)

type PrintStyle struct {
	Width int
	Left  bool
	Back  string
	Fore  string
}

type PrintSpecifier struct {
	Name string
	Char rune

	Style PrintStyle

	print printfunc
}

func (p PrintSpecifier) structurable() bool {
	switch p.Char {
	case 't', 'n', 'p', 'l', 'u', 'g', 'h', 'm':
		return true
	default:
		return false
	}
}

func PrintTime(format string) PrintSpecifier {
	return createPrint("time", 't', printTime(format))
}

func PrintPID() PrintSpecifier {
	return createPrint("pid", 'p', printPID)
}

func PrintProcess() PrintSpecifier {
	return createPrint("process", 'n', printProcess)
}

func PrintLevel() PrintSpecifier {
	return createPrint("level", 'l', printLevel)
}

func PrintHost() PrintSpecifier {
	return createPrint("host", 'u', printHost)
}

func PrintUser() PrintSpecifier {
	return createPrint("user", 'u', printUser)
}

func PrintGroup() PrintSpecifier {
	return createPrint("group", 'g', printGroup)
}

func PrintMessage() PrintSpecifier {
	return createPrint("message", 'm', printMessage)
}

func PrintWord(ix int) PrintSpecifier {
	return createPrint("word", 'w', printWord(ix))
}

func createPrint(name string, char rune, fn printfunc) PrintSpecifier {
	return PrintSpecifier{
		Name:  name,
		Char:  char,
		print: fn,
	}
}

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
func ParsePrint(pattern string) ([]PrintSpecifier, error) {
	if pattern == "" {
		return nil, fmt.Errorf("%w: empty pattern not allowed", ErrSyntax)
	}
	var (
		str = scan(pattern)
		buf bytes.Buffer
		pfs []PrintSpecifier
	)
	for {
		char := str.read()
		if str.done() {
			break
		}
		if k := str.peek(); char == '%' && k != char {
			char = str.read()
			if buf.Len() > 0 {
				spec := PrintSpecifier{
					print: printLiteral(buf.String()),
				}
				pfs = append(pfs, spec)
				buf.Reset()
			}
			var style PrintStyle
			if isDigit(char) {
				str.unread()
				style.Width, _ = strconv.Atoi(str.readNumber())
				char = str.read()
			}
			if char == '[' {
				style.Fore = str.readUntil(func(r rune) bool {
					return r != ',' && r != ']'
				})
				if str.current() == ',' {
					style.Back = str.readUntil(func(r rune) bool { return r != ']' })
				}
				if str.current() != ']' {
					return nil, fmt.Errorf("missing closing ']")
				}
				char = str.read()
			}
			var spec PrintSpecifier
			switch char {
			case 't':
				format, _, err := parseTimeFormat(str)
				if err != nil {
					return nil, err
				}
				spec = PrintTime(format)
			case 'n':
				spec = PrintProcess()
			case 'p':
				spec = PrintPID()
			case 'u':
				spec = PrintUser()
			case 'g':
				spec = PrintGroup()
			case 'h':
				spec = PrintHost()
			case 'l':
				spec = PrintLevel()
			case 'm':
				spec = PrintMessage()
			case 'w':
				spec = PrintWord(0)
			default:
				if !isDigit(char) {
					return nil, fmt.Errorf("%w(print): unknown specifier %%%c", ErrPattern, char)
				}
				str.unread()
				n, _ := strconv.Atoi(str.readNumber())
				spec = PrintWord(n)
			}
			spec.Style = style
			pfs = append(pfs, spec)
		} else {
			if char == '%' && k == char {
				str.read()
			}
			buf.WriteRune(char)
		}
	}
	if buf.Len() > 0 {
		spec := PrintSpecifier{
			print: printLiteral(buf.String()),
		}
		pfs = append(pfs, spec)
	}
	return pfs, nil
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
