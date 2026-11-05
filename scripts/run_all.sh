#!/usr/bin/env bash
# Reproduce the whole POC: corpus -> Tier A -> evaluation (+ Tier B if PII_ORT_LIB and models/tier_b exist).
set -euo pipefail
cd "$(dirname "$0")/.."
go test ./...
go run ./cmd/piigen -n "${N:-30000}" -seed "${SEED:-7}" -out data/corpus.jsonl
go run ./cmd/piitrain -corpus data/corpus.jsonl -out models/tier_a_lr.json -prune "${PRUNE:-0.01}"
if [ -n "${PII_ORT_LIB:-}" ] && [ -f models/tier_b/manifest.json ]; then
  go run ./cmd/piiexport -corpus data/corpus.jsonl -out data/tierb
  if [ "${TRAIN_TIER_B:-0}" = 1 ]; then
    python3 ml/train_tier_b.py --data data/tierb --out models/tier_b --base "${TIER_B_BASE:-scratch}" --arch "${TIER_B_ARCH:-tiny}" --epochs "${TIER_B_EPOCHS:-2}"
  else
    echo "note: Tier B is NOT retrained (set TRAIN_TIER_B=1); evaluating the committed models/tier_b against the regenerated corpus"
  fi
  PII_ONNX_DIR="$PWD/models/tier_b" go test -tags onnx ./pkg/pii/onnx/
  go run -tags onnx ./cmd/piieval -corpus data/corpus.jsonl -model models/tier_a_lr.json -out docs/RESULTS.md -onnx models/tier_b
  go run -tags onnx ./cmd/piibench -onnx models/tier_b -n 300
else
  go run ./cmd/piieval -corpus data/corpus.jsonl -model models/tier_a_lr.json -out docs/RESULTS.md
fi
