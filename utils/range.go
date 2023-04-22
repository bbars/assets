package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Range struct {
	From int64
	To   int64
}

func (r Range) Length() int64 {
	return r.To - r.From
}

func (r Range) HttpHeader(size int64) string {
	if r.To < 0 || r.From < 0 {
		err := r.Normalize(size)
		if err != nil {
			panic(err)
		}
	}
	return fmt.Sprintf(
		"bytes %d-%d/%d",
		r.From,
		r.To-1, // exclusive to inclusive
		size,
	)
}

func (r *Range) Normalize(size int64) (err error) {
	if 0 <= r.From && 0 <= r.To && r.From < r.To && r.To <= size {
		// range already normalized
		return
	}

	if r.From == 0 && r.To < 0 {
		// tail
		r.From = size + r.To
		if r.From < 0 {
			r.From = 0
		}
		r.To = size + 1
	} else if r.From >= 0 && r.To == 0 {
		// to end
		r.To = size + 1
	}

	if r.From < 0 || r.To < 0 {
		err = &RangeError{message: "invalid range"}
		return
	}
	if r.To > size {
		r.To = size
	}
	if r.To < r.From {
		err = errors.Wrap(&RangeError{Range: *r}, "negative length")
		return
	}
	if size <= r.From {
		err = errors.Wrap(&RangeError{Range: *r}, "overflow")
		return
	}
	return
}

func ParseHttpRangeHeader(s string) (r *Range, err error) {
	r = &Range{From: 0, To: 0}
	sLen := len(s)
	if sLen < 8 || s[0:6] != "bytes=" {
		err = &RangeError{message: "range supports bytes only"}
		return
	}

	bld := strings.Builder{}
	var from string
	var to string
	bldTo := false

	for _, c := range s[6:] {
		if '0' <= c && c <= '9' {
			bld.WriteRune(c)
		} else if c == '-' {
			if bldTo {
				err = &RangeError{message: "invalid range syntax"}
				return
			}
			from = bld.String()
			bld.Reset()
			bldTo = true
		} else {
			err = &RangeError{message: "invalid range syntax"}
			return
		}
	}
	to = bld.String()

	if from != "" && to != "" {
		r.From, _ = strconv.ParseInt(from, 10, 64)
		r.To, _ = strconv.ParseInt(to, 10, 64)
		r.To += 1 // inclusive to exclusive
		if r.From > r.To {
			err = errors.Wrap(&RangeError{Range: *r}, "negative length")
			return
		}
	} else if from == "" && to != "" {
		r.To, _ = strconv.ParseInt(to, 10, 64)
		r.To = -r.To
	} else if from != "" {
		r.From, _ = strconv.ParseInt(from, 10, 64)
	} else {
		panic("wat???")
	}
	return
}

type RangeError struct {
	Range   Range
	message string
}

var _ error = &RangeError{}

func (re *RangeError) Error() string {
	if re.message != "" {
		return re.message
	}
	return "invalid range"
}
