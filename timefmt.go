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
	"yy":   "06",   // year 2 digits
	"yyyy": "2006", // year 4 digits
	"m":    "1",    // month no padding
	"mm":   "01",   // month 2 digits zero padded
	"mmm":  "Jan",  // abbr month name
	"ccc":  "Mon",  // abbr day of week name
	"d":    "2",    // day of month no padding
	"dd":   "02",   // day of month 2 digits zero padded
	"ddd":  "002",  // day of year 3 digits zero padded
	"h":    "3",    // hour of day zero padding 0-12
	"hh":   "03",   // hour of day 2 digits zero padding 0-12
	// "H":    "",     // hour of day no padding 0-24
	"HH":   "15",   // hour of day zero padding 0-24
	"M":    "4",    // minute of hour no padding
	"MM":   "04",   // minute of hour 2 digits zero padding
	"s":    "5",    // second of minute no padding
	"ss":   "05",   // second of minute 2 digits zero padding
	"S":    "0",    // milliseconds
	"SSS":  "000",  // milliseconds
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
