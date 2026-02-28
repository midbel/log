package log

import (
	"errors"
	"time"
)

var commonFormat = map[string]string{
	"": "%t(mmm d HH:MM:ss) %u %n[%p]: %m",
}

var (
	ErrPattern = errors.New("invalid pattern")
	ErrSyntax  = errors.New("syntax error")
)

type LogField struct {
	Name  string
	Value string
}

type Entry struct {
	Lino int
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

var defaultParseFormat = map[string]string{}

var defaultPrintFormat = map[string]string{}

func resolvePrintFormat(pattern string) (string, bool) {
	str, ok := commonFormat[pattern]
	if ok {
		return str, ok
	}
	pattern, ok = defaultPrintFormat[pattern]
	return pattern, ok
}

func resolveParseFormat(pattern string) (string, bool) {
	str, ok := commonFormat[pattern]
	if ok {
		return str, ok
	}
	pattern, ok = defaultParseFormat[pattern]
	return pattern, ok
}
