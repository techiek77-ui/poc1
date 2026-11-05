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


## Round 2 — independent reviewers (unanchored), high effort

Three reviewer agents with narrow scopes read the code without seeing the first review, verified findings with scratch programs, and reported. Everything marked **fixed** has a regression test unless noted.

### Detection / attribution / labels

| # | where | finding | outcome |
|---|---|---|---|
| A1 | `labels` | lazily built phrase index had no synchronisation: concurrent first use is a data race / `concurrent map writes` crash (reproduced with `-race`) | **fixed**: `sync.Once` |
| A2 | `attrib.sqlColumnFor` | `\'`-escaped quotes desynchronised the column walk; multi-row `VALUES (…), (…)` lost mapping after the first tuple — a real Aadhaar in row 2 scored "no label" | **fixed**: escape handling, per-tuple column reset; tests |
| A3 | `attrib` value regex | adjacent numbers separated by ` / ` or ` - ` merged into one span and spans touching the candidate were dropped, so intervening values went unseen and a competing label governed ("Ref No 1234567 / 7381… is my Aadhaar") | **fixed**: single-separator groups only; overlapping spans clipped and re-tested |
| A4 | `attrib.sepClass` | the "verb" class ("7381… is my Aadhaar") was unreachable: "is"/"as" were stripped before being tested | **fixed**; test |
| A5 | `detect/csv.go` | any `"` toggled quote state, so one stray quote (`5'11" tall`) swallowed the rest of the file into a single cell; delimiter counted inside quotes | **fixed**: quotes only open at a cell start / close before a delimiter, `""` escapes, delimiter chosen outside quotes over several lines; tests |
| A6 | `labels` + `detect.LabelsIn` | phrases matched across any gap/line break (`Account` + 124 bytes of rules + `Number`), producing hits longer than the search bound, so `LabelsIn` missed them | **fixed**: tokens of a phrase must be ≤ 3 bytes apart with no line break; `LabelsIn` bound is the measured longest hit |
| A7 | `detect/names.go` | PIN-anchored address walk-back cut the span one or two bytes late after a `:` label (`"lat 302, MG Road…"`) | **fixed**; test |
| A8 | `detect/names.go` | names starting with an initial (`A. Kumar`, `R. K. Sharma`) were never proposed; sentence-final names kept `.`/`'s` | **fixed**; tests |
| A9 | `attrib.Analyze` | line statistics copied and scanned the rest of the document per candidate (quadratic; 16k candidates on one line → 9.7 s) | **fixed**: bounded 512-byte scan, no copy |
| A10 | `detect` | label-driven name lookup and `CSVLayout.ColumnAt` were linear scans per candidate | **fixed**: binary search |
| A11 | `detect` UPI | the validator could never fail, so `care@axisbank.com` was also reported as a UPI id | **fixed**: context check rejects `@handle.tld`; test |
| A12 | `detect.embeddedInDigitRun` | any single digit before a separator vetoed a mobile (`Sl No 10 9876543210`) | **fixed**: the neighbouring group must have ≥ 3 digits; test |
| A13 (minor) | `labels.ocrNorm` | 1-letter+1-digit tokens (`D1`, `R5`) were "OCR-corrected" into labels | **fixed**: ≥ 2 letters, except the `N0` case |

### Evaluation / data / ML pipeline

| # | where | finding | outcome |
|---|---|---|---|
| B1 | generator + `eval.match` | addresses were one span, so the **PIN code inside every address was labelled non-PII**: 291 of the 339 "PINCODE false positives" were real PINs and the model was trained (5 708 negatives vs 2 732 positives) to suppress them; counting them as PII, overall retention was 98.5 %, not 99.6 % | **fixed**: generator emits a nested PINCODE span inside every address and sets `nested: true`; `match` prefers a same-type span covering the detection; in corpora without nested labels such detections are *unknown*, not negatives; corpus regenerated, model retrained, numbers re-reported |
| B2 | `piitrain`/`piieval` | different `-real-split` defaults (train vs test) and no manifest check: the documented workflow evaluated on records the model had trained on | **fixed**: `piieval` defaults to the manifest's mode and prints a note when an input file's hash appears in the training manifest |
| B3 | `corpus.AssignSplits` | hash mode split by per-record id, putting decisions about the same snippet in train and test | **fixed**: partially labelled documents hash by snippet text; test |
| B4 | `piiconvert` | default `-unmapped nonpii` turned forgotten PII labels into negatives, silently and at 3× weight; typos in `-unmapped`/`-split`/`-format` accepted | **fixed**: unmapped labels are an error unless `-unmapped` is given; `drop` marks documents partial; flags validated |
| B5 | loaders | unknown entity types (`AADHAR`), formats (`CSV`), splits (`Test`) and split modes (`trian`) were accepted and silently distorted counts; `value` vs `start/end` not cross-checked | **fixed**: validated with errors; test |
| B6 | `eval.match` | "half of the shorter span" admitted fragment-grabs (a 139-byte CSV-row slice as an ADDRESS TP) and nested same-type spans lost to a larger enclosing span | **fixed**: TP requires ≥ 50 % coverage both ways with same-type spans searched first; span-level retention reported next to detection retention; test |
| B7 | `verify.Load` | an empty artifact loaded and suppressed everything; thresholds unchecked; training window not recorded | **fixed**: degenerate artifacts and out-of-range thresholds refused; `window` in the manifest and checked by `piieval`/`piiscan` |
| B8 | `piiconvert` offsets | byte/rune decided per document; invalid UTF-8 advanced 3 bytes per bad byte | **fixed**: decided once per file, verified against span text when available, `utf8.DecodeRuneInString`; tests |
| B9 | `piiconvert` | write → re-read → truncate → rewrite with every error ignored | **fixed**: single write, errors checked |
| B10 | `piieval` | tables used thresholds recomputed from dev while the "keep=" column used the artifact's | **fixed**: headline row uses shipped thresholds, re-tuned thresholds are a separate row |
| B11 (minor) | `verify.Train`, `eval.exampleLine` | `MinCount` counted repeats within one example; examples cut inside multi-byte characters | **fixed** |

### Generator / ONNX runtime / Python / packaging

| # | where | finding | outcome |
|---|---|---|---|
| C1 | `Dockerfile`, `cmd/piiscan` | the ONNX image had no Tier B entrypoint (`piiscan` never linked the ONNX package) and a plain `docker build` produced the glibc stage, not the distroless one | **fixed**: `piiscan -onnx <dir>` (same build-tag pattern as `piieval`); final stage selected by `--build-arg RUNTIME=…`, static by default; README/crossbuild wording corrected |
| C2 | `ml/train_tier_b.py`, `onnx.New` | manifest `inputs` came from the tokenizer, not the exported graph; a model that ignores `token_type_ids` (DeBERTa) loaded in Go and silently scored every candidate 0.5 — the "B onnx" row would have read 100 % retention / 0 % reduction | **fixed**: inputs read from the ONNX graph; Go refuses a manifest that disagrees with the graph; verifier errors are counted and `piieval` prints a warning instead of a flattering row |
| C3 | tokenizer normalizer | uncased BERT/MiniLM ship `strip_accents: null`, which the Go tokenizer reads as false; with `true` it panicked on `İ`/`ẞ` | **fixed**: both sides use an explicit `BertNormalizer`; the Go runtime applies the same NFD + strip-marks + lowercase before tokenising (idempotent), so the problematic code points never reach it; adversarial strings are in the parity references |
| C4 | tokenizer normalizer | standalone `StripAccents` strips different marks than Go on Devanagari, so identity-card windows tokenised differently | **fixed** by the same explicit normalizer (verified identical tokens on Devanagari references) |
| C5 | parity gate | `train_tier_b.py` never wrote `reference.json`, so the Go/Python parity test passed vacuously for real models; literal `[SEP]` in text tokenised differently | **fixed**: training writes references from the int8 model over 32 dev windows plus adversarial strings; the test fails rather than skips when `PII_ONNX_DIR` is set; square brackets are neutralised in `onnxinput.Clean`, applied on both sides |
| C6 | `synth.Pool` | pools keyed by exact string, so case/punctuation variants of one phrasing leaked across splits and several fields had no held-out phrasing | **fixed**: pools over normalised word sequences, assigned per field in a fixed rotation that guarantees a dev-only and a test-only phrasing for every PII field (`TestLabelPoolCoverage`); labels added where a field had too few |
| C7 | threshold selection | the 1 % dev quantile did not deliver 99 % retention on test for 5 of 18 types | **fixed**: one-standard-deviation margin and `min_n` 100 in Go and Python; the 0.5 cap still applies |
| C8 | `scripts/run_all.sh` | died at the Tier B step in a fresh checkout (no `piiexport`), and never retrained Tier B | **fixed**: exports first, runs the parity test, retrains with `TRAIN_TIER_B=1`, says so otherwise |
| C9 | `models/tier_b` | trained on an earlier corpus (mid-rune windows) and evaluated as fp32 while int8 ships | **fixed**: retrained on the final corpus; dev/test metrics, thresholds and references come from the int8 session; manifest records the training-data hash |
| C10 | templates | labels were almost never separated from their values, so attribution was barely exercised | **fixed**: aligned header/value rows, slash and comma pairs, value-before-label layouts (≈20 % of multi-field prose, 1 in 7 forms) — and the attribution gained column-aligned header lookup for such text |
| C11 | runtime hygiene | unknown `PII_ORT_EP` fell back to CPU silently; `Init` cached a failure forever; `PII_ORT_THREADS` errors ignored; attention mask ignored; `piibench` divided by zero on an empty file | **fixed** |
| C12 | generator tests | did not compare spans across runs, check span text against values, or cover `card`; pool coverage was logged, not asserted | **fixed**: `TestSpansMatchValues` (pre-noise exact, post-noise OCR image), truth compared across runs, pool coverage asserted |

### Follow-ups found while re-measuring

* Address candidates (my own baseline component, not the user's regex pipeline) were the largest remaining false-positive source after the above: the PIN-anchored walk-back accepted any segment containing a digit (so it swallowed UPI handles and account numbers), label-driven capture ran past the PIN code into the next sentence, and a header-row `Address` label captured the value row beneath it. All three are fixed with tests; an address now needs a real cue word, stops at its PIN code (+ state), and header rows never introduce value rows.
* New signals: a place word at the end of a NAME span ("Sudha Estate", "Kavita Sharma Road"), dummy-shape features (`seqrun`, `repgroup`, `dummy`), and `id_type / id_number` CSV column pairs and date-valued non-birth labels ("Valid Till 01-10-2038") in the generator so the model meets them in training.

### Effect on the numbers

The honest corpus (nested PIN labels, held-out phrasings by normalised form, separated-label layouts) is harder: Tier A went from a reported 97.9 % to **93.0 % FP reduction at 99.4 % retention** (precision 89.0 % → 99.1 %), rules from 71.8 % to 76.8 % (the rules benefited from the attribution fixes), and the golden set is unchanged in spirit (rules 86 % at 100 % retention, linear 86 % at 98.6 %). The earlier figure was inflated by the labelling flaw in B1 and by layouts that never separated a label from its value. `docs/RESULTS.md` is regenerated from the current code.

## Open items from round 1 (status)

* `ParseCSVLayout` now counts delimiters outside quotes over several lines (round 2); still prefer the product's own parser when integrating.
* `sqlColumnFor` now handles backslash-escaped quotes and multi-row `VALUES` (round 2); `UPDATE … SET` is still not mapped.
* `nameCandidates` label-driven lookup is now a binary search (round 2).
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
