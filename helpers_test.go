package fitparser_test

import (
	"bytes"
	"testing"
	"time"

	fitparser "github.com/TeraThreshold/fit-parser"
	"github.com/TeraThreshold/fit-parser/internal/fitgen"
)

// t0 is the synthetic start time used across the tests.
var t0 = time.Date(2024, 5, 4, 7, 30, 0, 0, time.UTC)

// decode encodes b and decodes the result, failing the test on any error.
func decode(t *testing.T, b *fitgen.Builder) *fitparser.File {
	t.Helper()
	data, err := b.Bytes()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	f, err := fitparser.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if f == nil {
		t.Fatal("Decode returned nil file without error")
	}
	return f
}

// one returns the only element of s, failing the test when len(s) != 1.
func one[T any](t *testing.T, s []T) T {
	t.Helper()
	if len(s) != 1 {
		t.Fatalf("got %d elements, want 1: %+v", len(s), s)
	}
	return s[0]
}
