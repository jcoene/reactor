# v8

v8 provides simple v8 bindings for [reactor](https://github.com/jcoene/reactor), packaged with statically compiled node libraries. It's largely extracted from [augustoroman/v8](https://github.com/augustoroman/v8).

## Goals

- Can be installed with a simple `go get`
- Minimal footprint to support [reactor](https://github.com/jcoene/reactor)
- Memory stable (cgo thread issues contained, no leaks)

## Non-Goals

- General purpose Javascript execution

## Supported environments

- Darwin x86_64 (node 6.0.286.54)
- Linux x86_64 (node 6.0.286.54)

## Credits

This work is based off of several existing libraries:

- https://github.com/augustoroman/v8
- https://github.com/fluxio/go-v8
- https://github.com/kingland/go-v8
- https://github.com/mattn/go-v8
