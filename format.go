package log

import (
	"bytes"
	"errors"
	"fmt"
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
	parsefunc func(*Entry, *scanner) error
	hostfunc  func(*scanner) (string, error)
)

func parseFormat(pattern string) (parsefunc, error) {
	if pattern == "" {
		return nil, fmt.Errorf("%w: empty pattern not allowed", ErrSyntax)
	}
	var (
		str = scan(pattern)
		tmp bytes.Buffer
		pfs []parsefunc
	)
	for {
		char := str.read()
		if str.done() {
			break
		}
		if char == utf8.RuneError {
			return nil, fmt.Errorf("error reading pattern: %s", pattern)
		}
		if k := str.peek(); char != '%' || char == k {
			if char == '%' {
				str.read()
			}
			tmp.WriteRune(char)
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

func parseSpecifier(str *scanner) (parsefunc, error) {
	char := str.read()
	switch char {
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
		if str.peek() == '(' {
			str.read()
			name = str.readUntil(func(r rune) bool { return r != ')' })
		}
		return getWord(name), nil
	default:
		return nil, fmt.Errorf("%w: specifier '%%%c' not recognized", ErrSyntax, char)
	}
}

const (
	defaultTimeFormat = "yyyy-mm-dd HH:MM:SS"
	defaultHostFormat = "hostname"
)

func parseHostFormat(str *scanner) (hostfunc, error) {
	if k := str.peek(); k != '(' {
		return getHostname, nil
	}
	str.read()
	var (
		char rune
		hfs  []hostfunc
	)
	for !str.done() {
		if char = str.read(); isEOL(char) {
			return nil, fmt.Errorf("%w: missing ')'", ErrSyntax)
		} else if char == ')' {
			break
		}
		str.unread()

		var (
			pat = str.readAlpha()
			fn  = hostMapping[pat]
		)
		if fn == nil {
			return nil, fmt.Errorf("%s not recognized", pat)
		}
		hfs = append(hfs, fn)
		if str.peek() == ')' {
			continue
		}
		pat = str.readUntil(func(r rune) bool { return !isAlpha(r) })
		if pat != "" {
			hfs = append(hfs, getHostLiteral(pat))
		}
		str.unread()
	}
	return mergeHost(hfs), nil
}

func getHostname(str *scanner) (string, error) {
	return str.readAlpha(), nil
}

func getHostFQDN(str *scanner) (string, error) {
	return str.readAlpha(), nil
}

func getHostIP4(str *scanner) (string, error) {
	return str.readAlpha(), nil
}

func getHostIP6(str *scanner) (string, error) {
	return str.readAlpha(), nil
}

func getHostPort(str *scanner) (string, error) {
	return str.readAlpha(), nil
}

func getHostMask(str *scanner) (string, error) {
	return str.readAlpha(), nil
}

func getHostLiteral(in string) hostfunc {
	return func(str *scanner) (string, error) {
		for _, char := range in {
			c := str.read()
			if char != c {
				return "", charactersMismatch(char, c)
			}
		}
		return in, nil
	}
}

func mergeHost(hfs []hostfunc) hostfunc {
	return func(str *scanner) (string, error) {
		var parts []string
		for _, fn := range hfs {
			s, err := fn(str)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, ""), nil
	}
}

func mergeParse(pfs []parsefunc) parsefunc {
	return func(e *Entry, str *scanner) error {
		for _, pf := range pfs {
			if err := pf(e, str); err != nil {
				return err
			}
		}
		return nil
	}
}

func getUser(e *Entry, str *scanner) error {
	e.User = str.readLiteral()
	return nil
}

func getGroup(e *Entry, str *scanner) error {
	e.Group = str.readLiteral()
	return nil
}

func getProcess(e *Entry, str *scanner) error {
	e.Process = str.readLiteral()
	return nil
}

func getLevel(e *Entry, str *scanner) error {
	e.Level = str.readLiteral()
	return nil
}

func getPID(e *Entry, str *scanner) error {
	var (
		pid = str.readLiteral()
		err error
	)
	e.Pid, err = strconv.Atoi(pid)
	return err
}

func getBlank(_ *Entry, str *scanner) error {
	str.readBlank()
	return nil
}

func getMessage(e *Entry, str *scanner) error {
	e.Message = str.readLiteral()
	return nil
}

func getWord(name string) parsefunc {
	return func(e *Entry, str *scanner) error {
		word := str.readLiteral()
		if name != "" && e.Named != nil {
			e.Named[name] = word
		}
		e.Words = append(e.Words, word)
		return nil
	}
}

func getWhen(format string) parsefunc {
	iter := strings.Count(format, " ")
	return func(e *Entry, str *scanner) error {
		var (
			parts []string
			err   error
		)
		for i := 0; i <= iter; i++ {
			frag := str.readUntil(func(r rune) bool { return !isBlank(r) })
			str.unread()

			parts = append(parts, frag)
			if i < iter {
				str.readBlank()
			}
		}
		e.When, err = time.Parse(format, strings.Join(parts, " "))
		return err
	}
}

func getHost(get hostfunc) parsefunc {
	fn := func(e *Entry, str *scanner) error {
		var err error
		e.Host, err = get(str)
		return err
	}
	return fn
}

func getLiteral(in string) parsefunc {
	return func(_ *Entry, str *scanner) error {
		for _, curr := range in {
			char := str.read()
			if curr != char {
				return charactersMismatch(curr, char)
			}
		}
		return nil
	}
}

func charactersMismatch(want, got rune) error {
	return fmt.Errorf("%w: characters mismatched! want '%c', got '%c'", ErrPattern, want, got)
}

var hostMapping = map[string]hostfunc{
	"hostname": getHostname,
	"fqdn":     getHostFQDN,
	"ip4":      getHostIP4,
	"ip6":      getHostIP6,
	"port":     getHostPort,
	"mask":     getHostMask,
}
