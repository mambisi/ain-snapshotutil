package main

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestParseRange(t *testing.T) {
	p, err := ParseRange("2..5")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, p.min, uint64(2))
	assert.Equal(t, p.max, uint64(5))

	p, err = ParseRange("2..")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, p.min, uint64(2))
	if p.max != math.MaxUint64 {
		t.Failed()
	}

	p, err = ParseRange("..1")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, p.min, uint64(0))
	if p.max != 1 {
		t.Failed()
	}

}

func TestParseRangeComma(t *testing.T) {
	p, err := ParseRange("2..5,0")
	if err == nil {
		t.Fatal(p)
	}

	p, err = ParseRange("2,4,10")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, p.min, uint64(0))
	if p.max != math.MaxUint64 {
		t.Failed()
	}
	assert.Equal(t, p.dict, &map[uint64]int{2: 0, 4: 1, 10: 2})

}
