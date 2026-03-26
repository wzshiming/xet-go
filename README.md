# xet-go

Go bindings for the [xet-core](https://github.com/huggingface/xet-core) HuggingFace Xet storage library.

The low-level CGo bindings in [`xet/`](xet/) are auto-generated from
[`xet.h`](xet/xet.h) using [c-for-go](https://github.com/xlab/c-for-go).

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

## License

[MIT](LICENSE)
