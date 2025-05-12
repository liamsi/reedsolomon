package reedsolomon

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

// TestAddMod8FuzzRoundtrip fuzzes the roundtrip reconstruction to try to trigger silent corruption.
func TestAddMod8FuzzRoundtrip(t *testing.T) {
	rand.Seed(time.Now().UnixNano()) // Non-deterministic seed
	const numData = 10
	const numParity = 5
	const shardSize = 64
	iter := 0
	for {
		iter++
		// Fill data shards with random bytes
		data := make([][]byte, numData)
		for i := range data {
			data[i] = make([]byte, shardSize)
			_, _ = rand.Read(data[i])
		}
		shards := append(data, make([][]byte, numParity)...)
		for i := range shards[numData:] {
			shards[numData+i] = make([]byte, shardSize)
		}
		enc, err := New(numData, numParity, WithLeopardGF(true))
		if err != nil {
			t.Fatalf("iteration %d: failed to create encoder: %v", iter, err)
		}
		err = enc.Encode(shards)
		if err != nil {
			t.Fatalf("iteration %d: encode failed: %v", iter, err)
		}
		// Save original data
		orig := make([][]byte, len(shards))
		for i := range shards {
			orig[i] = make([]byte, len(shards[i]))
			copy(orig[i], shards[i])
		}
		// Erase the maximum number of data shards (numParity) to ensure the error is not masked
		// by the parity shards
		erasedSet := map[int]struct{}{}
		for len(erasedSet) < numParity {
			e := rand.Intn(numData)
			erasedSet[e] = struct{}{}
		}
		erasedList := make([]int, 0, numParity)
		for e := range erasedSet {
			erasedList = append(erasedList, e)
			shards[e] = nil
		}
		// Attempt to reconstruct
		err = enc.Reconstruct(shards)
		if err != nil {
			t.Fatalf("iteration %d: reconstruct failed: %v", iter, err)
		}
		// Verify the reconstructed shards
		ok, verr := enc.Verify(shards)
		if !ok || verr != nil {
			t.Fatalf("iteration %d: verify failed after reconstruction: ok=%v err=%v", iter, ok, verr)
		}
		// Compare all erased/reconstructed shards to their originals
		for _, erased := range erasedList {
			if !bytes.Equal(shards[erased], orig[erased]) {
				t.Fatalf("iteration %d: data corruption detected in shard %d\noriginal: %v\nreconstructed: %v", iter, erased, orig[erased], shards[erased])
			}
		}
		if iter%1000 == 0 {
			t.Logf("%d iterations completed without corruption", iter)
		}
	}
}
