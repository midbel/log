package log

import (
	"bytes"
	"fmt"
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

const (
	defaultTimeFormat = "2006-01-02 15:04:05"
	defaultHostFormat = "hostname"
)

type (
	parsefunc func(*LogField, *scanner) error
	hostfunc  func(*scanner) (string, error)
)

type Specifier struct {
	Name  string
	Char  rune
	parse parsefunc
}

func ParseFormat(pattern string) ([]Specifier, error) {
	if pattern == "" {
		return nil, fmt.Errorf("%w: empty pattern not allowed", ErrSyntax)
	}
	str := scan(pattern)
	pfs, err := parsePattern(scan(pattern))
	if err != nil {
		return nil, err
	}
	if !str.done() {
		return nil, fmt.Errorf("end of pattern expected - token remains")
	}
	return pfs, nil
}

func parsePattern(str *scanner) ([]Specifier, error) {
	var (
		tmp bytes.Buffer
		pfs []Specifier
	)
	for {
		char := str.read()
		if str.done() {
			break
		}
		if char == utf8.RuneError {
			return nil, fmt.Errorf("error reading pattern")
		}
		if k := str.peek(); char != '%' || char == k {
			if char == '%' {
				str.read()
			}
			tmp.WriteRune(char)
			continue
		}
		if tmp.Len() > 0 {
			spec := Specifier{
				Name:  "literal",
				parse: getLiteral(tmp.String()),
			}
			pfs = append(pfs, spec)
			tmp.Reset()
		}
		spec, err := parseSpecifier(str)
		if err != nil {
			return nil, err
		}
		pfs = append(pfs, spec)
	}
	if tmp.Len() > 0 {
		spec := Specifier{
			Name:  "literal",
			parse: getLiteral(tmp.String()),
		}
		pfs = append(pfs, spec)
	}
	return pfs, nil
}

func parseSpecifier(str *scanner) (Specifier, error) {
	var spec Specifier
	spec.Char = str.read()
	switch spec.Char {
	case 't':
		spec.Name = "time"
		format, size, err := parseTimeFormat(str)
		if err != nil {
			return spec, err
		}
		spec.parse = getWhen(format, size)
	case 'b':
		spec.Name = "blank"
		spec.parse = getBlank
	case 'n':
		spec.Name = "process"
		spec.parse = getProcess
	case 'p':
		spec.Name = "pid"
		spec.parse = getPID
	case 'u':
		spec.Name = "user"
		spec.parse = getUser
	case 'g':
		spec.Name = "group"
		spec.parse = getGroup
	case 'h':
		get, err := parseHostFormat(str)
		if err != nil {
			return spec, err
		}
		spec.Name = "host"
		spec.parse = getHost(get)
	case 'l':
		spec.Name = "level"
		spec.parse = getLevel
	case 'm':
		spec.Name = "message"
		spec.parse = getMessage
	case 'w':
		var name string
		if str.peek() == '(' {
			str.read()
			name = str.readUntil(func(r rune) bool { return r != ')' })
		}
		spec.Name = "word"
		spec.parse = getWord(name)
	default:
		return spec, fmt.Errorf("%w: specifier '%%%c' not recognized", ErrSyntax, spec.Char)
	}
	return spec, nil
}

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

func getUser(lf *LogField, str *scanner) error {
	lf.Name = "u"
	lf.Value = str.readLiteral()
	return nil
}

func getGroup(lf *LogField, str *scanner) error {
	lf.Name = "g"
	lf.Value = str.readLiteral()
	return nil
}

func getProcess(lf *LogField, str *scanner) error {
	lf.Name = "n"
	lf.Value = str.readLiteral()
	return nil
}

func getLevel(lf *LogField, str *scanner) error {
	lf.Name = "l"
	lf.Value = str.readLiteral()
	return nil
}

func getPID(lf *LogField, str *scanner) error {
	lf.Name = "p"
	lf.Value = str.readLiteral()
	return nil
}

func getBlank(_ *LogField, str *scanner) error {
	str.readBlank()
	return nil
}

func getMessage(lf *LogField, str *scanner) error {
	lf.Name = "m"
	lf.Value = str.readAll()
	return nil
}

func getWord(name string) parsefunc {
	return func(lf *LogField, str *scanner) error {
		lf.Name = "w"
		lf.Value = str.readLiteral()
		return nil
	}
}

func getWhen(format string, size int) parsefunc {
	return func(lf *LogField, str *scanner) error {
		lf.Name = "t"

		var (
			when time.Time
			err  error
		)
		for i := len(format); i >= size; i-- {
			str.save()
			input := str.readN(i)
			when, err = time.Parse(format, input)
			if err == nil {
				lf.Value = input
				break
			}
			str.restore()
		}
		_ = when
		if err != nil {
			err = ErrPattern
		}
		return err
	}
}

func getHost(get hostfunc) parsefunc {
	fn := func(lf *LogField, str *scanner) error {
		var err error
		lf.Name = "h"
		lf.Value, err = get(str)
		return err
	}
	return fn
}

func getLiteral(in string) parsefunc {
	return func(lf *LogField, str *scanner) error {
		for _, curr := range in {
			char := str.read()
			if curr != char {
				return charactersMismatch(curr, char)
			}
		}
		// lf.Name = "*"
		// lf.Value = in
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
