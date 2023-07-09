package log

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
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

func makeAll(fs []filterfunc) filterfunc {
	return func(e Entry) bool {
		for _, f := range fs {
			if !f(e) {
				return false
			}
		}
		return true
	}
}

func makeAny(fs []filterfunc) filterfunc {
	return func(e Entry) bool {
		for _, f := range fs {
			if f(e) {
				return true
			}
		}
		return false
	}
}

func makeNot(f filterfunc) filterfunc {
	return func(e Entry) bool {
		return !f(e)
	}
}

func makeNe(str *scanner) (filterfunc, error) {
	fn, err := makeEq(str)
	if err != nil {
		return nil, err
	}
	return makeNot(fn), nil
}

func makeEq(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && equal(set, value)
	}
	return fn, nil
}

func makeLt(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && lessThan(set, value)
	}
	return fn, nil
}

func makeLe(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && (lessThan(set, value) || equal(set, value))
	}
	return fn, nil
}

func makeGt(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && !lessThan(set, value) && !equal(set, value)
	}
	return fn, nil
}

func makeGe(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		return err == nil && (!lessThan(set, value) || equal(set, value))
	}
	return fn, nil
}

func makeLike(str *scanner) (filterfunc, error) {
	field, value, err := parseFieldValue(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		if err != nil {
			return false
		}
		return strings.Contains(fmt.Sprintf("%s", set), value)
	}
	return fn, nil
}

func makeBetween(str *scanner) (filterfunc, error) {
	field, list, err := parseFieldList(str)
	if err != nil {
		return nil, err
	}
	if len(list) != 2 {
		return nil, fmt.Errorf("too many values given for between")
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		if err != nil {
			return false
		}
		if equal(set, list[0]) || equal(set, list[1]) {
			return true
		}
		return !lessThan(set, list[0]) && lessThan(set, list[1])
	}
	return fn, nil
}

func makeIn(str *scanner) (filterfunc, error) {
	field, list, err := parseFieldList(str)
	if err != nil {
		return nil, err
	}
	fn := func(e Entry) bool {
		set, err := getField(field, e)
		if err != nil {
			return false
		}
		search := fmt.Sprintf("%s", set)
		i := sort.SearchStrings(list, search)
		return i < len(list) && list[i] == search
	}
	return fn, nil
}

func lessThan(val any, value string) bool {
	switch v := val.(type) {
	case int:
		n, _ := strconv.Atoi(value)
		return v < n
	case string:
		return v < value
	case time.Time:
		w, _ := parseTime(value)
		return v.Before(w)
	default:
		return false
	}
}

func equal(val any, value string) bool {
	switch v := val.(type) {
	case int:
		n, _ := strconv.Atoi(value)
		return val == n
	case string:
		return v == value
	case time.Time:
		w, _ := parseTime(value)
		return v.Equal(w)
	default:
		return false
	}
}

func getField(field string, e Entry) (any, error) {
	var set any
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
		set = e.Pid
	case "process":
		set = e.Process
	case "message":
		set = e.Message
	case "time":
		set = e.When
	default:
		return nil, fmt.Errorf("field %s not recognized", field)
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
		fn = makeAll(fs)
	case "any":
		fs, err := parseVariadic(str)
		if err != nil {
			return nil, err
		}
		fn = makeAny(fs)
	case "not":
		fn, err := parseUnary(str)
		if err != nil {
			break
		}
		fn = makeNot(fn)
	case "eq":
		fn, err = makeEq(str)
	case "ne":
		fn, err = makeNe(str)
	case "lt":
		fn, err = makeLt(str)
	case "le":
		fn, err = makeLe(str)
	case "gt":
		fn, err = makeGt(str)
	case "ge":
		fn, err = makeGe(str)
	case "in":
		fn, err = makeIn(str)
	case "like":
		fn, err = makeLike(str)
	case "between":
		fn, err = makeBetween(str)
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

func parseFieldList(str *scanner) (string, []string, error) {
	if char := str.read(); char != '(' {
		return "", nil, fmt.Errorf("%w: missing '('", ErrSyntax)
	}
	str.readBlank()

	field := str.readText()
	if char := str.read(); char != ',' {
		return "", nil, fmt.Errorf("%w: missing ','", ErrSyntax)
	}
	str.readBlank()

	var list []string
	for !str.done() && str.current() != ')' {
		list = append(list, str.readLiteral())
		switch char := str.read(); char {
		case ',':
			str.readBlank()
			if char = str.current(); char == ')' {
				return "", nil, fmt.Errorf("%w: unexpected ',' before ')'", ErrSyntax)
			}
		case ')':
		default:
			return "", nil, fmt.Errorf("%w: unexpected character '%c'", ErrSyntax, char)
		}
	}

	if char := str.read(); char != ')' {
		return "", nil, fmt.Errorf("%w: missing ')'", ErrSyntax)
	}
	str.readBlank()

	sort.Strings(list)
	return field, list, nil
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
