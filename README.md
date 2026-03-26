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
