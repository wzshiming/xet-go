# XORB Support in xet-go

This repository now includes full support for XORB (XOR-based) hashing, which allows efficient verification of file integrity without downloading all chunks.

## Features

The following XORB-related functions are available:

### 1. ChunkData

Splits raw data into content-addressable chunks using content-defined chunking (CDC):

```go
data := []byte("your data here")
chunks, err := xetgo.ChunkData(data)
if err != nil {
    log.Fatal(err)
}

for _, chunk := range chunks {
    fmt.Printf("Chunk: hash=%s, size=%d\n", chunk.Hash, chunk.Size)
}
```

### 2. HashChunk

Computes the Xet hash for a single chunk of data:

```go
data := []byte("chunk data")
hash, err := xetgo.HashChunk(data)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Hash: %s\n", hash)
```

### 3. ComputeXorbHash

Computes the XORB hash from a list of chunk hashes and sizes. The XORB hash is an XOR-based aggregate hash used for efficient verification of file integrity:

```go
// After chunking data
chunks, _ := xetgo.ChunkData(data)

// Compute XORB hash
xorbHash, err := xetgo.ComputeXorbHash(chunks)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("XORB hash: %s\n", xorbHash)
```

## API Reference

### Types

```go
// ChunkInfo describes a single chunk
type ChunkInfo struct {
    Hash string  // Chunk hash (hex string)
    Size uint64  // Chunk size in bytes
}
```

### Functions

```go
// ChunkData splits data into content-addressable chunks
func ChunkData(data []byte) ([]ChunkInfo, error)

// HashChunk computes the hash for a single chunk
func HashChunk(data []byte) (string, error)

// ComputeXorbHash computes the XORB hash from chunk metadata
func ComputeXorbHash(chunks []ChunkInfo) (string, error)
```

## Testing

The implementation includes comprehensive tests:

```bash
go test -v
```

Tests include:
- `TestChunkData`: Validates data chunking
- `TestHashChunk`: Validates single chunk hashing
- `TestHashChunkEmpty`: Validates error handling for empty data
- `TestComputeXorbHash`: Validates XORB hash computation
- `TestComputeXorbHashEmpty`: Validates error handling for empty chunk lists
- `TestComputeXorbHashWithManualChunks`: Validates XORB hash with manually created chunks

## Implementation Details

The XORB implementation is provided by the xet-core Rust library and exposed through:

1. **C API** (`xet/xet.h`): Defines the C-compatible interface
2. **Rust Implementation** (`xet-sys/src/lib.rs`): Wraps xet-core's chunking and XORB functionality
3. **Go Bindings** (`xet/xet.go`): Auto-generated CGo bindings
4. **High-level API** (`xetgo.go`): Idiomatic Go wrappers

## References

- [xet-core xorb_object implementation](https://github.com/huggingface/xet-core/tree/main/xet_core_structures/src/xorb_object)
