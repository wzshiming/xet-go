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
