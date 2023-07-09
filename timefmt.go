package log

import (
	"bytes"
	"fmt"
	"index/suffixarray"
	"strings"
	"time"
)

func parseTimeFormat(str *scanner) (string, error) {
	if k := str.peek(); k != '(' {
		return defaultTimeFormat, nil
	}
	str.read()
	var (
		tmp  bytes.Buffer
		res  bytes.Buffer
		char rune
	)
	for !str.done() {
		if char = str.read(); isEOL(char) {
			return "", fmt.Errorf("%w: missing ')'", ErrSyntax)
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

var timeCodes = indexArray(timeMapping)

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

func indexArray[T any](in map[string]T) *suffixarray.Index {
	var (
		keys []string
		data string
	)
	for k := range in {
		keys = append(keys, k)
	}
	data = strings.Join(keys, "\x00")
	return suffixarray.New([]byte(data))
}

var timesFormat = []string{
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
}

func parseTime(str string) (time.Time, error) {
	var (
		when time.Time
		err  error
	)
	for _, f := range timesFormat {
		when, err = time.Parse(f, str)
		if err == nil {
			return when, nil
		}
	}
	return when, err
}
