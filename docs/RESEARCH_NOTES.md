# Research notes: ML-based false-positive reduction for regex/checksum/keyword PII detection

Landscape scan performed 2026-11-04 by a research agent for this POC; lightly edited. Where a document host was unreachable from the sandbox the agent relied on search excerpts and GitHub-hosted source, and says so. Items it could not verify are listed at the end. Treat vendor-reported numbers as claims.

## 1. How existing tools disambiguate context

**Microsoft Presidio** (source read). `LemmaContextAwareEnhancer` defaults: `context_similarity_factor=0.35`, `min_score_with_context_similarity=0.4`, `context_prefix_count=5`, `context_suffix_count=0` — the window is up to 5 lemmatised tokens *before* the entity, nothing after. Boost: `score = min(max(score + 0.35, 0.4), 1.0)`. Checksums override context: `PatternRecognizer` sets a match to 1.0 when `validate_result` is true, so a Verhoeff-valid Aadhaar scores 1.0 without any keyword. The `IN_AADHAAR` recognizer: regex `\b[0-9]{12}\b`, context `["aadhaar","uidai"]`, validation = 12 digits, first digit ≥ 2, Verhoeff, not a palindrome; all India recognizers are disabled by default. Conflict resolution only drops a result contained in a same-type result with ≥ score; overlapping results of different types are both kept. No cross-type adjudication, no ML verification stage. On the canonical line both numbers are boosted and both score 1.0.

**Google Cloud Sensitive Data Protection.** Hotword rules with `proximity {windowBefore, windowAfter}` in characters and a `likelihoodAdjustment`; exclusion rules (`dictionary`, `regex`, `excludeInfoTypes` — drop a finding overlapping another infoType — and `excludeByHotword`). Many built-in detectors "require contextual clues". Nothing documented about attribution; the only lever for the two-numbers case is negative evidence (a custom "bank account" infoType plus `excludeInfoTypes`, or `excludeByHotword` with "bank|acc" in a small `windowBefore`).

**AWS.** Macie custom identifiers: `maximumMatchDistance` 1–300 chars (default 50) from the end of a keyword to the end of the match; for unstructured data the keyword must precede the match; `INDIA_AADHAAR_NUMBER` and `INDIA_PERMANENT_ACCOUNT_NUMBER` are keyword-required managed identifiers. Comprehend `DetectPiiEntities` is model-based with per-entity confidence and `IN_AADHAAR` / `IN_PERMANENT_ACCOUNT_NUMBER` types; no rules exposed.

**Microsoft Purview.** Aadhaar SIT: first digit ≠ 0/1, Verhoeff via `Func_india_aadhaar`, keywords `aadhaar, aadhar, uid, uidai, आधार`, `patternsProximity=300`, confidence 85 with keyword vs 75 without. PAN and GST SITs similar (GST has a checksum function). Same per-element proximity model.

**Nightfall.** ML detectors plus RE2/word-list custom detectors; context rules raise/lower confidence based on 40 bytes before and after; exclusion rules post-detection. Vendor claims are unverified.

**Cloudflare DLP** ("AI context analysis", 2025). After a pattern match, the surrounding context (matched text redacted) is embedded and a nearest-neighbour search over previously reported true/false positives adjusts confidence.

Bottom line: every vendor uses a per-candidate proximity window (5 tokens / 40 B / 50–300 chars); none documents label-to-value attribution.

## 2. Open-weight PII span/token models (as of Nov 2026)

| model | base / params | license | reported quality | coverage |
|---|---|---|---|---|
| GLiNER2-PII (Fastino, 2026) | GLiNER2, ~205–300M | Apache 2.0 | SPY exact-match F1 0.47 (best of compared) | 42 types incl. national_id, tax_id, bank_account; no Indian labels; EN/FR/ES/DE/IT/PT/NL |
| OpenAI Privacy Filter (2026) | 1.5B MoE, ~50M active | Apache 2.0 | ~96–97 % F1 on PII-Masking-300k (own claim) | 8 coarse categories, English-focused |
| nvidia/gliner-PII | GLiNER large → bi-large, ~0.45–0.57B | NVIDIA Open Model | strict F1 0.70 (Argilla) / 0.64 (AI4Privacy) | 55+ types, English, trained on Nemotron-PII (synthetic) |
| urchade/gliner_multi_pii-v1 | GLiNER multi, ~209M | Apache 2.0 | SPY F1 ≈ 0.38–0.40 | 50+ synthetic types, multilingual, no Hindi claim |
| Piiranha-v1 | mdeberta-v3-base, ~278M | CC-BY-NC-ND-4.0 | F1 93.1 on own split | 17 types, 6 European languages |
| ai4privacy open-pii-masking-500k + ModernBERT anonymiser | — / ModernBERT-base | mixed | F1 0.988 own split | Hindi & Telugu experimental; no Indian-ID classes |
| Kaggle PII Data Detection 2024 winners | DeBERTa-v3-large ensembles | code MIT-ish | F5 metric | essay PII, generic ID_NUM |
| Indian: chitragupta-pii-indic | Apache-2.0, 18 types incl. Aadhaar/PAN/GSTIN/voter, Verhoeff + GSTIN check | — | 0.959 F1 with rules vs 0.876 model-only (author's synthetic set) | private synthetic data |
| Indian: rotalabs/sakshi-pii-in | MuRIL, PERSON/ADDRESS/DOB only | — | micro-F1 0.979 | Hinglish/Devanagari/Bengali/Tamil; deliberately defers IDs to regex+checksum |
| Others | hiteshwadhwani/pii-model-indicv2 (IndicBERTv2), plingampally/meridianpii-hi-v1, VK1402/AADHAAR_Extractor (GLiNER2-base, synthetic) | | | |

Sizes for CPU: GLiNER paper tiers 50M/90M/300M; released checkpoints 166M/209M/459M. Independent multilingual benchmark (REDACT): Claude Sonnet 4.6 0.636 F1 > GPT-4.1 0.597 > OpenAI Privacy Filter 0.512 > GLiNER 0.32 > Presidio 0.195.

## 3. Second-stage candidate verification (regex → classifier)

* **GitHub secret scanning.** An LLM verification step with explicit reasoning cut customer-confirmed false positives by 75.76 %; it was then replaced by a fine-tuned ModernBERT context classifier scoring each candidate from file path, credential type, variable names and nearby text, trained on user feedback plus near-miss lookalikes: +13 % precision, +19 % recall vs the LLM pipeline, 86 % faster post-push scans, >98 % lower serving cost, "<2 ms per candidate set".
* **Saha et al., COMSNETS 2020** ("Secrets in Source Code: Reducing False Positives using ML"): regex candidates → LR + NB + SVM voting classifier on context features: precision 84 %, recall 89 % on ~2k test examples.
* **SAP Credential Digger**: regex scanner → "path model" and "snippet model" auto-mark false positives (no published numbers).
* **RECAP (arXiv 2510.07551)**: regex for structured PII, candidate kept only if a zero-shot LLM confirms it, explicitly to kill numeric coincidences; weighted F1 0.511 → 0.657.
* **ESEM 2025** secret-breach paper: regex candidates → fine-tuned Llama-3.1-8B binary classifier, F1 0.985.
* **Capital One DataProfiler**: char-level CNN labeler for sensitive data beating a regex baseline.
* **Label-to-value attribution** is treated as key-value linking / relation extraction in document AI (FUNSD RE F1: LayoutLMv3-large ≈ 80, GeoLayoutLM 89.5) — a learned, imperfect task even with layout features. No PII-specific paper on "which keyword governs which number" was found.

## 4. LLM-as-verifier for PII

Accuracy evidence: PII-TRACE (13k synthetic dialogues, 13 languages, frontier LLMs zero-shot): no detector achieves full entity coverage without substantial false positives on PII-free text; REDACT: LLMs most robust on high-sensitivity entities; SPY: zero-shot recall ≫ precision (over-flagging names); CAPID: context-aware LLM detection yields many non-sensitive "context fragment" FPs; PRvL: under-redaction dominant. Verifier-on-candidates reports: RECAP, the ESEM-2025 paper, GitHub Copilot secret scanning ("GPT-4 as a confirming scanner improved precision without reducing recall at higher cost"). Failure modes: over-flagging/overconfidence, unstable reasoning, span drift on OCR fragments, indirect prompt injection from scanned content (OWASP LLM01). Pattern if ever used: send only `[left] <<candidate>> [right]` plus type and checksum result, closed-set JSON answer, structured outputs, frozen prefix. Cost arithmetic at list prices (Oct 2026): roughly $30 per million candidates with a small model via batch APIs. Not applicable to this product (local-only), recorded for completeness.

## 5. Indian PII data resources and ID structure

* Aadhaar: 12 random digits, Verhoeff check digit, first digit ≠ 0/1 (Presidio/Purview/python-stdnum agree; UIDAI's own spec page not found); Presidio also rejects palindromes. VID: 16 digits, Verhoeff, not starting 0/1. A single check digit rejects only ~90 % of random 12-digit strings.
* 12-digit shapes that collide with Aadhaar: EPFO UAN, EPS PPO number, NPS PRAN, APAAR student ID, IMPS/UPI RRN (mask `YDDDHHSSSSSS`), bank accounts (9–18 digits, several banks at 12), mobiles written with the 91 prefix. Nearby but non-colliding: ABHA 14, VID 16, Aadhaar enrolment ID 14.
* PAN = 3 letters + holder-type letter from `CPHFATBLJG` + 1 letter + 4 digits + 1 letter (no public checksum); GSTIN = 2-digit state code + PAN + entity code + `Z` + base-36 check character.
* Generators: Faker `en_IN` `aadhaar_id()` appends a **Luhn** digit, not Verhoeff (fails a correct validator ~90 % of the time); `indic-faker` and `india-kit` do Verhoeff; python-stdnum `in_` validators; presidio-research data generator for template-to-span synthesis.
* Datasets: ai4privacy 500k (Hindi/Telugu experimental, no Indian-ID classes), Nemotron-PII (English, synthetic), Naamapadam (11 Indic languages, PER/LOC/ORG only), COMI-LINGUA (Hinglish NER). **No public corpus with Aadhaar/PAN/GSTIN/IFSC/bank accounts in realistic context was found**; Indian models train on private synthetic data.

## 6. Practical CPU inference

No clean published table for a 20–100M encoder at 128 tokens int8 on ONNX Runtime. Data points: Intel "single-digit millisecond" DistilBERT on Ice Lake; "Fast DistilBERT on CPUs" 2.76 ms/sample (ORT int8 + sparsity, seq 32); GitHub's ModernBERT verifier "<2 ms per candidate set"; a vendor post quoting ~80 ms for MiniLM-L6 at 128 tokens on one core (precision unclear). Working expectation: low single-digit to ~20 ms per call for a 6-layer int8 model on a modern x86 core — consistent with what this POC measured from Go (see RESULTS.md). Non-Python runtimes: Java (`com.microsoft.onnxruntime`, DJL), .NET (`Microsoft.ML.OnnxRuntime` + `Microsoft.ML.Tokenizers`), Go (`yalue/onnxruntime_go` + `daulet/tokenizers` or `sugarme/tokenizer`, `knights-analytics/hugot`), Rust (`ort`, `candle`). A tiny feature-based verifier is competitive for this binary task (Saha 2020; Bank of England working paper on penalised LR in the tens-of-shot regime; Credential Digger).

## What could not be verified

Google's exact Aadhaar infoType name and likelihood internals; Macie's keyword list for `INDIA_AADHAAR_NUMBER`; UIDAI's official statement of the first-digit rule; Kaggle private-leaderboard scores; Piiranha's test set; GLiNER2-PII parameter count discrepancy; ai4privacy 500k license; published int8 latency tables for MiniLM/DeBERTa-xsmall/ModernBERT; GitHub's context-window length and hardware; per-model precision in PII-TRACE/REDACT; vendor marketing claims; the "mobile with 91 prefix" collision and specific bank account lengths are from secondary sources.
