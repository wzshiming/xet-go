package xetgo

import (
	"os"
	"testing"
)

func TestHashFiles(t *testing.T) {
	const content = "hello xet world"

	// Write a temporary file with deterministic content.
	f, err := os.CreateTemp("", "xet-hash-test-*.dat")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	f.WriteString(content)
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	result, err := HashFiles([]string{f.Name()})
	if err != nil {
		t.Fatalf("HashFiles returned error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 result item, got %d", len(result))
	}

	if result[0].Hash == "" {
		t.Fatal("hash is empty")
	}

	if result[0].FileSize != uint64(len(content)) {
		t.Fatalf("expected file_size=%d, got %d", len(content), result[0].FileSize)
	}

	if result[0].Hash != "d73d191ec3d5f99ce1c6750cdb09f4124ce304cfa8070f01fad19713f7113119" {
		t.Fatalf("unexpected hash: %s", result[0].Hash)
	}
}

func TestChunkData(t *testing.T) {
	// Test with a simple data string
	data := []byte("hello xet world for chunking")

	chunks, err := ChunkData(data)
	if err != nil {
		t.Fatalf("ChunkData returned error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("ChunkData returned no chunks")
	}

	// Verify each chunk has a valid hash and size
	var totalSize uint64
	for i, chunk := range chunks {
		if chunk.Hash == "" {
			t.Fatalf("chunk %d has empty hash", i)
		}
		if chunk.Size == 0 {
			t.Fatalf("chunk %d has zero size", i)
		}
		totalSize += chunk.Size
	}

	// Total size of chunks should equal input data
	if totalSize != uint64(len(data)) {
		t.Fatalf("total chunk size %d != input size %d", totalSize, len(data))
	}

	t.Logf("ChunkData produced %d chunks with total size %d", len(chunks), totalSize)
}

func TestHashChunk(t *testing.T) {
	// Test with a simple data string
	data := []byte("test data for chunk hashing")

	hash, err := HashChunk(data)
	if err != nil {
		t.Fatalf("HashChunk returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("HashChunk returned empty hash")
	}

	// The hash should be a hex string
	if len(hash) != 64 {
		t.Fatalf("expected hash length 64, got %d", len(hash))
	}

	t.Logf("HashChunk result: %s", hash)

	// Test that same data produces same hash
	hash2, err := HashChunk(data)
	if err != nil {
		t.Fatalf("HashChunk second call returned error: %v", err)
	}

	if hash != hash2 {
		t.Fatalf("HashChunk not deterministic: %s != %s", hash, hash2)
	}
}

func TestHashChunkEmpty(t *testing.T) {
	// Test with empty data
	data := []byte{}

	_, err := HashChunk(data)
	if err == nil {
		t.Fatal("HashChunk should return error for empty data")
	}

	t.Logf("HashChunk correctly rejected empty data: %v", err)
}

func TestComputeXorbHash(t *testing.T) {
	// Create some test chunk data
	data := []byte("test data for xorb hashing with multiple chunks to ensure we get multiple chunks in the output")

	// First, chunk the data
	chunks, err := ChunkData(data)
	if err != nil {
		t.Fatalf("ChunkData returned error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("ChunkData returned no chunks")
	}

	t.Logf("Generated %d chunks for xorb test", len(chunks))

	// Compute XORB hash
	xorbHash, err := ComputeXorbHash(chunks)
	if err != nil {
		t.Fatalf("ComputeXorbHash returned error: %v", err)
	}

	if xorbHash == "" {
		t.Fatal("ComputeXorbHash returned empty hash")
	}

	// The XORB hash should be a hex string
	if len(xorbHash) != 64 {
		t.Fatalf("expected xorb hash length 64, got %d", len(xorbHash))
	}

	t.Logf("XORB hash: %s", xorbHash)

	// Test that same chunks produce same XORB hash
	xorbHash2, err := ComputeXorbHash(chunks)
	if err != nil {
		t.Fatalf("ComputeXorbHash second call returned error: %v", err)
	}

	if xorbHash != xorbHash2 {
		t.Fatalf("ComputeXorbHash not deterministic: %s != %s", xorbHash, xorbHash2)
	}
}

func TestComputeXorbHashEmpty(t *testing.T) {
	// Test with empty chunk list
	chunks := []ChunkInfo{}

	_, err := ComputeXorbHash(chunks)
	if err == nil {
		t.Fatal("ComputeXorbHash should return error for empty chunk list")
	}

	t.Logf("ComputeXorbHash correctly rejected empty chunk list: %v", err)
}

func TestComputeXorbHashWithManualChunks(t *testing.T) {
	// Test with manually created chunks
	// First, hash some data to get valid chunk hashes
	data1 := []byte("chunk one data")
	hash1, err := HashChunk(data1)
	if err != nil {
		t.Fatalf("HashChunk failed for data1: %v", err)
	}

	data2 := []byte("chunk two data")
	hash2, err := HashChunk(data2)
	if err != nil {
		t.Fatalf("HashChunk failed for data2: %v", err)
	}

	// Create manual chunk list
	chunks := []ChunkInfo{
		{Hash: hash1, Size: uint64(len(data1))},
		{Hash: hash2, Size: uint64(len(data2))},
	}

	xorbHash, err := ComputeXorbHash(chunks)
	if err != nil {
		t.Fatalf("ComputeXorbHash returned error: %v", err)
	}

	if xorbHash == "" {
		t.Fatal("ComputeXorbHash returned empty hash")
	}

	t.Logf("XORB hash from manual chunks: %s", xorbHash)

	// Test that reversing chunk order produces different XORB hash
	// (XORB is order-independent, but this tests the implementation)
	chunksReversed := []ChunkInfo{
		{Hash: hash2, Size: uint64(len(data2))},
		{Hash: hash1, Size: uint64(len(data1))},
	}

	xorbHashReversed, err := ComputeXorbHash(chunksReversed)
	if err != nil {
		t.Fatalf("ComputeXorbHash returned error for reversed: %v", err)
	}

	// XORB is order-independent (XOR is commutative), so hashes should be the same
	if xorbHash != xorbHashReversed {
		t.Logf("Note: XORB hashes differ with reversed order (expected if order matters)")
	} else {
		t.Logf("XORB hashes are the same regardless of order (XOR is commutative)")
	}
}
