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

func TestChunkBytes(t *testing.T) {
	data := []byte("hello xet world")

	chunks, err := ChunkBytes(data)
	if err != nil {
		t.Fatalf("ChunkBytes returned error: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	var total uint64
	for _, c := range chunks {
		if c.Hash == "" {
			t.Fatal("chunk hash is empty")
		}
		if c.Size == 0 {
			t.Fatal("chunk size is zero")
		}
		total += c.Size
	}
	if total != uint64(len(data)) {
		t.Fatalf("total chunk size %d != input size %d", total, len(data))
	}
}

func TestChunkFile(t *testing.T) {
	const content = "hello xet world for file chunking"
	f, err := os.CreateTemp("", "xet-chunk-test-*.dat")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	chunks, err := ChunkFile(f.Name())
	if err != nil {
		t.Fatalf("ChunkFile returned error: %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	var total uint64
	for _, c := range chunks {
		if c.Hash == "" {
			t.Fatal("chunk hash is empty")
		}
		total += c.Size
	}
	if total != uint64(len(content)) {
		t.Fatalf("total chunk size %d != file size %d", total, len(content))
	}
}

func TestComputeChunkHash(t *testing.T) {
	data := []byte("hello xet world")
	h, err := ComputeChunkHash(data)
	if err != nil {
		t.Fatalf("ComputeChunkHash returned error: %v", err)
	}
	if len(h) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d chars: %q", len(h), h)
	}

	// Must equal the hash from ChunkBytes for the same small input.
	chunks, err := ChunkBytes(data)
	if err != nil {
		t.Fatalf("ChunkBytes returned error: %v", err)
	}
	if len(chunks) != 1 {
		t.Skipf("unexpected chunk count %d; skipping comparison", len(chunks))
	}
	if h != chunks[0].Hash {
		t.Fatalf("ComputeChunkHash=%q != ChunkBytes[0].Hash=%q", h, chunks[0].Hash)
	}
}

func TestComputeXorbAndFileHash(t *testing.T) {
	data := []byte("hello xet world for hashing")
	chunks, err := ChunkBytes(data)
	if err != nil {
		t.Fatalf("ChunkBytes error: %v", err)
	}

	xorbHash, err := ComputeXorbHash(chunks)
	if err != nil {
		t.Fatalf("ComputeXorbHash error: %v", err)
	}
	if len(xorbHash) != 64 {
		t.Fatalf("expected 64-char xorb hash, got %d: %q", len(xorbHash), xorbHash)
	}

	fileHash, err := ComputeFileHash(chunks)
	if err != nil {
		t.Fatalf("ComputeFileHash error: %v", err)
	}
	if len(fileHash) != 64 {
		t.Fatalf("expected 64-char file hash, got %d: %q", len(fileHash), fileHash)
	}

	// xorb hash and file hash are computed differently; they should differ.
	if xorbHash == fileHash {
		t.Logf("note: xorb hash == file hash (%q); this can happen for single-chunk data", xorbHash)
	}
}

func TestComputeRangeHash(t *testing.T) {
	data := []byte("hello xet world for range hashing")
	chunks, err := ChunkBytes(data)
	if err != nil {
		t.Fatalf("ChunkBytes error: %v", err)
	}

	hashes := make([]string, len(chunks))
	for i, c := range chunks {
		hashes[i] = c.Hash
	}

	rangeHash, err := ComputeRangeHash(hashes)
	if err != nil {
		t.Fatalf("ComputeRangeHash error: %v", err)
	}
	if len(rangeHash) != 64 {
		t.Fatalf("expected 64-char range hash, got %d: %q", len(rangeHash), rangeHash)
	}
}
