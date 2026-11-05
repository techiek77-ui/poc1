# Reducing false positives in regex-based PII detection with a local ML verifier

**Scope.** Indian PII (Aadhaar, PAN, GSTIN, payment cards, voter ID, passport, driving licence, IFSC, mobile, email, UPI, bank account, UAN, PIN code, DOB, names, addresses) detected by a Go product that runs on CPU-only servers, GPU servers and endpoint agents, with no external API. Current pipeline: regex → checksum/structure validation → a same-type keyword within 60 characters (or the CSV/Excel column header). English text.

**Question.** Can an ML model cut the false positives this pipeline produces, without losing true positives?

**Answer.** Yes, and it should be added as a *verifier* behind the existing detector rather than as a replacement for it. On this POC's held-out data the verifier removes **97–98 % of false positives while keeping 99.6 % of true positives** (precision 87.7 % → 99.7 %); on a hand-written set of 48 documents in styles it was never trained on it removes **94 %** of false positives at 98.6 % retention, and a rules-only version of the same idea removes 86 % at 100 % retention. It costs about 70 µs per candidate in pure Go, runs on endpoints, and ships as a data file that you can retrain and push independently of the binary. The numbers come from synthetic data and must be re-validated on real scans (section 9); the design is what carries over.

Measured tables: [`RESULTS.md`](RESULTS.md). Code map and how to run: [`../README.md`](../README.md).

---

## 1. Why the keyword window produces false positives

Every false positive in the brief's example and in the data falls into one of six buckets. They matter because each needs a different signal, and because most of them are *not* fixable by a tighter checksum.

| bucket | example | why the current gate passes it |
|---|---|---|
| **Attribution** — the keyword belongs to a neighbouring value | `aadhar 7381…, bank acc 5093…` | the window is symmetric and does not ask *which* value the word `aadhar` labels |
| **Shape collision** — another identifier has the same digit shape | UAN, EPFO PPO, NPS PRAN, APAAR, UPI/IMPS RRN, 12-digit bank accounts, mobiles written as `91XXXXXXXXXX` | a Verhoeff check passes ~10 % of random 12-digit strings (≈8 % after the first-digit ≠ 0/1 rule); with millions of account and reference numbers in scanned files, 8 % is a lot |
| **Keyword present, value absent** | `Aadhaar seeding pending. Txn ref 6508…` | the word is about a process, the number is a reference |
| **Structured layouts** | CSV with an ambiguous header, JSON key 60 chars away, SQL `INSERT … VALUES`, log `k=v` pairs | the governing label is the header/key/column, not whatever is within 60 characters |
| **Test and dummy data** | `TEST_AADHAAR = "2222 2222 2227"` | repeated/sequential digits with a valid check digit |
| **Text PII** (names, addresses) | `Mohammed Ali Road`, `Dear Parent` | capitalised-word heuristics have no notion of what a name is |

Names and addresses are a different problem from the identifiers: there is no regex to verify, so the question is *detection* (NER), not verification. The POC treats them with a gazetteer/pattern candidate generator plus the same verifier, and section 10 recommends a proper NER model as the upgrade.

## 2. What the field does (research summary)

A background scan of vendor documentation, open-source code and papers (full notes with links in section 12):

* **No DLP vendor documents label-to-value attribution.** Microsoft Presidio boosts a match if a context word appears in the 5 lemmatised tokens *before* it and, importantly, scores any checksum-valid Aadhaar at 1.0 with or without a keyword; it keeps overlapping results of different types. Google Sensitive Data Protection uses hotword proximity windows and *exclusion rules* (drop a finding that overlaps another infoType, or that has a negative hotword nearby) — the only documented lever for the two-numbers case is negative evidence. AWS Macie requires the keyword to precede the match within 50 characters (default). Microsoft Purview uses a 300-character proximity. Nightfall uses 40 bytes either side. All of them would flag both numbers in the brief's example.
* **The closest production precedent is GitHub secret scanning.** Regex candidates are re-scored by a small fine-tuned encoder (ModernBERT) that reads the file path, the credential type, variable names and nearby text. It replaced an LLM verification step that had cut customer-confirmed false positives by ~76 %; the fine-tuned classifier beat the LLM on both precision (+13 %) and recall (+19 %) at under 2 ms per candidate set and >98 % lower serving cost. Cloudflare DLP's "AI context analysis" is the same idea with an embedding nearest-neighbour over reported true/false positives.
* **Second-stage classifiers on regex candidates work with small models.** Saha et al. (2020) reached 84 % precision / 89 % recall on secret candidates with a logistic-regression/NB/SVM ensemble over context features; RECAP (2025) keeps a regex PII match only if a verifier confirms it, explicitly to remove "numeric coincidences".
* **Open PII span models exist but do not cover Indian identifiers** (GLiNER2-PII, OpenAI Privacy Filter, Piiranha, ai4privacy) or are trained on private synthetic data (the Indian ones: chitragupta-pii-indic, sakshi-pii-in). The authors of the better Indian models explicitly keep regex + checksum for IDs and use the model for names/addresses. No public corpus with Aadhaar/PAN/GSTIN in realistic context exists; Faker's `aadhaar_id()` appends a *Luhn* digit, not Verhoeff, so it fails a correct validator 90 % of the time.
* **LLM-as-verifier** is accurate but not an option here (local-only, endpoints), and the evidence shows over-flagging and prompt-injection exposure when it reads scanned content. Where it has been used on *candidates only* (RECAP, GitHub) it was later replaced by a small fine-tuned model.
* **CPU cost of small encoders**: single-digit milliseconds per 128-token input for a 6-layer int8 model on a modern x86 core is the consistent expectation; measured here in section 7.

Conclusion from the research: the right shape is *regex for recall, a learned context verifier for precision*, with structured context (headers, keys) and negative evidence (competing labels) as first-class inputs. That is what was built.

## 3. Recommended architecture

```
file ──► regex + checksum ──► candidates ──► context analysis ──► verifier ──► score + reason ──► policy
         (unchanged)          (type, span)   (attrib)             (rules | Tier A | Tier B)      (threshold per type, log)
```

1. **Keep the detector.** Regex + checksum has the recall; nothing downstream can find what it misses. Only its *keyword gate* becomes optional (section 6.3).
2. **Context analysis (`pkg/pii/attrib`)** is format-aware and shared by every verifier:
   * the **governing label** of a value: the CSV header or SQL column if it carries a label; else the nearest label to the left with *no other value between it and the candidate*; else a label tightly attached on the right (`7381… (Aadhaar)`, `… is my Aadhaar`); a weak/generic word (`No.`, `card`, `uid`) immediately left is skipped in favour of a specific label that precedes it (`Permanent Account Number Card ABCPE1234F`);
   * every other label in the window, by side and distance, with the kind it belongs to — including ~20 families of *non-PII* identifiers (order, transaction/RRN/UTR, invoice, AWB, customer/CIF, policy, ticket, PRAN, PPO, APAAR, consumer number, device/IMEI, PNR, timestamp, amount, cheque, vehicle, enrolment ID, user id) whose labels are the negative evidence;
   * number shape (grouping, first digit, repeated/sequential/repeating-group digits, `91`+mobile shape, Luhn/Verhoeff), test/sample wording, line statistics, and for names/addresses dictionary hits, titles, address cues, PIN, the word after the span.
3. **Verifier tiers.** Same interface, pick per deployment:
   * **Rules** — no model, ~1 µs: suppress only when a competing label governs or the digits are dummy. Fixes the brief's example outright.
   * **Tier A, feature model (pure Go)** — logistic regression over ~200 k sparse features derived from the analysis (label identities × claimed type, separators, distances, tokens and character trigrams around the value, shape, structured keys). ~40 µs per candidate, 6 MB artifact, trains in 10 s, explains every decision in the log.
   * **Tier B, transformer (ONNX Runtime from Go)** — a small encoder reading `type | key | left « value » right`. The runtime path is built and verified; with a pretrained encoder it is the ceiling, at 3–10 ms per candidate on one core (section 7). Blocked in this environment only by network access to the pretrained weights.
4. **Policy, not model, decides.** Each verifier returns a probability and a reason. Thresholds are per entity type, chosen on held-out data to keep ≥ 99 % of true positives, and **capped at 0.5**: a candidate the model considers more likely than not is never dropped. Suppressed detections are logged with the reason, which is also the raw material for retraining (section 8).

## 4. What was built

| component | status |
|---|---|
| Baseline reproduction (18 entity types, 60-char window, CSV headers) | done, tested; faithful to the brief, regex/keyword lists are data |
| Label catalogue (~900 phrases, 38 kinds, tokenizer-based matcher handling `aadhaar_no`, `aadhaarNo`, OCR `Acc0unt N0`) | done |
| Synthetic corpus generator (prose, forms, CSV, JSON, logs, code, OCR, identity cards; coherent personas; hard negatives) | done, deterministic |
| Context analysis + rule verifier | done, tested on the canonical case |
| Tier A: features, pure-Go training, inference, explanations, versioned artifact | done |
| Evaluation harness (TP retention, FP reduction, per type/shape/category, sweep, gate experiment, latency) | done |
| Golden set (48 hand-written documents) | done |
| Tier B: Go runtime (ONNX Runtime + HF tokenizer), Python training/export/quantisation, Go/Python parity test, latency benchmark | runtime done and verified; model trained from scratch only (no pretrained weights reachable) |

### 4.1 The synthetic corpus, and why it is adversarial

30 000 documents, 236 k labelled spans, split 70/10/20 by document. Label phrasings and sentence templates are *pooled* by hash so that the dev/test splits contain wordings never seen in training. Every PII type appears in every data shape. Distractors are generated with their own realistic labels, next to PII, and **35 % of non-Aadhaar 12-digit values are forced to pass the Verhoeff check with a leading 2–9** (the real rate is ~8 %, the higher rate makes the test harder). The brief's example is a template. Masked Aadhaar, test fixtures with dummy numbers, headerless CSVs, JSON with numeric values, `uid=` meaning user-id in logs, two values under one label, streets named after people and identity cards that do not print the word "Aadhaar" are all included because the first golden-set run showed they were missing.

### 4.2 Evaluation protocol

A detection is a (span, claimed type). It is a true positive if a labelled span of that type covers at least half of it; otherwise a false positive, bucketed by what the span really was (`bank_account`, `rrn`, `ppo`, `dummy`, `no_truth`…). Verifiers only re-score the baseline's detections, so **TP retention** (kept TPs / baseline TPs) and **FP reduction** are the two numbers that matter; recall against all labelled spans is reported separately. Thresholds are chosen on dev, results reported on test, and the golden set is scored with the same thresholds.

## 5. Results

Numbers below are from the final run in [`RESULTS.md`](RESULTS.md) (6 000 test documents, 30 271 baseline detections).

### 5.1 Held-out synthetic test split

| configuration | TP | FP | precision | TP retention | FP reduction |
|---|---:|---:|---:|---:|---:|
| 0 baseline (regex + checksum + keyword ≤ 60 chars) | 26 544 | 3 727 | 87.7 % | 100 % | – |
| 1 rules @0.5 | 25 917 | 1 077 | 96.0 % | 97.6 % | 71.1 % |
| 1 rules @99 % retention | 26 485 | 1 423 | 94.9 % | 99.8 % | 61.8 % |
| **A linear (pure Go)** | **26 430** | **92** | **99.7 %** | **99.6 %** | **97.5 %** |

FP reduction at other retention targets (thresholds from dev, measured on test): linear 93.1 % at 99.9 % retention, 96.5 % at 99.5 %, 97.5 % at 99.0 %; rules 47.7 % / 61.5 % / 61.8 %.

By data shape the linear verifier removes 95–99 % of false positives everywhere except code fixtures (dummy numbers labelled as Aadhaar remain the largest residual category). By entity type the biggest absolute wins are BANK_ACCOUNT (the 12-digit collisions), AADHAAR, UAN and PINCODE; the full per-type table is in `RESULTS.md`.

### 5.2 Golden set — 48 documents in styles the generator does not produce

Bank SMS, UPI notifications, WhatsApp-style chat, an HR email, logs where `uid` is a user id, EPFO/NPS/PPO/LPG/electricity documents, an ITR acknowledgement, a GST invoice, a shipment notification, a loan letter, passport/DL forms, a voter list, credit-card statements, SQL dumps, test fixtures, identity cards with OCR noise, a passbook, headerless CSVs, mobiles written with the `91` prefix, and the brief's example in both orders.

| configuration | TP | FP | precision | TP retention | FP reduction |
|---|---:|---:|---:|---:|---:|
| 0 baseline | 147 | 52 | 73.9 % | 100 % | – |
| 1 rules @0.5 | 147 | 7 | 95.5 % | **100 %** | 86.5 % |
| A linear | 145 | 3 | 98.0 % | 98.6 % | **94.2 %** |

Every remaining error is listed in `RESULTS.md`. The two true positives the linear model drops are an Aadhaar in a SQL `INSERT` (score 0.29 — structured-column mapping is new to the model) and a mobile written `919876543216` with no label of its own; the remaining false positives are test-fixture dummies (`TEST_PAN`, `TEST_ACCOUNT`) and `9999999999` with a `mobile` label. The rules keep every true positive on this set and are wrong only where the text is genuinely ambiguous (`number` as a column header; `uid=` in a login log).

### 5.3 What the ML adds beyond rules

The rules get ~62–71 % of the false positives on synthetic data and 86 % on the golden set because attribution plus competing labels *is* most of the problem. The model adds the cases rules cannot express: generic or missing headers combined with other evidence (`id_number` next to an `acct` column), the long tail of context words (`login`, `password`, `test`, `ref` two tokens away), interactions between shape and claim (12 contiguous digits claimed as a bank account next to an Aadhaar label), and calibrated confidence that lets you trade retention for reduction per entity type (the sweep). The model is also where real-world feedback can go; rules only grow by hand.

Where the model is weaker than rules: under distribution shift it is over-confident (the first golden run dropped 7 % of true positives because thresholds learned on synthetic data sat at 0.99; capping thresholds at 0.5 fixed it) and it has no notion it has not seen (SQL columns before the attribution step learned them). Run both: rules as the floor, model on top, and log disagreements.

## 6. Three findings worth acting on regardless of ML

1. **Attribution alone fixes the brief's example** — the nearest label on the left with no value in between, plus "a label followed by its own value does not also label the previous value". This is a 200-line change to the existing gate.
2. **Competing-label suppression needs a catalogue of the *other* identifiers.** The product knows what Aadhaar keywords look like; it also needs to know what `UTR`, `RRN`, `AWB`, `PPO`, `PRAN`, `APAAR`, `CIF`, `consumer no`, `IMEI` look like. That list is in `pkg/pii/labels` and is the single most reusable artifact of this POC.
3. **The keyword gate costs recall.** 17 % of labelled PII in the synthetic test set is never proposed because no keyword is within 60 characters (identity-card text, bare table cells, headerless exports). Replacing the gate with the model's score (score every regex + checksum match) raised recall from 83.0 % to 95.1 % at a precision *better* than today's baseline (93.1 % vs 87.7 %) — see "Keyword gate vs. model gate" in `RESULTS.md`. On the golden set: 94.0 % → 96.7 %. That is a product decision (it changes what gets reported), but it is the single largest recall lever available.

Two smaller ones: Aadhaar numbers never start with 0 or 1, which already excludes UAN (1…) and PRAN (11…) shapes; and `uid` should be a weak keyword — in logs it means user id.

## 7. Deployment in Go

**Tier A (pure Go, no CGO).** `models/tier_a_lr.json` is a 6 MB JSON artifact (weights, per-entity thresholds, manifest); it loads in well under a second and needs ~25 MB of heap. The size is the feature *strings*, not the weights: hashing features to 32-bit indices and writing a binary weight table would bring it to about 1 MB with no accuracy change if endpoint size matters. Per candidate on this container (Xeon 2.1 GHz, one goroutine): context analysis 25 µs, rules 0.6 µs, linear model 41 µs including the explanation string. At 1 000 candidates per second per core it is irrelevant next to file I/O and regex scanning. Suitable for endpoint agents as-is.

**Tier B (ONNX Runtime).** Ships as the ONNX Runtime shared library (`libonnxruntime.so` 28 MB for 1.31.1 Linux x64; `onnxruntime.dll` on Windows; the Go binding loads it at runtime; the binding needs CGO on every platform (MinGW on Windows); binding and runtime versions must be paired — `onnxruntime_go` v1.38 needs ONNX Runtime ≥ 1.31), an int8 `.onnx` model, `tokenizer.json` and `manifest.json`. GPU servers would add the CUDA execution provider; CPU needs nothing else. Measured from Go on this container with a single intra-op thread, int8, mean input 55 tokens:

| model shape | parameters | int8 file | per candidate (Go, 1 thread) |
|---|---:|---:|---:|
| tiny (4 layers × 256) — the from-scratch model | 5.4 M | ~6 MB | see `RESULTS.md` |
| MiniLM-L6 shape (6 × 384) | ~14 M with 8 k vocab (22 M with a 30 k vocab) | 14 MB | ~10 ms mean while sharing the CPU with training; re-measured in `RESULTS.md` |
| BERT-base shape (12 × 768) | ~86 M | 92 MB | ~47 ms |

For endpoints the question is candidates per minute: at 10 ms per candidate, 1 000 candidates per minute costs 17 % of one core. Post-regex candidates are rare relative to text volume, so a MiniLM-sized model is viable on endpoints if candidate rates are modest, and comfortably so on servers; BERT-base is a server-only option. Tier A everywhere plus Tier B on servers is the configuration to start with.

## 8. Model lifecycle: retraining and shipping upgrades

You asked whether you should be able to retrain the models and ship upgrades. Yes, and the POC is built around it:

* **The model is a data file, never code.** Tier A is JSON; Tier B is `.onnx` + `tokenizer.json`. The binary loads them at start-up (or on a reload signal). A retrained model is a file push through whatever channel you already use for signature/keyword updates — no release, no endpoint upgrade.
* **Versioned artifact with a manifest**: version, training timestamp, training-data hash, example counts, dev metrics, per-entity thresholds and a **feature-schema version**. The loader refuses an artifact whose schema it does not implement, so an old agent never misreads a new model and a new model never ships to an agent that cannot use it. The schema version only changes when `pkg/pii/verify/features.go` changes.
* **Reproducible training**: `piigen` (seeded) → `piitrain` → `piieval` is a 60-second pipeline. Real labelled data is appended as another corpus file with the same JSONL format.
* **Promotion gate**: a candidate model must keep ≥ 99 % TP retention on (a) the frozen synthetic regression split, (b) the golden set, and (c) the real labelled set, and must not reduce FP reduction on any entity type by more than an agreed margin. `piieval` prints exactly that table.
* **Rollout**: shadow mode first (score and log, do not suppress), compare suppressions against analyst decisions for a week, then enable per entity type, then tighten thresholds. Rollback is the previous file.
* **Feedback loop**: every analyst decision on a detection ("confirmed" / "false positive") is a labelled candidate window — `piiexport` already writes that exact format for synthetic data. This is how GitHub's and Cloudflare's models are trained, and it is the only way to get real-distribution labels without copying customer files: store the 60-character windows and the decision, not the documents.
* **Drift monitoring**: per entity type, track the share of detections suppressed and the score histogram; a shift without a model change means the data changed and a retrain is due.
* **Weak-label safety**: keep the "never drop above 0.5" cap and the rules floor on every version, so a bad retrain degrades to today's behaviour rather than to silence.

## 9. Limitations and risks

* **All training and most evaluation data is synthetic.** The pooled phrasing split and the golden set are the two guards against learning the generator, and the golden set did catch real problems, but 48 documents is a smoke test, not a benchmark. The synthetic numbers are an upper bound; the golden numbers are a better guess of the first week on real data. Everything in sections 5–7 should be read as "the design works and the margins are large", not as a precision guarantee.
* **Names and addresses** are handled with dictionaries and patterns; precision on real prose will be lower than on identifiers. A token-classification NER model (served through the same ONNX path) is the right tool and is the first thing to add once model weights are available.
* **Tier B is unproven as a classifier here**: the environment cannot reach huggingface.co, so only a from-scratch tiny model was trained (the runtime, export, quantisation, parity and latency are proven). A pretrained small encoder is what made GitHub's verifier beat their LLM; expect it to matter most on unseen phrasing and OCR noise.
* **SQL/positional layouts, multi-line labels, tables rendered as text** are handled partially (CSV, JSON keys, `INSERT … VALUES` are covered; fixed-width and PDF tables are not).
* English only; Hindi/Hinglish keywords and Devanagari are out of scope by decision.
* The 12-digit `91`+mobile and `uid` cases remain genuinely ambiguous without more context; the model is right more often than the rules there, but neither is reliable.

## 10. Roadmap

1. **Now (1–2 weeks):** port attribution + competing-label suppression (rules) into the product gate behind a flag; add the non-PII label catalogue; run Tier A in shadow mode on real scans; start storing analyst decisions as labelled windows.
2. **Then:** retrain Tier A on real windows + synthetic; enable suppression per entity type where shadow results hold; evaluate the model gate (section 6.3) for recall.
3. **With pretrained weights available (allow huggingface.co for the training box, or bring the files in offline):** fine-tune a MiniLM/DeBERTa-xsmall-class verifier with `ml/train_tier_b.py`, compare against Tier A on the same harness, deploy on servers first.
4. **Names/addresses:** fine-tune a small NER (PERSON, ADDRESS) on synthetic + real windows, serve through `pkg/pii/onnx`, keep the gazetteer baseline as the fallback on endpoints.

## 11. How to reproduce

See [`../README.md`](../README.md). The whole synthetic pipeline (generate, train, evaluate including the golden set) runs in about a minute; `go test ./...` covers checksums, the label matcher, the baseline, the generator and the rule verifier; `go test -tags onnx ./pkg/pii/onnx` checks Go/Python parity for the ONNX path.

## 12. Sources

Vendor and open-source behaviour: Presidio `LemmaContextAwareEnhancer` and `PatternRecognizer` (github.com/microsoft/presidio), Presidio `IN_AADHAAR` recognizer; Google Sensitive Data Protection hotword and exclusion rules (docs.cloud.google.com/dlp/docs/creating-custom-infotypes-likelihood, …/dlp-inspect-string-custom-omit-overlap); AWS Macie custom data identifiers (docs.aws.amazon.com/macie/latest/user/cdis-options.html) and keyword-required managed identifiers; Microsoft Purview India SITs (learn.microsoft.com/purview/sit-defn-india-unique-identification-number); Nightfall context rules (help.nightfall.ai); Cloudflare "Improving DLP accuracy with AI context analysis" (blog.cloudflare.com, 2025); GitHub "Making secret scanning more trustworthy" and "GitHub's model for secret detection" (github.blog); Saha et al., COMSNETS 2020, "Secrets in Source Code: Reducing False Positives using Machine Learning"; RECAP, arXiv 2510.07551; GLiNER2-PII (arXiv 2605.09973), OpenAI Privacy Filter, Piiranha, ai4privacy open-pii-masking-500k, Nemotron-PII, chitragupta-pii-indic, rotalabs/sakshi-pii-in; python-stdnum `in_` validators; Faker `en_IN` ssn provider (Luhn, not Verhoeff, for Aadhaar); UIDAI "Features of Aadhaar"; HDFC Bank on RRN format; PII-TRACE (arXiv 2609.22200) and REDACT (arXiv 2606.19881) on LLM detector behaviour; "Fast DistilBERT on CPUs" (arXiv 2211.07715) and Intel Neural Compressor validated models for int8 CPU latency expectations.
