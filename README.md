# xet-go

Go bindings for the [xet-core](https://github.com/huggingface/xet-core) HuggingFace Xet storage library.

The low-level CGo bindings in [`hf_xet/`](hf_xet/) are auto-generated from
[`hf_xet.h`](hf_xet/hf_xet.h) using [c-for-go](https://github.com/xlab/c-for-go).

## Requirements

| Tool | Version |
|------|---------|
| Go | ≥ 1.21 |
| CGo | enabled (default) |
| hf_xet shared library | ≥ 1.4.x |

The native `hf_xet` shared library (`libhf_xet.so` / `libhf_xet.dylib` /
`hf_xet.dll`) must be built from [xet-core](https://github.com/huggingface/xet-core)
and installed (or otherwise accessible to the linker).

## Building

```bash
# Point CGo at the library when it is not in a standard path:
export CGO_LDFLAGS="-L/path/to/hf_xet/lib -lhf_xet"
export CGO_CFLAGS="-I/path/to/hf_xet/include"

go build ./...
```

## Package layout

```
xet-go/
├── hf_xet/           # CGo bindings package
│   ├── hf_xet.h      # C header defining the xet C API
│   ├── hf_xet.go     # Auto-generated CGo wrappers (c-for-go)
│   ├── cgo_helpers.go # Auto-generated memory-management helpers
│   ├── cgo_helpers.h  # Auto-generated C helper declarations
│   ├── types.go       # Auto-generated Go type definitions
│   ├── doc.go         # Package documentation
│   └── link.go        # CGo linker flags
├── hf_xet.yml        # c-for-go manifest (re-run with `make generate`)
├── Makefile
└── go.mod
```

## API overview

The package mirrors the C API declared in [`hf_xet.h`](hf_xet/hf_xet.h):

| Go function | Description |
|-------------|-------------|
| `UploadFiles(filePaths, count, endpoint, token, sha256s, sha256Count, skipSHA256)` | Upload local files to Xet storage |
| `UploadBytes(buffers, sizes, count, endpoint, token, sha256s, sha256Count, skipSHA256)` | Upload in-memory buffers to Xet storage |
| `HashFiles(filePaths, count)` | Compute Xet content-hashes without uploading |
| `DownloadFiles(files, count, endpoint, token)` | Download files from Xet storage |
| `FreeUploadResult(result)` | Release memory for an upload/hash result |
| `FreeDownloadResult(result)` | Release memory for a download result |

All result structs must be freed with the corresponding `Free*` function.

## Regenerating bindings

Install [c-for-go](https://github.com/xlab/c-for-go) and run:

```bash
go install github.com/xlab/c-for-go@latest
make generate
```

## License

[MIT](LICENSE)
