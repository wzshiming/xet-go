# Makefile for xet-go
#
# Targets:
#   rust-build        — Build the xet-sys Rust static library (release).
#   rust-build-debug  — Build the xet-sys Rust static library (debug).
#   copy-libs         — Copy the freshly built library into libs/{os}/{arch}/.
#   generate          — Re-generate CGo bindings using c-for-go.
#   build             — Build the Rust static library, copy to libs/, then build Go packages.
#   test              — Build the Rust static library, copy to libs/, then run all Go tests.
#   clean             — Remove all build artefacts.

.PHONY: all rust-build rust-build-debug copy-libs generate build test clean

CARGO_MANIFEST := xet-sys/Cargo.toml

# Detect the current OS/arch so copy-libs knows where to place the library.
UNAME_S := $(shell uname -s 2>/dev/null || echo Windows)
UNAME_M := $(shell uname -m 2>/dev/null || echo x86_64)

ifeq ($(UNAME_S),Linux)
  LIB_OS := linux
else ifeq ($(UNAME_S),Darwin)
  LIB_OS := darwin
else
  LIB_OS := windows
endif

ifeq ($(UNAME_M),x86_64)
  LIB_ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
  LIB_ARCH := arm64
else ifeq ($(UNAME_M),arm64)
  LIB_ARCH := arm64
else
  LIB_ARCH := amd64
endif

LIB_DEST := libs/$(LIB_OS)/$(LIB_ARCH)

all: build

# ---------------------------------------------------------------------------
# Rust static library
# ---------------------------------------------------------------------------

rust-build:
	cargo build --release --manifest-path $(CARGO_MANIFEST)

rust-build-debug:
	cargo build --manifest-path $(CARGO_MANIFEST)

# ---------------------------------------------------------------------------
# Copy the compiled library into the pre-built libs/ directory
# ---------------------------------------------------------------------------

copy-libs: rust-build
	mkdir -p $(LIB_DEST)
	cp xet-sys/target/release/libxet_sys.a $(LIB_DEST)/libxet_sys.a

# ---------------------------------------------------------------------------
# Regenerate CGo bindings
# ---------------------------------------------------------------------------
generate:
	go run github.com/xlab/c-for-go@latest -ccdefs -ccincl xet.yml

# ---------------------------------------------------------------------------
# Go build / test (depend on the pre-built library being present)
# ---------------------------------------------------------------------------

build: copy-libs
	go build ./...

test: copy-libs
	go test ./...

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

clean:
	cargo clean --manifest-path $(CARGO_MANIFEST)
	rm -f xet/xet.go xet/types.go xet/doc.go \
	      xet/cgo_helpers.go xet/cgo_helpers.h
