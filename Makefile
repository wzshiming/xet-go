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
# Install c-for-go if it is not available:
#   go install github.com/xlab/c-for-go@latest
generate:
	c-for-go -ccdefs -ccincl hf_xet.yml
	@# Move generated files from the nested package dir to hf_xet/
	@if [ -d hf_xet/hf_xet ]; then \
		cp hf_xet/hf_xet/*.go hf_xet/ && \
		cp hf_xet/hf_xet/*.h  hf_xet/ 2>/dev/null || true && \
		rm -rf hf_xet/hf_xet; \
	fi

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
	rm -f hf_xet/hf_xet.go hf_xet/types.go hf_xet/doc.go \
	      hf_xet/cgo_helpers.go hf_xet/cgo_helpers.h
