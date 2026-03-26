// Package xet provides Go CGo bindings for the xet-core HuggingFace Xet
// storage library.
//
// # Linking
//
// Pre-built static libraries for all supported platforms are bundled in the
// libs/ directory of this module.  No Rust or Cargo installation is required
// to build Go programs that import this package.
//
// If you need to rebuild the static library from source (e.g. after modifying
// xet-sys/), run:
//
//	make copy-libs     # builds xet-sys and copies the library into libs/
//
// Or, to only copy an already-built library into libs/:
//
//	cp xet-sys/target/release/libxet_sys.a libs/<os>/<arch>/libxet_sys.a
//
// The CGo flags below prefer the pre-built library in libs/ and fall back to
// the Cargo release build output when the pre-built library is absent.

package xet

/*
// Link the Rust static library.
// ${SRCDIR} is resolved by CGo to the directory containing this file (xet/).
// The pre-built library (libs/) is searched first; the Cargo release output
// (xet-sys/target/release/) is the fallback for local Rust development.
#cgo linux,amd64   LDFLAGS: -L${SRCDIR}/../libs/linux/amd64   -L${SRCDIR}/../xet-sys/target/release -lxet_sys
#cgo linux,arm64   LDFLAGS: -L${SRCDIR}/../libs/linux/arm64   -L${SRCDIR}/../xet-sys/target/release -lxet_sys
#cgo darwin,amd64  LDFLAGS: -L${SRCDIR}/../libs/darwin/amd64  -L${SRCDIR}/../xet-sys/target/release -lxet_sys
#cgo darwin,arm64  LDFLAGS: -L${SRCDIR}/../libs/darwin/arm64  -L${SRCDIR}/../xet-sys/target/release -lxet_sys
#cgo windows,amd64 LDFLAGS: -L${SRCDIR}/../libs/windows/amd64 -L${SRCDIR}/../xet-sys/target/release -lxet_sys

// System libraries required by the Rust runtime and xet-core on each platform.
#cgo linux   LDFLAGS: -ldl -lm -lpthread -lrt
#cgo darwin  LDFLAGS: -framework Security -framework CoreFoundation -framework IOKit -framework SystemConfiguration
#cgo windows LDFLAGS: -lws2_32 -luserenv -lbcrypt -lntdll
*/
import "C"
