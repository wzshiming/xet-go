package hf_xet_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/wzshiming/xet-go/hf_xet"
)

// hexRe matches the Xet content-addressable hash returned by HashFiles.
// xet-core hashes are lowercase hex strings.
var hexRe = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// TestHashFiles verifies the Rust → CGo binding end-to-end:
//
//  1. A temporary file is written with known content.
//  2. HashFiles is called; the result must carry no error.
//  3. Exactly one XetUploadInfo is returned with a non-empty hex hash and a
//     file_size equal to the number of bytes written.
//  4. FreeUploadResult releases the result without crashing.
func TestHashFiles(t *testing.T) {
	const content = "hello xet world"

	// Write a temporary file with deterministic content.
	f, err := os.CreateTemp("", "xet-hash-test-*.dat")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	paths := []string{f.Name()}
	result := hf_xet.HashFiles(paths, uint64(len(paths)))
	if result == nil {
		t.Fatal("HashFiles returned nil")
	}
	defer hf_xet.FreeUploadResult(result)

	// Check for error.
	if errMsg := result.Err(); errMsg != "" {
		t.Fatalf("HashFiles returned error: %s", errMsg)
	}

	// Verify item count.
	if result.Len() != len(paths) {
		t.Fatalf("expected %d result item(s), got %d", len(paths), result.Len())
	}

	// Inspect the single item.
	hash := result.HashAt(0)
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if !hexRe.MatchString(hash) {
		t.Fatalf("hash %q is not a hex string", hash)
	}

	fileSize := result.FileSizeAt(0)
	if fileSize != uint64(len(content)) {
		t.Fatalf("expected file_size=%d, got %d", len(content), fileSize)
	}

	t.Logf("hash=%s file_size=%d", hash, fileSize)
}
