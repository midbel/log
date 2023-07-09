package log

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

type cursor struct {
	curr int
	next int
}

type scanner struct {
	input string
	cursor

	old cursor
}

func scan(str string) *scanner {
	return &scanner{
		input: str,
	}
}

func (s *scanner) rest() string {
	return s.input[s.next:]
}

func (s *scanner) reset() {
	s.cursor = cursor{}
	s.old = s.cursor
}

func (s *scanner) readN(n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		char := s.read()
		buf.WriteRune(char)
	}
	return buf.String()
}

func (s *scanner) readUntil(accept func(rune) bool) string {
	var buf bytes.Buffer
	for !s.done() {
		char := s.read()
		if !accept(char) {
			break
		}
		buf.WriteRune(char)
	}
	return buf.String()
}

func (s *scanner) readLiteral() string {
	char := s.peek()
	if isQuote(char) {
		return s.readQuote()
	}
	return s.readAlpha()
}

func (s *scanner) readText() string {
	defer s.unread()
	return s.readUntil(isLetter)
}

func (s *scanner) readAlpha() string {
	defer s.unread()
	return s.readUntil(isAlpha)
}

func (s *scanner) readQuote() string {
	quote := s.current()
	s.read()
	return s.readUntil(func(c rune) bool { return c == quote })
}

func (s *scanner) readBlank() {
	defer s.unread()
	s.readUntil(isBlank)
}

func (s *scanner) readAll() string {
	return s.readUntil(isEOL)
}

func (s *scanner) read() rune {
	s.old = s.cursor

	char, size := utf8.DecodeRuneInString(s.input[s.next:])
	s.curr = s.next
	s.next += size
	return char
}

func (s *scanner) unread() error {
	if s.cursor == s.old {
		return fmt.Errorf("unread can only be called once after call to read")
	}
	s.cursor = s.old
	return nil
}

func (s *scanner) current() rune {
	char, _ := utf8.DecodeRuneInString(s.input[s.curr:])
	return char
}

func (s *scanner) peek() rune {
	if s.next >= len(s.input) {
		return utf8.RuneError
	}
	char, _ := utf8.DecodeRuneInString(s.input[s.next:])
	return char
}

func (s *scanner) done() bool {
	return s.curr >= len(s.input)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlpha(r rune) bool {
	return isDigit(r) || isLetter(r) || r == '-' || r == '_' || r == '.'
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
