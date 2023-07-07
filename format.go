package log

import (
	"bytes"
	"errors"
	"fmt"
	"index/suffixarray"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// line specifiers (read)
// %t: time (time format, eg, %y-%m-%d)
// %n: process
// %p: pid
// %u: user
// %g: group
// %h: host (host format, eg, ip:port, fqdn)
// %l: level (list of accepted level)
// %m: message
// %w: word
// %b: blank
// %*: discard one or multiple characters
// %%: a percent sign
// c : any character(s)

var (
	ErrPattern = errors.New("invalid pattern")
	ErrSyntax  = errors.New("syntax error")
)

type Entry struct {
	Line string

	Pid     int
	Process string
	User    string
	Group   string
	Level   string
	Message string
	Words   []string
	Named   map[string]string
	Host    string
	When    time.Time
}

func Empty() Entry {
	var e Entry
	e.Named = make(map[string]string)
	return e
}

type (
	parsefunc func(*Entry, io.RuneScanner) error
	hostfunc  func(io.RuneScanner) (string, error)
)

func parseFormat(pattern string) (parsefunc, error) {
	if pattern == "" {
		return nil, fmt.Errorf("%w: empty pattern not allowed", ErrSyntax)
	}
	var (
		str = bytes.NewReader([]byte(pattern))
		tmp bytes.Buffer
		pfs []parsefunc
	)
	for str.Len() > 0 {
		c, _, _ := str.ReadRune()
		if c == utf8.RuneError {
			return nil, fmt.Errorf("error reading pattern: %s", pattern)
		}
		if k := peek(str); c != '%' || c == k {
			if c == '%' {
				str.ReadRune()
			}
			tmp.WriteRune(c)
			continue
		}
		if tmp.Len() > 0 {
			in := tmp.String()
			pfs = append(pfs, getLiteral(in))
			tmp.Reset()
		}
		fn, err := parseSpecifier(str)
		if err != nil {
			return nil, err
		}
		pfs = append(pfs, fn)
	}
	if tmp.Len() > 0 {
		in := tmp.String()
		pfs = append(pfs, getLiteral(in))
	}
	return mergeParse(pfs), nil
}

func parseSpecifier(str io.RuneScanner) (parsefunc, error) {
	r, _, _ := str.ReadRune()
	switch r {
	case 't':
		format, err := parseTimeFormat(str)
		if err != nil {
			return nil, err
		}
		return getWhen(format), nil
	case 'b':
		return getBlank, nil
	case 'n':
		return getProcess, nil
	case 'p':
		return getPID, nil
	case 'u':
		return getUser, nil
	case 'g':
		return getGroup, nil
	case 'h':
		get, err := parseHostFormat(str)
		if err != nil {
			return nil, err
		}
		return getHost(get), nil
	case 'l':
		return getLevel, nil
	case 'm':
		return getMessage, nil
	case 'w':
		var name string
		if peek(str) == '(' {
			str.ReadRune()
			name = readUntil(str, func(r rune) bool { return r == ')' })
		}
		return getWord(name), nil
	default:
	}
	return nil, fmt.Errorf("%w: unsupported specifier %%%c", ErrSyntax, r)
}

const (
	defaultTimeFormat = "yyyy-mm-dd HH:MM:SS"
)

func parseHostFormat(str io.RuneScanner) (hostfunc, error) {
	return nil, nil
}

var timeMapping = map[string]string{
	"yy":   "06",
	"yyyy": "2006",
	"m":    "1",
	"mm":   "01",
	"mmm":  "Jan",
	"ccc":  "Mon",
	"d":    "2",
	"dd":   "02",
	"ddd":  "002",
	"H":    "3",
	"HH":   "15",
	"M":    "4",
	"MM":   "04",
	"ss":   "05",
	"S":    "0",
	"SSS":  "000",
}

var timeCodes = prepareTimeCodes()

func prepareTimeCodes() *suffixarray.Index {
	var (
		keys []string
		data string
	)
	for k := range timeMapping {
		keys = append(keys, k)
	}
	data = strings.Join(keys, "\x00")
	return suffixarray.New([]byte(data))
}

func parseTimeFormat(str io.RuneScanner) (string, error) {
	if k := peek(str); k != '(' {
		return defaultTimeFormat, nil
	}
	str.ReadRune()
	var (
		tmp  bytes.Buffer
		res  bytes.Buffer
		char rune
	)
	for {
		if char, _, _ = str.ReadRune(); isEOL(char) {
			return "", ErrPattern
		} else if char == ')' {
			break
		}
		prev := tmp.String()
		if !isLetter(char) {
			match := timeCodes.Lookup(tmp.Bytes(), -1)
			if len(match) > 0 {
				code := timeMapping[tmp.String()]
				res.WriteString(code)
			}
			res.WriteRune(char)
			tmp.Reset()
			continue
		}
		tmp.WriteRune(char)
		switch match := timeCodes.Lookup(tmp.Bytes(), -1); {
		case len(match) == 1:
			res.WriteString(timeMapping[tmp.String()])
			tmp.Reset()
		case len(match) == 0 && prev != "":
			res.WriteString(timeMapping[prev])
			res.WriteRune(char)
			tmp.Reset()
		default:
			// pass
		}
	}
	return res.String(), nil
}

func mergeParse(pfs []parsefunc) parsefunc {
	return func(e *Entry, r io.RuneScanner) error {
		for _, pf := range pfs {
			if err := pf(e, r); err != nil {
				return err
			}
		}
		return nil
	}
}

func getUser(e *Entry, r io.RuneScanner) error {
	e.User = readLiteral(r)
	return nil
}

func getGroup(e *Entry, r io.RuneScanner) error {
	e.Group = readLiteral(r)
	return nil
}

func getProcess(e *Entry, r io.RuneScanner) error {
	e.Process = readLiteral(r)
	return nil
}

func getLevel(e *Entry, r io.RuneScanner) error {
	e.Level = readLiteral(r)
	return nil
}

func getPID(e *Entry, r io.RuneScanner) error {
	var (
		str = readLiteral(r)
		err error
	)
	e.Pid, err = strconv.Atoi(str)
	return err
}

func getBlank(_ *Entry, r io.RuneScanner) error {
	readBlank(r)
	return nil
}

func getMessage(e *Entry, r io.RuneScanner) error {
	e.Message = readLiteral(r)
	return nil
}

func getWord(name string) parsefunc {
	return func(e *Entry, r io.RuneScanner) error {
		str := readLiteral(r)
		if name != "" && e.Named != nil {
			e.Named[name] = str
		}
		e.Words = append(e.Words, str)
		return nil
	}
}

func getWhen(format string) parsefunc {
	iter := strings.Count(format, " ")
	return func(e *Entry, r io.RuneScanner) error {
		var (
			parts []string
			err   error
		)
		for i := 0; i <= iter; i++ {
			str := readUntil(r, func(r rune) bool { return !isBlank(r) })
			r.UnreadRune()

			parts = append(parts, str)
			if i < iter {
				readBlank(r)
			}
		}
		e.When, err = time.Parse(format, strings.Join(parts, " "))
		return err
	}
}

func getHost(get hostfunc) parsefunc {
	fn := func(e *Entry, r io.RuneScanner) error {
		var err error
		e.Host, err = get(r)
		return err
	}
	return fn
}

func getLiteral(str string) parsefunc {
	return func(_ *Entry, r io.RuneScanner) error {
		for _, curr := range str {
			char, _, _ := r.ReadRune()
			if curr != char {
				return charactersMismatch(curr, char)
			}
		}
		return nil
	}
}

func readLiteral(r io.RuneScanner) string {
	c := peek(r)
	if isQuote(c) {
		return readQuote(r, c)
	}
	return readAlpha(r)
}

func readQuote(r io.RuneScanner, quote rune) string {
	r.ReadRune()
	return readUntil(r, func(c rune) bool { return c == quote })
}

func readAlpha(r io.RuneScanner) string {
	defer r.UnreadRune()
	return readUntil(r, isAlpha)
}

func readBlank(r io.RuneScanner) {
	defer r.UnreadRune()
	readUntil(r, isBlank)
}

func readAll(r io.RuneScanner) string {
	return readUntil(r, isEOL)
}

func readUntil(r io.RuneScanner, accept func(rune) bool) string {
	var buf bytes.Buffer
	for {
		c, _, _ := r.ReadRune()
		if !accept(c) {
			break
		}
		buf.WriteRune(c)
	}
	return buf.String()
}

func peek(r io.RuneScanner) rune {
	defer r.UnreadRune()
	c, _, _ := r.ReadRune()
	return c
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlpha(r rune) bool {
	return isDigit(r) || isLetter(r) || r == '-' || r == '_'
}

func isBlank(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func isEOL(r rune) bool {
	return r == 0 || r == utf8.RuneError
}

func isQuote(r rune) bool {
	return r == '\'' || r == '"'
}

func isEscape(r rune) bool {
	return r == '\\' || r == '@' || r == '*' || r == '(' || r == ')' || r == '|'
}

func charactersMismatch(want, got rune) error {
	return fmt.Errorf("%w: characters mismatched! want %c, got %c", ErrPattern, want, got)
}
