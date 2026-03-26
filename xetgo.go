// Package xetgo provides an idiomatic Go API for HuggingFace Xet storage.
//
// It wraps the low-level CGo bindings in [github.com/wzshiming/xet-go/xet],
// hiding manual memory management and C-style types behind clean Go types and
// standard error returns.
//
// # Building
//
// The native implementation is compiled from the Rust crate in xet-sys/ into
// a static library.  Build it before using this package:
//
//	make rust-build
//
// # Usage
//
//	results, err := xetgo.HashFiles([]string{"/path/to/file"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(results[0].Hash)
package xetgo

import (
	"errors"

	"github.com/wzshiming/xet-go/xet"
)

// TokenInfo holds authentication credentials for Xet storage.
type TokenInfo struct {
	// Token is the bearer token string.
	Token string
	// Expiry is the token expiry as a UNIX epoch timestamp in seconds.
	// A value of 0 means the token does not expire.
	Expiry uint64
}

// UploadResult describes a single uploaded or hashed file.
type UploadResult struct {
	// Hash is the Xet content-addressable hash (hex string).
	Hash string
	// FileSize is the size of the file in bytes.
	FileSize uint64
	// SHA256 is the SHA-256 hex digest of the file, or empty if not computed.
	SHA256 string
}

// DownloadRequest is the input descriptor for a single file to download.
type DownloadRequest struct {
	// DestinationPath is the local filesystem path where the file will be written.
	DestinationPath string
	// Hash is the Xet content-hash to fetch (hex string).
	Hash string
	// FileSize is the expected size of the file in bytes, or -1 if unknown.
	FileSize int64
}

// HashFiles computes Xet content-hashes for local files without uploading them.
// This is useful for pre-flight deduplication checks.
//
// filePaths must not be empty.  The returned slice has the same length as
// filePaths with one UploadResult per file, in the same order.
func HashFiles(filePaths []string) ([]UploadResult, error) {
	raw := xet.HashFiles(filePaths, uint64(len(filePaths)))
	if raw == nil {
		return nil, errors.New("xetgo: HashFiles returned nil result")
	}
	defer xet.FreeUploadResult(raw)
	return collectUploadResults(raw)
}

// UploadFiles uploads local files to Xet storage.
//
// endpoint is the CAS endpoint URL.  Pass "" to use the default endpoint.
// token is the authentication credential.  Pass nil for unauthenticated access.
// sha256s may be nil (SHA-256 digests are computed automatically) or a slice of
// pre-computed SHA-256 hex digests, one per file in the same order as filePaths.
// skipSHA256 skips SHA-256 computation and verification entirely; it is
// mutually exclusive with providing a non-nil sha256s slice.
//
// The returned slice has the same length as filePaths with one UploadResult per
// file, in the same order.
func UploadFiles(filePaths []string, endpoint string, token *TokenInfo, sha256s []string, skipSHA256 bool) ([]UploadResult, error) {
	ti := toRawTokenInfo(token)
	skip := int32(0)
	if skipSHA256 {
		skip = 1
	}
	raw := xet.UploadFiles(filePaths, uint64(len(filePaths)), endpoint, ti, sha256s, uint64(len(sha256s)), skip)
	if raw == nil {
		return nil, errors.New("xetgo: UploadFiles returned nil result")
	}
	defer xet.FreeUploadResult(raw)
	return collectUploadResults(raw)
}

// DownloadFiles downloads files from Xet storage to the local filesystem.
//
// endpoint is the CAS endpoint URL.  Pass "" to use the default endpoint.
// token is the authentication credential.  Pass nil for unauthenticated access.
//
// The returned slice contains the local destination paths of the downloaded
// files, in the same order as the input files slice.
func DownloadFiles(files []DownloadRequest, endpoint string, token *TokenInfo) ([]string, error) {
	infos := make([]xet.Xetdownloadinfo, len(files))
	for i, f := range files {
		infos[i] = xet.Xetdownloadinfo{
			DestinationPath: []byte(f.DestinationPath),
			Hash:            []byte(f.Hash),
			FileSize:        f.FileSize,
		}
	}
	ti := toRawTokenInfo(token)
	raw := xet.DownloadFiles(infos, uint64(len(infos)), endpoint, ti)
	if raw == nil {
		return nil, errors.New("xetgo: DownloadFiles returned nil result")
	}
	defer xet.FreeDownloadResult(raw)
	return collectDownloadResults(raw)
}

// toRawTokenInfo converts a *TokenInfo to the xet low-level type.
// It returns nil when token is nil (unauthenticated access).
func toRawTokenInfo(token *TokenInfo) *xet.Xettokeninfo {
	if token == nil {
		return nil
	}
	return &xet.Xettokeninfo{
		Token:  []byte(token.Token),
		Expiry: token.Expiry,
	}
}

// collectUploadResults converts a raw *xet.Xetuploadresult into a Go slice,
// returning an error if the underlying C result carries an error message.
func collectUploadResults(raw *xet.Xetuploadresult) ([]UploadResult, error) {
	if errMsg := raw.Err(); errMsg != "" {
		return nil, errors.New(errMsg)
	}
	n := raw.Len()
	results := make([]UploadResult, n)
	for i := range results {
		results[i] = UploadResult{
			Hash:     raw.HashAt(i),
			FileSize: raw.FileSizeAt(i),
			SHA256:   raw.SHA256At(i),
		}
	}
	return results, nil
}

// collectDownloadResults converts a raw *xet.Xetdownloadresult into a Go
// string slice, returning an error if the underlying C result carries an error.
func collectDownloadResults(raw *xet.Xetdownloadresult) ([]string, error) {
	if errMsg := raw.Err(); errMsg != "" {
		return nil, errors.New(errMsg)
	}
	n := raw.Len()
	paths := make([]string, n)
	for i := range paths {
		paths[i] = raw.PathAt(i)
	}
	return paths, nil
}

// ChunkInfo describes a single content-defined chunk produced by the xet
// chunking algorithm.
type ChunkInfo struct {
	// Hash is the Xet chunk hash (hex string).
	Hash string
	// Size is the chunk size in bytes.
	Size uint64
}

// ChunkBytes splits data into content-defined chunks.
//
// Returns one ChunkInfo per chunk, in order.  An empty slice is returned for
// empty input.
func ChunkBytes(data []byte) ([]ChunkInfo, error) {
	raw, err := xet.ChunkBytes(data)
	if err != nil {
		return nil, err
	}
	return convertChunkInfos(raw), nil
}

// ChunkFile splits the file at filePath into content-defined chunks.
//
// Returns one ChunkInfo per chunk, in the order they appear in the file.
func ChunkFile(filePath string) ([]ChunkInfo, error) {
	raw, err := xet.ChunkFile(filePath)
	if err != nil {
		return nil, err
	}
	return convertChunkInfos(raw), nil
}

// ComputeChunkHash computes the Xet chunk hash of raw bytes.
func ComputeChunkHash(data []byte) (string, error) {
	return xet.ComputeChunkHash(data)
}

// ComputeXorbHash computes the xorb hash from an ordered list of chunk infos.
func ComputeXorbHash(chunks []ChunkInfo) (string, error) {
	return xet.ComputeXorbHash(convertToXetChunkInfos(chunks))
}

// ComputeFileHash computes the file hash from an ordered list of chunk infos.
func ComputeFileHash(chunks []ChunkInfo) (string, error) {
	return xet.ComputeFileHash(convertToXetChunkInfos(chunks))
}

// ComputeRangeHash computes the range hash from an ordered list of chunk
// hashes (hex strings).
func ComputeRangeHash(hashes []string) (string, error) {
	return xet.ComputeRangeHash(hashes)
}

// XorbCheckResult holds the result of checking an xorb object.
type XorbCheckResult struct {
	// XorbHash is the computed xorb hash (hex string).
	XorbHash string
	// Chunks holds the per-chunk hashes and sizes.
	Chunks []ChunkInfo
	// TotalBytes is the total uncompressed data size of the xorb in bytes.
	TotalBytes uint64
}

// CheckXorbBytes deserializes an xorb object from bytes, computes its hash,
// and returns chunk information.
func CheckXorbBytes(data []byte) (*XorbCheckResult, error) {
	raw, err := xet.CheckXorbBytes(data)
	if err != nil {
		return nil, err
	}
	return convertXorbCheckResult(raw), nil
}

// CheckXorbFile deserializes an xorb object from the file at filePath,
// computes its hash, and returns chunk information.
func CheckXorbFile(filePath string) (*XorbCheckResult, error) {
	raw, err := xet.CheckXorbFile(filePath)
	if err != nil {
		return nil, err
	}
	return convertXorbCheckResult(raw), nil
}

// convertChunkInfos converts xet.ChunkInfo values to xetgo.ChunkInfo values.
func convertChunkInfos(raw []xet.ChunkInfo) []ChunkInfo {
	out := make([]ChunkInfo, len(raw))
	for i, c := range raw {
		out[i] = ChunkInfo{Hash: c.Hash, Size: c.Size}
	}
	return out
}

// convertToXetChunkInfos converts xetgo.ChunkInfo values to xet.ChunkInfo values.
func convertToXetChunkInfos(chunks []ChunkInfo) []xet.ChunkInfo {
	out := make([]xet.ChunkInfo, len(chunks))
	for i, c := range chunks {
		out[i] = xet.ChunkInfo{Hash: c.Hash, Size: c.Size}
	}
	return out
}

// convertXorbCheckResult converts a *xet.XorbCheckResult to *xetgo.XorbCheckResult.
func convertXorbCheckResult(raw *xet.XorbCheckResult) *XorbCheckResult {
	return &XorbCheckResult{
		XorbHash:   raw.XorbHash,
		Chunks:     convertChunkInfos(raw.Chunks),
		TotalBytes: raw.TotalBytes,
	}
}
