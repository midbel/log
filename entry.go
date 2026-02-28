package log

import (
	"errors"
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
