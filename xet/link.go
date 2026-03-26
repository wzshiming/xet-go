// Package xet provides Go CGo bindings for the xet-core HuggingFace Xet
// storage library.
//
// # Linking
//
// The native implementation is compiled from the Rust crate in xet-sys/ into
// a static library (libxet_sys.a).  Build it before using this package:
//
//	make rust-build          # release build (recommended)
//	make rust-build-debug    # debug build
//
// Or directly with Cargo:
//
//	cargo build --release --manifest-path xet-sys/Cargo.toml
//
// The CGo flags below link against the release artefact.  To use a debug
// build, set CGO_LDFLAGS to point at the debug output directory instead:
//
//	export CGO_LDFLAGS="-L$(pwd)/xet-sys/target/debug -lxet_sys"

package xet

/*
// Link the Rust static library.
// ${SRCDIR} is resolved by CGo to the directory containing this file (xet/).
#cgo LDFLAGS: -L${SRCDIR}/../xet-sys/target/release -lxet_sys

// System libraries required by the Rust runtime and xet-core on each platform.
#cgo linux   LDFLAGS: -ldl -lm -lpthread -lrt
#cgo darwin  LDFLAGS: -framework Security -framework CoreFoundation -framework IOKit -framework SystemConfiguration
#cgo windows LDFLAGS: -lws2_32 -luserenv -lbcrypt -lntdll
*/
import "C"
