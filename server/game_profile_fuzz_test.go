package main

import (
	"math"
	"testing"
)

// FuzzGameSafeArithmetic checks that resource accounting never wraps int64.
func FuzzGameSafeArithmetic(f *testing.F) {
	for _, pair := range [][2]int64{{0, 0}, {1, 1}, {math.MaxInt64, 1}, {math.MaxInt64, 2}, {-1, 2}, {math.MinInt64, -1}, {math.MinInt64, 1}} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, a, b int64) {
		sum, err := safeGameAdd(a, b)
		if err == nil && (b > 0 && sum < a || b < 0 && sum > a) {
			t.Fatalf("wrapped sum: %d+%d=%d", a, b, sum)
		}
		product, mulErr := safeGameMul(a, b)
		if mulErr == nil && a != 0 && (a == math.MinInt64 && b == -1 || product/a != b) {
			t.Fatalf("wrapped product: %d*%d=%d", a, b, product)
		}
	})
}
