# xet-go

Go bindings for the [xet-core](https://github.com/huggingface/xet-core) HuggingFace Xet storage library.

The low-level CGo bindings in [`hf_xet/`](hf_xet/) are auto-generated from
[`hf_xet.h`](hf_xet/hf_xet.h) using [c-for-go](https://github.com/xlab/c-for-go).

The native implementation lives in [`xet-sys/`](xet-sys/), a Rust crate that
wraps xet-core's async Rust API with a C-compatible static library
(`libxet_sys.a`).  The static library is linked directly into the Go binary
via CGo — no separate shared-library installation is required.

## Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| Go | ≥ 1.21 | Build Go packages |
| CGo | enabled (default) | Compile the C/Go glue code |
| Rust + Cargo | stable ≥ 1.85 | Build the `xet-sys` static library |
| Internet access | — | Cargo downloads xet-core on first build |

## Building

```bash
# 1. Build the Rust static library (one-time or after xet-core updates).
make rust-build

# 2. Build all Go packages (depends on step 1).
make build

# Or do both in one command:
make
```

Alternatively, without Make:

```bash
cargo build --release --manifest-path xet-sys/Cargo.toml
go build ./...
```

## Package layout

```
xet-go/
├── xet-sys/              # Rust crate: statically-linked C API
│   ├── Cargo.toml        #   crate-type = ["staticlib"]
│   └── src/
│       └── lib.rs        #   #[no_mangle] extern "C" wrappers around xet-core
├── hf_xet/               # Go CGo bindings package
│   ├── hf_xet.h          #   C header (C ABI declaration)
│   ├── hf_xet.go         #   Auto-generated CGo wrappers (c-for-go)
│   ├── cgo_helpers.go    #   Auto-generated memory-management helpers
│   ├── cgo_helpers.h     #   Auto-generated C helper declarations
│   ├── types.go          #   Auto-generated Go type definitions
│   ├── doc.go            #   Package documentation
│   └── link.go           #   CGo linker flags (points at xet-sys/target/release/)
├── hf_xet.yml            # c-for-go manifest (re-run with `make generate`)
├── Makefile
└── go.mod
```

## Architecture

```
Go code
  │
  │  import "github.com/wzshiming/xet-go/hf_xet"
  ▼
hf_xet/ (CGo)
  │  auto-generated wrappers call C functions declared in hf_xet.h
  ▼
xet-sys/target/release/libxet_sys.a  (Rust staticlib)
  │  #[no_mangle] extern "C" functions call xet-core's async Rust API
  ▼
xet-core (xet-pkg / xet-data / xet-client / xet-runtime)
```

## API overview

The package mirrors the C API declared in [`hf_xet.h`](hf_xet/hf_xet.h):

| Go function | Description |
|-------------|-------------|
| `UploadFiles(filePaths, count, endpoint, token, sha256s, sha256Count, skipSHA256)` | Upload local files to Xet storage |
| `HashFiles(filePaths, count)` | Compute Xet content-hashes without uploading |
| `DownloadFiles(files, count, endpoint, token)` | Download files from Xet storage |
| `FreeUploadResult(result)` | Release memory for an upload/hash result |
| `FreeDownloadResult(result)` | Release memory for a download result |

All result structs must be freed with the corresponding `Free*` function.

## Regenerating CGo bindings

If `hf_xet.h` changes, regenerate the Go wrappers:

```bash
go install github.com/xlab/c-for-go@latest
make generate
```

## Cleaning

```bash
make clean   # removes xet-sys/target/ and generated Go files
```

## License

[MIT](LICENSE)
