package log

import (
	"bytes"
	"fmt"
	"index/suffixarray"
	"strings"
	"time"
)

func parseTimeFormat(str *scanner) (string, int, error) {
	if k := str.peek(); k != '(' {
		return defaultTimeFormat, 0, nil
	}
	str.read()
	var (
		tmp  bytes.Buffer
		res  bytes.Buffer
		char rune
		size int
	)
	for !str.done() {
		if char = str.read(); isEOL(char) {
			return "", 0, fmt.Errorf("%w: missing ')'", ErrSyntax)
		} else if char == ')' {
			break
		}
		prev := tmp.String()
		if !isLetter(char) {
			match := timeCodes.Lookup(tmp.Bytes(), -1)
			if len(match) > 0 {
				code := timeMapping[tmp.String()]
				res.WriteString(code.Fmt)
				size += code.Len
			}
			res.WriteRune(char)
			tmp.Reset()
			continue
		}
		tmp.WriteRune(char)
		switch match := timeCodes.Lookup(tmp.Bytes(), -1); {
		case len(match) == 1:
			code := timeMapping[tmp.String()]
			size += code.Len
			res.WriteString(code.Fmt)
			tmp.Reset()
		case len(match) == 0 && prev != "":
			code := timeMapping[prev]
			size += code.Len
			res.WriteString(code.Fmt)
			res.WriteRune(char)
			tmp.Reset()
		default:
			// pass
		}
	}
	return res.String(), size, nil
}

var timeCodes = indexArray(timeMapping)

type timeFormatLen struct {
	Len int
	Fmt string
}

func makeFormatLen(str string, size int) timeFormatLen {
	if size == 0 {
		size = len(str)
	}
	return timeFormatLen{
		Len: size,
		Fmt: str,
	}
}

var timeMapping = map[string]timeFormatLen{
	"yy":   makeFormatLen("06", 2),   // year 2 digits
	"yyyy": makeFormatLen("2006", 4), // year 4 digits
	"m":    makeFormatLen("1", 1),    // month no padding
	"mm":   makeFormatLen("01", 2),   // month 2 digits zero padded
	"mmm":  makeFormatLen("Jan", 3),  // abbr month name
	"ccc":  makeFormatLen("Mon", 3),  // abbr day of week name
	"d":    makeFormatLen("_2", 1),    // day of month space padding
	"dd":   makeFormatLen("02", 2),   // day of month 2 digits zero padded
	"ddd":  makeFormatLen("002", 3),  // day of year 3 digits zero padded
	"h":    makeFormatLen("_3", 1),    // hour of day space padding 0-12
	"hh":   makeFormatLen("03", 2),   // hour of day 2 digits zero padding 0-12
	// "H":    "",       // hour of day no padding 0-24
	"HH":  makeFormatLen("15", 2),     // hour of day zero padding 0-24
	"M":   makeFormatLen("4", 1),      // minute of hour no padding
	"MM":  makeFormatLen("04", 2),     // minute of hour 2 digits zero padding
	"s":   makeFormatLen("5", 1),      // second of minute no padding
	"ss":  makeFormatLen("05", 2),     // second of minute 2 digits zero padding
	"S":   makeFormatLen("0", 1),      // milliseconds
	"SSS": makeFormatLen("000", 3),    // milliseconds
	"ZZ":  makeFormatLen("Z07:00", 6), // timezone offset
	"ZZZ": makeFormatLen("Z0700", 5),  // timezone offset
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
