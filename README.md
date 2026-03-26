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

A pre-built `libxet_sys.a` for your platform is already included in the
module, so a plain `go build` / `go test` is all that is required.

## Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| Go | ≥ 1.21 | Build Go packages |
| CGo | enabled (default) | Compile the C/Go glue code |
| Rust + Cargo | stable ≥ 1.85 | **Only needed to rebuild `xet-sys/` from source** |

## Rebuilding the native library from source

If you modify the Rust code in `xet-sys/`, rebuild and update `libs/` with:

```sh
make copy-libs   # runs cargo build --release, then copies the .a into libs/
```

The [`update-libs`](.github/workflows/update-libs.yml) CI workflow runs
automatically on every push to `main` that touches `xet-sys/`, building the
static library on all supported platforms and committing the results back to
`libs/`.

## Supported platforms

| OS | Architecture |
|----|-------------|
| Linux | amd64, arm64 |
| macOS | amd64 (Intel), arm64 (Apple Silicon) |
| Windows | amd64 |

## License

[MIT](LICENSE)
