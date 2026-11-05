# Implementation review

Self-review of the POC at high effort (eight review angles over `pkg/pii`, `cmd`, `ml`), followed by fixes. Findings are ordered by severity; "fixed" means a code change with a regression test where one made sense.

| # | where | finding | outcome |
|---|---|---|---|
| 1 | `verify/features.go` | `fmt=` feature emitted twice; training deduplicates features per example but scoring summed every entry → train/serve mismatch | **fixed**: duplicate removed and `Model.Score` deduplicates |
| 2 | `labels/labels.go` | `arn no` and `value` listed under two kinds; the phrase index is built from map iteration, so the winning kind was nondeterministic across processes | **fixed**: duplicates removed; `TestPhrasesUniqueAcrossKinds` guards it |
| 3 | `detect/detect.go` `LabelsIn` | linear scan over all document labels per candidate → quadratic on large CSV/log files | **fixed**: labels are sorted by offset; binary search with a bounded phrase length |
| 4 | `detect/names.go` | dictionary-driven name spans started at every capitalised word, producing overlapping sub-spans (`Kumar Singh` inside `Ramesh Kumar Singh`) that double-counted true positives | **fixed**: a span suppresses starts inside it; `TestNoOverlappingDictionaryNames` |
| 5 | `attrib/attrib.go` | any 4+ digit run counted as an "intervening value", so a year between label and value broke attribution | **fixed**: `isValueLike` requires ≥ 6 digits for pure numbers, ≥ 2 digits and ≥ 5 chars for alphanumerics |
| 6 | `detect/csv.go` | rows were split on newlines before quotes were tracked, so quoted multi-line cells shifted every later column | **fixed**: quote state carried across lines; `TestCSVQuotedNewline`. Open: the delimiter is still inferred from the first line only |
| 7 | `verify/linear.go` | thresholds capped at 0.5 because the model is overconfident; the cap treats the symptom | **investigated, kept**: L2 from 1e-6 to 1e-4 and 4 vs 8 epochs changed max \|w\| from 3.7 to 2.8 with identical dev metrics — saturation comes from many agreeing features, not from large weights. Default L2 raised to 1e-5; the real fix is calibration on real labelled windows (Platt/isotonic on the analyst feedback set), tracked in the roadmap |
| 8 | `onnx/onnx.go` `Encode` | truncation dropped the final `[SEP]`, unlike the training-side tokenizer truncation | **fixed**: final token kept when truncating |
| 9 | `detect/detect.go` `embeddedInDigitRun` | only space/dash continuations detected; `7381 9204 5614/1234` passed as an Aadhaar prefix | **fixed**: `/` added (`,` deliberately not: it would break comma-separated values) |
| 10 | `synth/catalog.go` `Pool` | dev and test drew from the same held-out phrasing pool, so test was not stricter than the split thresholds were chosen on | **fixed**: three-way pools (train-only / dev-only / test-only / shared); corpus regenerated. Caveat: the from-scratch Tier B model was trained before this change and may have seen dev-pool phrasings; it never saw test-pool ones |

## Open items (not fixed in the POC)

* `ParseCSVLayout` infers the delimiter from the first line; a header with commas inside quotes could pick the wrong delimiter. Use the product's own parser when integrating.
* `sqlColumnFor` does not handle `''`-escaped quotes or parentheses inside quoted values, and only `INSERT … VALUES` (not `UPDATE … SET`).
* `nameCandidates` is O(labels × words) for label-driven spans; fine for documents, not for multi-megabyte files — index words by offset if it matters.
* `Linear.Score` computes the explanation on every call (sorting contributions); make it lazy in production to roughly halve Tier A cost.
* The Tier A artifact size is dominated by feature strings; feature hashing to 32-bit indices would cut it from ~6 MB to ~1 MB.
* `piiexport` writes the gated/ungated files in map order (harmless, nondeterministic listing).
* The golden set is 48 documents written by the same author as the generator; it is a smoke test, not an independent benchmark. Real labelled windows are the missing piece.
* OCR noise in the generator corrupts digits inside values at a low rate; those spans remain labelled PII and count as baseline misses, which is intended but makes absolute recall numbers slightly pessimistic.

## What was checked and found sound

* Checksums against published vectors and `python-stdnum` (Verhoeff, Luhn, GSTIN).
* Byte-offset integrity of every generated span, including after OCR noise and in documents containing multi-byte characters.
* Determinism of generation for a given seed (timestamps anchored).
* Truth matching: recall counts each labelled span once; duplicate detections of one span do not inflate recall (they did inflate precision until finding 4 was fixed).
* Evaluation leakage: thresholds come from `dev` only; `test` phrasings are unseen by train and dev; the golden set is scored with dev thresholds and never used for tuning except to add generator coverage for whole categories (multi-value labels, `uid` logs, streets named after people), which is the intended loop.
* Train/serve consistency of the ONNX path: identical token ids and probabilities between Python and Go on the smoke model.
