// Package hf_xet provides Go CGo bindings for the xet-core HuggingFace Xet
// storage library.
//
// # Linking
//
// This package requires the hf_xet native shared library to be available at
// link time and at runtime.  Set the CGO_LDFLAGS and CGO_CFLAGS environment
// variables (or install the library system-wide) before building.
//
// Example:
//
//	export CGO_LDFLAGS="-L/path/to/hf_xet/lib -lhf_xet"
//	export CGO_CFLAGS="-I/path/to/hf_xet/include"
//	go build ./...
//
// On Linux the runtime linker also needs to find the shared library, either
// via LD_LIBRARY_PATH or by installing it under a path known to ldconfig.

package hf_xet

/*
#cgo linux   LDFLAGS: -lhf_xet
#cgo darwin  LDFLAGS: -lhf_xet
#cgo windows LDFLAGS: -lhf_xet
*/
import "C"
