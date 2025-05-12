package reedsolomon

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAddMod8Pattern128(t *testing.T) {
	numData := 10
	numParity := 5
	shardSize := 64 // Must be multiple of 8 for Leopard
	// All data shards filled with 128
	data := make([][]byte, numData)
	for i := range data {
		data[i] = bytes.Repeat([]byte{128}, shardSize)
	}
	shards := append(data, make([][]byte, numParity)...)
	for i := range shards[numData:] {
		shards[numData+i] = make([]byte, shardSize)
	}
	enc, err := New(numData, numParity, WithLeopardGF(true))
	if err != nil {
		t.Fatalf("failed to create encoder: %v", err)
	}
	err = enc.Encode(shards)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
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
		t.Fatalf("reconstruct failed: %v", err)
	}
	// Compare reconstructed shard to original
	if !bytes.Equal(shards[erased], orig[erased]) {
		// Show a hex dump of the first 64 bytes (or the whole shard if smaller)
		max := len(shards[erased])
		if max > 64 {
			max = 64
		}
		origHex := fmt.Sprintf("% x", orig[erased][:max])
		reconHex := fmt.Sprintf("% x", shards[erased][:max])
		t.Fatalf("\nPoC: Silent data corruption detected in shard %d!\nOriginal (first %d bytes):   %s\nReconstructed (first %d bytes): %s\n\nFull original:      % x\nFull reconstructed: % x\n\nThis demonstrates the addMod8 bug causes silent data corruption during reconstruction when all data shards are filled with 128.", erased, max, origHex, max, reconHex, orig[erased], shards[erased])
	}
}
