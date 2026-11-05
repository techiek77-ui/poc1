# PII false-positive reduction POC (Go)

A proof of concept for cutting false positives in a regex + checksum + keyword PII
detector for Indian identifiers, without losing true positives. Everything runs
locally (CPU, no external API): a transparent **rule verifier**, a **feature-based
ML verifier with pure-Go inference**, and the plumbing for a **transformer verifier
served through ONNX Runtime from Go**.

The research write-up and recommendation is in [`docs/REPORT.md`](docs/REPORT.md);
the latest measured numbers are in [`docs/RESULTS.md`](docs/RESULTS.md).

## The problem in one line

```
aadhar 738192045614, bank acc 509347182063
```

Both numbers are 12 digits and pass the Verhoeff check, and the keyword `aadhar`
is within 60 characters of both, so the current pipeline reports two Aadhaar
numbers. The second one is a bank account.

## What is here

| path | what |
|---|---|
| `pkg/pii/detect` | the **baseline**: regexes + Verhoeff/Luhn/GSTIN validation + keyword within a 60-char window (or CSV header); gazetteer/pattern candidates for names and addresses |
| `pkg/pii/labels` | catalogue of ~900 label phrases for 18 PII types and 20 competing identifier families, with a tokenizer-based matcher (handles `aadhaar_no`, `aadhaarNo`, OCR `Acc0unt N0`) |
| `pkg/pii/attrib` | **context analysis**: which label governs a value (CSV header, SQL column, nearest left label with nothing in between, tightly attached right label), competing labels, number shape, test-context |
| `pkg/pii/verify` | `Rules` (configuration 1) and the `Linear` model (Tier A): features, pure-Go logistic-regression training and inference, explanations for the log |
| `pkg/pii/onnx` | Tier B runtime: ONNX Runtime session + HuggingFace tokenizer in Go (`-tags onnx`) |
| `pkg/pii/synth` | synthetic Indian-context corpus generator: prose, forms, CSV, JSON, logs, code, OCR, identity cards; hard negatives that pass the Verhoeff check |
| `pkg/pii/eval` | evaluation: TP/FP matching, operating points at a TP-retention target, breakdowns, markdown |
| `testdata/golden.txt` | 48 hand-written documents in styles the generator does not produce |
| `ml/` | Python: Tier B training/export (`train_tier_b.py`), smoke model for the Go↔ONNX parity test |
| `cmd/piigen`, `cmd/piitrain`, `cmd/piieval`, `cmd/piiscan`, `cmd/piiexport` | CLIs |

## Run it

```bash
go test ./...
```

```bash
go run ./cmd/piigen -n 30000 -seed 7 -out data/corpus.jsonl
```

```bash
go run ./cmd/piitrain -corpus data/corpus.jsonl -out models/tier_a_lr.json
```

```bash
go run ./cmd/piieval -corpus data/corpus.jsonl -model models/tier_a_lr.json -out docs/RESULTS.md
```

Try your own text:

```bash
echo 'aadhar 738192045614, bank acc 509347182063' | go run ./cmd/piiscan
```

```bash
go run ./cmd/piiscan -format csv customers.csv
```

## Tier B (transformer via ONNX Runtime)

```bash
go run ./cmd/piiexport -corpus data/corpus.jsonl -out data/tierb
```

```bash
python3 ml/train_tier_b.py --data data/tierb --out models/tier_b --base scratch --arch tiny --epochs 2
```

With access to huggingface.co, replace `--base scratch --arch tiny` by a pretrained
encoder, e.g. `--base microsoft/MiniLM-L12-H384-uncased`.

```bash
PII_ORT_LIB=/path/to/libonnxruntime.so go run -tags onnx ./cmd/piieval -onnx models/tier_b
```

The Go runtime needs the ONNX Runtime shared library (≈27 MB on Linux x64, from the
official GitHub releases) plus `model.int8.onnx`, `tokenizer.json` and
`manifest.json`; the model is data, not code, so a retrained model is a file push.

## Model lifecycle

`piitrain` writes a versioned artifact: weights, per-entity thresholds and a
manifest (feature-schema version, training-data hash, dev metrics). The Go loader
refuses an artifact whose feature schema it does not understand. Promotion is
gated by `piieval` (TP retention ≥ 99 % on the frozen regression split and the
golden set).
