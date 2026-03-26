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
