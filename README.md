# xet-go

Go bindings for the [xet-core](https://github.com/huggingface/xet-core) HuggingFace Xet storage library.

The low-level CGo bindings in [`xet/`](xet/) are auto-generated from
[`xet.h`](xet/xet.h) using [c-for-go](https://github.com/xlab/c-for-go).

The native implementation lives in [`xet-sys/`](xet-sys/), a Rust crate that
wraps xet-core's async Rust API with a C-compatible static library
(`libxet_sys.a`).  The static library is linked directly into the Go binary
via CGo — no separate shared-library installation is required.

Pre-built static libraries for all supported platforms are committed to the
[`libs/`](libs/) directory, so **no Rust or Cargo installation is needed** to
build Go programs that import this package.

## Features

- **File Upload & Download**: Upload and download files to/from HuggingFace Xet storage
- **Content Hashing**: Compute Xet content-addressable hashes for files
- **Content-Defined Chunking**: Split data into variable-sized chunks
- **XORB Hashing**: Compute XOR-based aggregate hashes for efficient integrity verification

For details on XORB support, see [XORB.md](XORB.md).

## Usage

```sh
go get github.com/wzshiming/xet-go
```

## Supported platforms

| OS      | Architecture |
| ------- | ------------ |
| Linux   | amd64, arm64 |
| macOS   | amd64, arm64 |
| Windows | amd64        |

## License

[MIT](LICENSE)
