// Package xet provides Go CGo bindings for the xet-core HuggingFace Xet
// storage library.
//
// This file contains handwritten bindings for the chunk, hash, and xorb-check
// operations added to xet.h.  Unlike the c-for-go–generated files it uses a
// simpler, direct CGo style that mirrors the patterns in result_helpers.go.

package xet

/*
#include "xet.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

// ChunkInfo holds the hash and size of a single content-defined chunk.
type ChunkInfo struct {
	Hash string
	Size uint64
}

// XorbCheckResult holds the result of an xorb integrity check.
type XorbCheckResult struct {
	XorbHash   string
	Chunks     []ChunkInfo
	TotalBytes uint64
}

// --------------------------------------------------------------------------
// ChunkBytes splits a raw byte slice into content-defined chunks.
// --------------------------------------------------------------------------

// ChunkBytes splits data into content-defined chunks and returns one ChunkInfo
// per chunk.
func ChunkBytes(data []byte) ([]ChunkInfo, error) {
	var dataPtr *C.uint8_t
	if len(data) > 0 {
		dataPtr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	raw := C.xet_chunk_data(dataPtr, C.size_t(len(data)))
	if raw == nil {
		return nil, errors.New("xet: xet_chunk_data returned nil")
	}
	defer C.xet_free_chunk_result(raw)
	return collectChunkResult(raw)
}

// ChunkFile splits the file at filePath into content-defined chunks.
func ChunkFile(filePath string) ([]ChunkInfo, error) {
	cpath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cpath))

	raw := C.xet_chunk_file(cpath)
	if raw == nil {
		return nil, errors.New("xet: xet_chunk_file returned nil")
	}
	defer C.xet_free_chunk_result(raw)
	return collectChunkResult(raw)
}

// collectChunkResult converts a *C.XetChunkResult into Go types, returning an
// error if the C result carries an error message.
func collectChunkResult(raw *C.XetChunkResult) ([]ChunkInfo, error) {
	if raw.error != nil {
		return nil, errors.New(C.GoString(raw.error))
	}
	n := int(raw.count)
	if n == 0 {
		return nil, nil
	}
	items := (*[1 << 20]C.XetChunkInfo)(unsafe.Pointer(raw.items))[:n:n]
	chunks := make([]ChunkInfo, n)
	for i := range chunks {
		hash := ""
		if items[i].hash != nil {
			hash = C.GoString(items[i].hash)
		}
		chunks[i] = ChunkInfo{
			Hash: hash,
			Size: uint64(items[i].size),
		}
	}
	return chunks, nil
}

// --------------------------------------------------------------------------
// Hash functions
// --------------------------------------------------------------------------

// ComputeChunkHash returns the Xet chunk hash of the given bytes.
func ComputeChunkHash(data []byte) (string, error) {
	var dataPtr *C.uint8_t
	if len(data) > 0 {
		dataPtr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	raw := C.xet_compute_chunk_hash(dataPtr, C.size_t(len(data)))
	if raw == nil {
		return "", errors.New("xet: xet_compute_chunk_hash returned nil")
	}
	defer C.xet_free_hash_result(raw)
	return extractHashResult(raw)
}

// ComputeXorbHash returns the xorb hash for the given ordered chunk list.
func ComputeXorbHash(chunks []ChunkInfo) (string, error) {
	hashes, sizes := splitChunkSlice(chunks)
	raw := callXorbOrFileHash(hashes, sizes, true)
	if raw == nil {
		return "", errors.New("xet: xet_compute_xorb_hash returned nil")
	}
	defer C.xet_free_hash_result(raw)
	return extractHashResult(raw)
}

// ComputeFileHash returns the file hash for the given ordered chunk list.
func ComputeFileHash(chunks []ChunkInfo) (string, error) {
	hashes, sizes := splitChunkSlice(chunks)
	raw := callXorbOrFileHash(hashes, sizes, false)
	if raw == nil {
		return "", errors.New("xet: xet_compute_file_hash returned nil")
	}
	defer C.xet_free_hash_result(raw)
	return extractHashResult(raw)
}

// ComputeRangeHash returns the range hash for the given ordered chunk hashes.
func ComputeRangeHash(hashes []string) (string, error) {
	if len(hashes) == 0 {
		raw := C.xet_compute_range_hash(nil, 0)
		if raw == nil {
			return "", errors.New("xet: xet_compute_range_hash returned nil")
		}
		defer C.xet_free_hash_result(raw)
		return extractHashResult(raw)
	}
	cHashes := make([]*C.char, len(hashes))
	for i, h := range hashes {
		cHashes[i] = C.CString(h)
	}
	defer func() {
		for _, p := range cHashes {
			C.free(unsafe.Pointer(p))
		}
	}()
	raw := C.xet_compute_range_hash(
		(**C.char)(unsafe.Pointer(&cHashes[0])),
		C.size_t(len(cHashes)),
	)
	if raw == nil {
		return "", errors.New("xet: xet_compute_range_hash returned nil")
	}
	defer C.xet_free_hash_result(raw)
	return extractHashResult(raw)
}

// extractHashResult converts a *C.XetHashResult to a (string, error) pair.
func extractHashResult(raw *C.XetHashResult) (string, error) {
	if raw.error != nil {
		return "", errors.New(C.GoString(raw.error))
	}
	if raw.hash == nil {
		return "", nil
	}
	return C.GoString(raw.hash), nil
}

// splitChunkSlice converts a []ChunkInfo into parallel C-string and uint64 arrays.
func splitChunkSlice(chunks []ChunkInfo) ([]*C.char, []C.uint64_t) {
	cHashes := make([]*C.char, len(chunks))
	cSizes := make([]C.uint64_t, len(chunks))
	for i, c := range chunks {
		cHashes[i] = C.CString(c.Hash)
		cSizes[i] = C.uint64_t(c.Size)
	}
	return cHashes, cSizes
}

// callXorbOrFileHash calls xet_compute_xorb_hash (xorb=true) or
// xet_compute_file_hash (xorb=false) with the given parallel arrays.
// The caller is responsible for freeing cHashes strings.
func callXorbOrFileHash(cHashes []*C.char, cSizes []C.uint64_t, xorb bool) *C.XetHashResult {
	defer func() {
		for _, p := range cHashes {
			C.free(unsafe.Pointer(p))
		}
	}()
	if len(cHashes) == 0 {
		if xorb {
			return C.xet_compute_xorb_hash(nil, nil, 0)
		}
		return C.xet_compute_file_hash(nil, nil, 0)
	}
	hashPtr := (**C.char)(unsafe.Pointer(&cHashes[0]))
	sizePtr := (*C.uint64_t)(unsafe.Pointer(&cSizes[0]))
	count := C.size_t(len(cHashes))
	if xorb {
		return C.xet_compute_xorb_hash(hashPtr, sizePtr, count)
	}
	return C.xet_compute_file_hash(hashPtr, sizePtr, count)
}

// --------------------------------------------------------------------------
// Xorb-check functions
// --------------------------------------------------------------------------

// CheckXorbBytes deserializes an xorb object from bytes, computes its hash,
// and returns chunk information.
func CheckXorbBytes(data []byte) (*XorbCheckResult, error) {
	var dataPtr *C.uint8_t
	if len(data) > 0 {
		dataPtr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	raw := C.xet_check_xorb_data(dataPtr, C.size_t(len(data)))
	if raw == nil {
		return nil, errors.New("xet: xet_check_xorb_data returned nil")
	}
	defer C.xet_free_xorb_check_result(raw)
	return collectXorbCheckResult(raw)
}

// CheckXorbFile deserializes an xorb object from the file at filePath, computes
// its hash, and returns chunk information.
func CheckXorbFile(filePath string) (*XorbCheckResult, error) {
	cpath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cpath))

	raw := C.xet_check_xorb_file(cpath)
	if raw == nil {
		return nil, errors.New("xet: xet_check_xorb_file returned nil")
	}
	defer C.xet_free_xorb_check_result(raw)
	return collectXorbCheckResult(raw)
}

// collectXorbCheckResult converts a *C.XetXorbCheckResult into Go types.
func collectXorbCheckResult(raw *C.XetXorbCheckResult) (*XorbCheckResult, error) {
	if raw.error != nil {
		return nil, errors.New(C.GoString(raw.error))
	}
	xorbHash := ""
	if raw.xorb_hash != nil {
		xorbHash = C.GoString(raw.xorb_hash)
	}
	n := int(raw.chunk_count)
	var chunks []ChunkInfo
	if n > 0 {
		items := (*[1 << 20]C.XetChunkInfo)(unsafe.Pointer(raw.chunks))[:n:n]
		chunks = make([]ChunkInfo, n)
		for i := range chunks {
			hash := ""
			if items[i].hash != nil {
				hash = C.GoString(items[i].hash)
			}
			chunks[i] = ChunkInfo{
				Hash: hash,
				Size: uint64(items[i].size),
			}
		}
	}
	return &XorbCheckResult{
		XorbHash:   xorbHash,
		Chunks:     chunks,
		TotalBytes: uint64(raw.total_bytes),
	}, nil
}
