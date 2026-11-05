#!/usr/bin/env bash
# Build the pure-Go scanner (rules + Tier A) for every endpoint platform, and the
# ONNX-enabled binaries where a C toolchain for the target exists.
set -euo pipefail
cd "$(dirname "$0")/.."
OUT=${OUT:-dist}; mkdir -p "$OUT"
for t in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
  os=${t%/*}; arch=${t#*/}; ext=""; [ "$os" = windows ] && ext=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags="-s -w" -o "$OUT/piiscan-$os-$arch$ext" ./cmd/piiscan
  echo "built $OUT/piiscan-$os-$arch$ext"
done
# ONNX tier: native build on each platform (CGO). Cross-compiling needs the target's C toolchain:
#   Windows from Linux:  CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -tags onnx ./cmd/piiscan
#   macOS from Linux:    osxcross (CC=o64-clang) or build on a Mac runner
if [ "${ONNX:-0}" = 1 ]; then
  CGO_ENABLED=1 go build -tags onnx -trimpath -ldflags="-s -w" -o "$OUT/piiscan-onnx-$(go env GOOS)-$(go env GOARCH)" ./cmd/piiscan
  echo "built ONNX-enabled binary for the host platform"
fi
