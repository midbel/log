package log

import (
	"fmt"
	"strings"
)

type filterfunc func(Entry) bool

// all(expr,...)
// any(expr,...)
// not(expr)
// eq(field, value)
// ne(field, value)
// lt(field, value)
// le(field, value)
// gt(field, value)
// ge(field, value)
// in(field, value)
// like(field, value)
// between(field, value)
func parseFilter(expr string) (filterfunc, error) {
	if expr == "" {
		return func(_ Entry) bool { return true }, nil
	}
	str := scan(expr)
	return parseFunction(str)
}

func filterAll(fs []filterfunc) filterfunc {
	return func(e Entry) bool {
		for _, f := range fs {
			if !f(e) {
				return false
			}
		}
		return true
	}
}

func filterAny(fs []filterfunc) filterfunc {
	return func(e Entry) bool {
		for _, f := range fs {
			if f(e) {
				return true
			}
		}
		return false
	}
}

func filterNot(f filterfunc) filterfunc {
	return func(e Entry) bool {
		return !f(e)
	}
}

func makeEq(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	return func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && set == value
	}, nil
}

func makeNe(str *scanner) (filterfunc, error) {
	fn, err := makeEq(str)
	if err != nil {
		return nil, err
	}
	return filterNot(fn), nil
}

func makeLike(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	return func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && strings.Contains(set, value)
	}, nil
}

func getField(field string, e Entry) (string, error) {
	var set string
	switch field {
	case "hostname", "host":
		set = e.Host
	case "level":
		set = e.Level
	case "user":
		set = e.User
	case "group":
		set = e.Group
	case "pid":
	case "process":
		set = e.Process
	case "message":
		set = e.Message
	case "time":
	default:
		return "", fmt.Errorf("field %s not recognized", field)
	}
	return set, nil
}

func parseFunction(str *scanner) (filterfunc, error) {
	var (
		fn  filterfunc
		err error
	)
	switch name := str.readAlpha(); name {
	case "all":
		fs, err := parseVariadic(str)
		if err != nil {
			return nil, err
		}
		fn = filterAll(fs)
	case "any":
		fs, err := parseVariadic(str)
		if err != nil {
			return nil, err
		}
		fn = filterAny(fs)
	case "not":
		fn, err := parseUnary(str)
		if err != nil {
			break
		}
		fn = filterNot(fn)
	case "eq":
		fn, err = makeEq(str)
	case "ne":
		fn, err = makeNe(str)
	case "lt":
	case "le":
	case "gt":
	case "ge":
	case "in":
	case "set":
	case "like":
		fn, err = makeLike(str)
	default:
		err = fmt.Errorf("function %s not recognized", name)
	}
	return fn, err
}

func parseFieldValue(str *scanner) (string, string, error) {
	if char := str.read(); char != '(' {
		return "", "", fmt.Errorf("%w: missing '('", ErrSyntax)
	}
	str.readBlank()

	field := str.readText()
	if char := str.read(); char != ',' {
		return "", "", fmt.Errorf("%w: missing ','", ErrSyntax)
	}
	str.readBlank()

	value := str.readAlpha()
	if char := str.read(); char != ')' {
		return "", "", fmt.Errorf("%w: missing ')'", ErrSyntax)
	}
	str.readBlank()

	return field, value, nil
}

func parseVariadic(str *scanner) ([]filterfunc, error) {
	if char := str.read(); char != '(' {
		return nil, fmt.Errorf("%w: missing '('", ErrSyntax)
	}
	str.readBlank()

	var fs []filterfunc
	for !str.done() && str.current() != ')' {
		fn, err := parseFunction(str)
		if err != nil {
			return nil, err
		}
		switch char := str.read(); char {
		case ',':
			str.readBlank()
			if char = str.current(); char == ')' {
				return nil, fmt.Errorf("%w: unexpected ',' before ')'", ErrSyntax)
			}
		case ')':
		default:
			return nil, fmt.Errorf("%w: unexpected character '%c'", ErrSyntax, char)
		}
		fs = append(fs, fn)
	}
	if char := str.current(); char != ')' {
		return nil, fmt.Errorf("%w: missing ')'", ErrSyntax)
	}
	str.readBlank()

	return fs, nil
}

func parseUnary(str *scanner) (filterfunc, error) {
	if char := str.read(); char != '(' {
		return nil, fmt.Errorf("%w: missing '('", ErrSyntax)
	}
	str.readBlank()

	fn, err := parseFunction(str)
	if err != nil {
		return nil, err
	}

	if char := str.read(); char != ')' {
		return nil, fmt.Errorf("%w: missing ')'", ErrSyntax)
	}
	str.readBlank()
	return fn, nil
}
