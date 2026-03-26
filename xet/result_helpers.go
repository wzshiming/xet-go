package xet

/*
#include "xet.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"
import "unsafe"

// Err returns the error message from an XetUploadResult, or an empty string
// when the operation succeeded (i.e. the C error pointer is NULL).
func (x *Xetuploadresult) Err() string {
	if x == nil || x.ref30b7a435 == nil {
		return ""
	}
	if x.ref30b7a435.error == nil {
		return ""
	}
	return C.GoString(x.ref30b7a435.error)
}

// Len returns the number of XetUploadInfo items in the result.
func (x *Xetuploadresult) Len() int {
	if x == nil || x.ref30b7a435 == nil {
		return 0
	}
	return int(x.ref30b7a435.count)
}

// HashAt returns the Xet content-hash string for the item at index i.
// It panics if i is out of range.
func (x *Xetuploadresult) HashAt(i int) string {
	n := x.Len()
	if i < 0 || i >= n {
		panic("xet: HashAt index out of range")
	}
	items := (*[1 << 20]C.XetUploadInfo)(unsafe.Pointer(x.ref30b7a435.items))[:n:n]
	if items[i].hash == nil {
		return ""
	}
	return C.GoString(items[i].hash)
}

// FileSizeAt returns the file size reported for the item at index i.
// It panics if i is out of range.
func (x *Xetuploadresult) FileSizeAt(i int) uint64 {
	n := x.Len()
	if i < 0 || i >= n {
		panic("xet: FileSizeAt index out of range")
	}
	items := (*[1 << 20]C.XetUploadInfo)(unsafe.Pointer(x.ref30b7a435.items))[:n:n]
	return uint64(items[i].file_size)
}

// SHA256At returns the SHA-256 hex digest for the item at index i, or an empty
// string if it was not computed.  It panics if i is out of range.
func (x *Xetuploadresult) SHA256At(i int) string {
	n := x.Len()
	if i < 0 || i >= n {
		panic("xet: SHA256At index out of range")
	}
	items := (*[1 << 20]C.XetUploadInfo)(unsafe.Pointer(x.ref30b7a435.items))[:n:n]
	if items[i].sha256 == nil {
		return ""
	}
	return C.GoString(items[i].sha256)
}

// Err returns the error message from an XetDownloadResult, or an empty string
// when the operation succeeded (i.e. the C error pointer is NULL).
func (x *Xetdownloadresult) Err() string {
	if x == nil || x.ref78b08ee == nil {
		return ""
	}
	if x.ref78b08ee.error == nil {
		return ""
	}
	return C.GoString(x.ref78b08ee.error)
}

// Len returns the number of downloaded paths in the result.
func (x *Xetdownloadresult) Len() int {
	if x == nil || x.ref78b08ee == nil {
		return 0
	}
	return int(x.ref78b08ee.count)
}

// PathAt returns the local destination path for the downloaded file at index i.
// It panics if i is out of range.
func (x *Xetdownloadresult) PathAt(i int) string {
	n := x.Len()
	if i < 0 || i >= n {
		panic("xet: PathAt index out of range")
	}
	ptrs := (*[1 << 20]*C.char)(unsafe.Pointer(x.ref78b08ee.paths))[:n:n]
	if ptrs[i] == nil {
		return ""
	}
	return C.GoString(ptrs[i])
}
