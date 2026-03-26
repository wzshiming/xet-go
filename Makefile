# Makefile for xet-go
#
# Targets:
#   generate  — Re-generate CGo bindings using c-for-go.
#   build     — Build all packages.
#   test      — Run all tests.
#   clean     — Remove generated artefacts.

.PHONY: all generate build test clean

all: build

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

build:
	go build ./...

test:
	go test ./...

clean:
	rm -f hf_xet/hf_xet.go hf_xet/types.go hf_xet/doc.go \
	      hf_xet/cgo_helpers.go hf_xet/cgo_helpers.h
