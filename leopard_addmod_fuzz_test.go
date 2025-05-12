package reedsolomon

import (
	"math/rand"
	"testing"
	"time"
)

// Minimal test for the addMod bug
func TestAddModBug(t *testing.T) {
	const modulus = 65535
	a := ffe(65534)
	b := ffe(1)
	got := addMod(a, b)
	want := ffe(0)
	if got != want {
		t.Fatalf("addMod(%d, %d) = %d, want %d", a, b, got, want)
	}
}

// Fuzz addMod near the modulus boundary
func TestAddModFuzzBoundary(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	const modulus = 65535
	for iter := 0; iter < 1000000; iter++ {
		a := ffe(rand.Intn(10) + modulus - 10) // values in [65525, 65534]
		b := ffe(rand.Intn(10) + 1)            // values in [1, 10]
		got := addMod(a, b)
		want := ffe((uint(a) + uint(b)) % modulus)
		if got != want {
			t.Fatalf("addMod(%d, %d) = %d, want %d", a, b, got, want)
		}
	}
}
