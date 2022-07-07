package main

import (
	"math"
	"strconv"
	"strings"
)

type Range struct {
	min  uint64
	max  uint64
	dict *map[uint64]int
}

func (r *Range) InRange(i uint64) bool {
	if r.dict != nil {
		_, ok := (*(r.dict))[i]
		return ok
	} else {
		return i >= r.min && i <= r.max
	}
}

func ParseRange(s string) (*Range, error) {
	s = strings.TrimSpace(s)
	tryParseEllipseRange := func(s string) (*Range, error) {
		var r = Range{
			min:  0,
			max:  math.MaxUint64,
			dict: nil,
		}
		res := strings.SplitN(s, "..", 2)
		if len(res[0]) > 0 {
			min, err := strconv.Atoi(res[0])
			if err != nil {
				return nil, err
			}
			r.min = uint64(min)
		}

		if len(res[1]) > 0 {
			max, err := strconv.Atoi(res[1])
			if err != nil {
				return nil, err
			}
			r.max = uint64(max)
		}
		return &r, nil
	}

	tryParseCommaRange := func(s string) (*Range, error) {
		var r = Range{
			min:  0,
			max:  math.MaxUint64,
			dict: nil,
		}
		res := strings.Split(s, ",")
		if len(res) > 0 {
			if res[0] == s {
				return &r, nil
			}
			var m = make(map[uint64]int)
			for i, v := range res {
				val, err := strconv.Atoi(v)
				if err != nil {
					return nil, err
				}
				m[uint64(val)] = i
			}
			r.dict = &m
		}
		return &r, nil
	}

	r, err := tryParseEllipseRange(s)
	if err != nil {
		return tryParseCommaRange(s)
	}
	return r, nil
}
