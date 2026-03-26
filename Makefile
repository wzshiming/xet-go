# Makefile for xet-go
#
# Targets:
#   rust-build        — Build the xet-sys Rust static library (release).
#   rust-build-debug  — Build the xet-sys Rust static library (debug).
#   generate          — Re-generate CGo bindings using c-for-go.
#   build             — Build the Rust static library then all Go packages.
#   test              — Build the Rust static library then run all Go tests.
#   clean             — Remove all build artefacts.

.PHONY: all rust-build rust-build-debug generate build test clean

CARGO_MANIFEST := xet-sys/Cargo.toml

all: build

# ---------------------------------------------------------------------------
# Rust static library
# ---------------------------------------------------------------------------

rust-build:
	cargo build --release --manifest-path $(CARGO_MANIFEST)

rust-build-debug:
	cargo build --manifest-path $(CARGO_MANIFEST)

# ---------------------------------------------------------------------------
# Regenerate CGo bindings
# ---------------------------------------------------------------------------
generate:
	go run github.com/xlab/c-for-go@latest -ccdefs -ccincl xet.yml

# ---------------------------------------------------------------------------
# Go build / test (depend on the Rust library being present)
# ---------------------------------------------------------------------------

build: rust-build
	go build ./...

test: rust-build
	go test ./...

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

clean:
	cargo clean --manifest-path $(CARGO_MANIFEST)
	rm -f xet/xet.go xet/types.go xet/doc.go \
	      xet/cgo_helpers.go xet/cgo_helpers.h
