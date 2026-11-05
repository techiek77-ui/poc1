#!/usr/bin/env bash
# Reproduce the whole POC: corpus -> Tier A -> evaluation (+ Tier B if PII_ORT_LIB and models/tier_b exist).
set -euo pipefail
cd "$(dirname "$0")/.."
go test ./...
go run ./cmd/piigen -n "${N:-30000}" -seed "${SEED:-7}" -out data/corpus.jsonl
go run ./cmd/piitrain -corpus data/corpus.jsonl -out models/tier_a_lr.json -prune "${PRUNE:-0.01}"
if [ -n "${PII_ORT_LIB:-}" ] && [ -f models/tier_b/manifest.json ]; then
  go run -tags onnx ./cmd/piieval -corpus data/corpus.jsonl -model models/tier_a_lr.json -out docs/RESULTS.md -onnx models/tier_b
  go run -tags onnx ./cmd/piibench -onnx models/tier_b -n 300
else
  go run ./cmd/piieval -corpus data/corpus.jsonl -model models/tier_a_lr.json -out docs/RESULTS.md
fi
