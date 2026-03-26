package hf_xet

/*
#include "hf_xet.h"
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
		panic("hf_xet: HashAt index out of range")
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
		panic("hf_xet: FileSizeAt index out of range")
	}
	items := (*[1 << 20]C.XetUploadInfo)(unsafe.Pointer(x.ref30b7a435.items))[:n:n]
	return uint64(items[i].file_size)
}
