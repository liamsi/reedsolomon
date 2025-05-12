package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestEncoderReconstructLeo(t *testing.T) {
	testEncoderReconstructLeo(t)
}

func testEncoderReconstructLeo(t *testing.T, o ...Option) {
	// Create some sample data
	var data = make([]byte, 2<<20)
	fillRandom(data)

	// Create 5 data slices of 50000 elements each
	enc, err := New(500, 300, testOptions(o...)...)
	if err != nil {
		t.Fatal(err)
	}
	shards, err := enc.Split(data)
	if err != nil {
		t.Fatal(err)
	}
	err = enc.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err := enc.Verify(shards)
	if !ok || err != nil {
		t.Fatal("not ok:", ok, "err:", err)
	}

	// Delete a shard
	shards[0] = nil

	// Should reconstruct
	err = enc.Reconstruct(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err = enc.Verify(shards)
	if !ok || err != nil {
		t.Fatal("not ok:", ok, "err:", err)
	}

	// Recover original bytes
	buf := new(bytes.Buffer)
	err = enc.Join(buf, shards, len(data))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("recovered bytes do not match")
	}

	// Corrupt a shard
	shards[0] = nil
	shards[1][0], shards[1][500] = 75, 75

	// Should reconstruct (but with corrupted data)
	err = enc.Reconstruct(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err = enc.Verify(shards)
	if ok || err != nil {
		t.Fatal("error or ok:", ok, "err:", err)
	}

	// Recovered data should not match original
	buf.Reset()
	err = enc.Join(buf, shards, len(data))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(buf.Bytes(), data) {
		t.Fatal("corrupted data matches original")
	}
}

// TestAddMod8BugTriggersCorruption demonstrates that the addMod8 bug can cause silent data corruption
// when using the public API with crafted input.
func TestAddMod8BugTriggersCorruption(t *testing.T) {
	numData := 10
	numParity := 5
	shardSize := 64
	patterns := make([][][]byte, 4)

	// Pattern 1: All zeros, all ones, etc.
	patterns[0] = make([][]byte, numData)
	for i := range patterns[0] {
		patterns[0][i] = bytes.Repeat([]byte{byte(i)}, shardSize)
	}
	// Pattern 2: All 128s
	patterns[1] = make([][]byte, numData)
	for i := range patterns[1] {
		patterns[1][i] = bytes.Repeat([]byte{128}, shardSize)
	}
	// Pattern 3: All 255s
	patterns[2] = make([][]byte, numData)
	for i := range patterns[2] {
		patterns[2][i] = bytes.Repeat([]byte{255}, shardSize)
	}
	// Pattern 4: Random data
	patterns[3] = make([][]byte, numData)
	for i := range patterns[3] {
		patterns[3][i] = make([]byte, shardSize)
		for j := range patterns[3][i] {
			patterns[3][i][j] = byte(rand.Intn(256))
		}
	}

	for p, data := range patterns {
		enc, err := New(numData, numParity, WithLeopardGF(true))
		if err != nil {
			t.Fatalf("Pattern %d: failed to create encoder: %v", p, err)
		}
		// Reset counters
		addMod8CallCount = 0
		addMod8Return255Count = 0

		shards := append(data, make([][]byte, numParity)...)
		for i := range shards[numData:] {
			shards[numData+i] = make([]byte, shardSize)
		}
		err = enc.Encode(shards)
		if err != nil {
			t.Fatalf("Pattern %d: encode failed: %v", p, err)
		}
		t.Logf("Pattern %d : addMod8 was called %d times; returned 255 %d times", p, addMod8CallCount, addMod8Return255Count)

		// Save original data
		orig := make([][]byte, len(shards))
		for i := range shards {
			orig[i] = make([]byte, len(shards[i]))
			copy(orig[i], shards[i])
		}
		// Erase a data shard (simulate loss)
		erased := 0
		shards[erased] = nil
		// Attempt to reconstruct
		err = enc.Reconstruct(shards)
		if err != nil {
			t.Fatalf("Pattern %d: reconstruct failed: %v", p, err)
		}
		// Compare reconstructed shard to original
		if !bytes.Equal(shards[erased], orig[erased]) {
			t.Fatalf("Pattern %d: data corruption detected in shard %d", p, erased)
		}

		shards[0] = nil
		err = enc.Reconstruct(shards)
		if err != nil {
			t.Fatalf("Pattern %d: reconstruct failed: %v", p, err)
		}
		if !bytes.Equal(shards[0], data[0]) {
			t.Fatalf("Pattern %d: Data corruption detected: expected %v, got %v", p, data[0], shards[0])
		}
		println("Pattern", p, ": addMod8 was called", addMod8CallCount, "times; returned 255", addMod8Return255Count, "times")
	}
}

func TestAddMod8Direct(t *testing.T) {
	got := addMod8(128, 127) // 128+127=255
	if got != 0 {            // Should wrap to 0 in GF(256)
		t.Fatalf("addMod8(128,127) = %d, want 0", got)
	}
}

func TestEncoderReconstructFailLeo(t *testing.T) {
	// Create some sample data
	var data = make([]byte, 2<<20)
	fillRandom(data)

	// Create 5 data slices of 50000 elements each
	enc, err := New(500, 300, testOptions()...)
	if err != nil {
		t.Fatal(err)
	}
	shards, err := enc.Split(data)
	if err != nil {
		t.Fatal(err)
	}
	err = enc.Encode(shards)
	if err != nil {
		t.Fatal(err)
	}

	// Check that it verifies
	ok, err := enc.Verify(shards)
	if !ok || err != nil {
		t.Fatal("not ok:", ok, "err:", err)
	}

	// Delete more than parity shards
	for i := 0; i < 301; i++ {
		shards[i] = nil
	}

	// Should not reconstruct
	err = enc.Reconstruct(shards)
	if err != ErrTooFewShards {
		t.Fatal("want ErrTooFewShards, got:", err)
	}
}

func TestSplitJoinLeo(t *testing.T) {
	var data = make([]byte, (250<<10)-1)
	fillRandom(data)

	enc, _ := New(500, 300, testOptions()...)
	shards, err := enc.Split(data)
	if err != nil {
		t.Fatal(err)
	}

	_, err = enc.Split([]byte{})
	if err != ErrShortData {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}

	buf := new(bytes.Buffer)
	err = enc.Join(buf, shards, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), data[:5000]) {
		t.Fatal("recovered data does match original")
	}

	err = enc.Join(buf, [][]byte{}, 0)
	if err != ErrTooFewShards {
		t.Errorf("expected %v, got %v", ErrTooFewShards, err)
	}

	err = enc.Join(buf, shards, len(data)+500*64)
	if err != ErrShortData {
		t.Errorf("expected %v, got %v", ErrShortData, err)
	}

	shards[0] = nil
	err = enc.Join(buf, shards, len(data))
	if err != ErrReconstructRequired {
		t.Errorf("expected %v, got %v", ErrReconstructRequired, err)
	}
}
