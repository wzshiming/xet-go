package xetgo_test

import (
	"fmt"
	"log"

	"github.com/wzshiming/xet-go"
)

// ExampleChunkData demonstrates how to chunk data into content-addressable chunks.
func ExampleChunkData() {
	data := []byte("This is some sample data that will be chunked using content-defined chunking algorithm")

	chunks, err := xetgo.ChunkData(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Data chunked into %d chunks\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("Chunk %d: size=%d, hash=%s\n", i+1, chunk.Size, chunk.Hash[:16]+"...")
	}
}

// ExampleHashChunk demonstrates how to compute the hash of a single chunk.
func ExampleHashChunk() {
	data := []byte("Sample chunk data")

	hash, err := xetgo.HashChunk(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Chunk hash: %s\n", hash[:16]+"...")
}

// ExampleComputeXorbHash demonstrates how to compute a XORB hash from chunk information.
// The XORB hash allows efficient verification of file integrity without downloading all chunks.
func ExampleComputeXorbHash() {
	// First, chunk some data
	data := []byte("Example data that will be chunked and then hashed with XORB algorithm")

	chunks, err := xetgo.ChunkData(data)
	if err != nil {
		log.Fatal(err)
	}

	// Compute XORB hash from the chunks
	xorbHash, err := xetgo.ComputeXorbHash(chunks)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("XORB hash: %s\n", xorbHash[:16]+"...")
}

// ExampleComputeXorbHash_manual demonstrates computing XORB hash with manually created chunks.
func ExampleComputeXorbHash_manual() {
	// Create some sample chunks with known hashes
	data1 := []byte("First chunk data")
	hash1, err := xetgo.HashChunk(data1)
	if err != nil {
		log.Fatal(err)
	}

	data2 := []byte("Second chunk data")
	hash2, err := xetgo.HashChunk(data2)
	if err != nil {
		log.Fatal(err)
	}

	// Build chunk info manually
	chunks := []xetgo.ChunkInfo{
		{Hash: hash1, Size: uint64(len(data1))},
		{Hash: hash2, Size: uint64(len(data2))},
	}

	// Compute XORB hash
	xorbHash, err := xetgo.ComputeXorbHash(chunks)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("XORB hash from manual chunks: %s\n", xorbHash[:16]+"...")
}
