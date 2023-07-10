package log

import (
	"errors"
	"time"
)

var commonFormat = map[string]string{
	"": "%t(mmm d HH:MM:ss) %u %n[%p]: %m",
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

var (
	ErrPattern = errors.New("invalid pattern")
	ErrSyntax  = errors.New("syntax error")
)

type Entry struct {
	Lino int    `json:"-"`
	Line string `json:"-"`

	Pid     int               `json:"pid,omitempty"`
	Process string            `json:"process,omitempty"`
	User    string            `json:"user,omitempty"`
	Group   string            `json:"group,omitempty"`
	Level   string            `json:"level,omitempty"`
	Message string            `json:"message,omitempty"`
	Words   []string          `json:"-"`
	Named   map[string]string `json:"-"`
	Host    string            `json:"hostname,omitempty"`
	When    time.Time         `json:"time,omitempty"`
}

func Empty() Entry {
	var e Entry
	e.Named = make(map[string]string)
	return e
}
