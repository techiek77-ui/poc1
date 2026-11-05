# Results on the held-out test split

Corpus `corpus.jsonl`: 30000 documents (6000 in test). Window 60 chars. Model `tier_a_lr.json` (version 20261105T045354Z, trained 2026-11-05T04:53:54Z).

Baseline = regex + checksum + same-type keyword within 60 chars (or CSV header). Every verifier only re-scores baseline detections. Thresholds for the ML configurations are chosen on the dev split to retain ≥ 99% of the baseline's true positives per entity type; the rule verifier uses its natural 0.5 threshold.

### Gated candidates (the product's detections)

| configuration | detections kept | TP | FP | precision | TP retention (detections) | span retention | FP reduction |
|---|---:|---:|---:|---:|---:|---:|---:|
| 0 baseline | 29616 | 26370 | 3246 | 89.0% | 100.00% | 100.00% | 0.0% |
| 1 rules @0.5 | 26141 | 25389 | 752 | 97.1% | 96.28% | 96.22% | 76.8% |
| 1 rules @99% | 29439 | 26365 | 3074 | 89.6% | 99.98% | 99.98% | 5.3% |
| 1 rules @0.5 +resolve | 26136 | 25389 | 747 | 97.1% | 96.28% | 96.22% | 77.0% |
| A linear | 26438 | 26212 | 226 | 99.1% | 99.40% | 99.42% | 93.0% |
| A linear (re-tuned on dev @99%) | 26438 | 26212 | 226 | 99.1% | 99.40% | 99.42% | 93.0% |
| A linear +resolve | 26428 | 26211 | 217 | 99.2% | 99.40% | 99.41% | 93.3% |
| B onnx | 29011 | 26184 | 2827 | 90.3% | 99.29% | 99.28% | 12.9% |

_TP retention counts detections (several may cover one labelled span); span retention counts labelled spans that keep at least one detection._

#### Per entity type (baseline FP → FP after each configuration; TP retention in brackets)

| type | truth spans | baseline TP | baseline recall | baseline FP | baseline precision | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear (re-tuned on dev @99%) | A linear +resolve | B onnx |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| AADHAAR | 5866 | 5163 | 88.0% | 818 | 86.3% | 195 (93.6%) | 793 (100.0%) | 195 (93.6%) | 70 (97.8%) | 70 (97.8%) | 70 (97.8%) | 791 (99.3%) |
| ADDRESS | 1639 | 2217 | 97.3% | 44 | 98.1% | 43 (98.9%) | 44 (100.0%) | 43 (98.9%) | 18 (99.5%) | 18 (99.5%) | 18 (99.5%) | 44 (100.0%) |
| BANK_ACCOUNT | 2558 | 2106 | 82.3% | 1388 | 60.3% | 281 (98.3%) | 1379 (100.0%) | 276 (98.3%) | 73 (99.4%) | 73 (99.4%) | 64 (99.4%) | 1260 (99.5%) |
| CREDIT_CARD | 1004 | 852 | 84.9% | 3 | 99.6% | 0 (98.4%) | 3 (99.9%) | 0 (98.4%) | 1 (99.9%) | 1 (99.9%) | 1 (99.9%) | 3 (100.0%) |
| DOB | 1395 | 1012 | 72.5% | 45 | 95.7% | 26 (97.5%) | 45 (100.0%) | 26 (97.5%) | 3 (100.0%) | 3 (100.0%) | 3 (100.0%) | 16 (99.5%) |
| DRIVING_LICENCE | 611 | 515 | 84.3% | 0 | 100.0% | 0 (97.9%) | 0 (100.0%) | 0 (97.9%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| EMAIL | 1384 | 1022 | 73.8% | 0 | 100.0% | 0 (98.4%) | 0 (100.0%) | 0 (98.4%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (99.7%) |
| GSTIN | 795 | 620 | 78.0% | 0 | 100.0% | 0 (98.2%) | 0 (100.0%) | 0 (98.2%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| IFSC | 1102 | 939 | 85.2% | 1 | 99.9% | 0 (99.0%) | 1 (100.0%) | 0 (99.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 1 (100.0%) |
| MOBILE | 2273 | 1863 | 82.0% | 58 | 97.0% | 9 (98.6%) | 58 (100.0%) | 9 (98.6%) | 5 (99.8%) | 5 (99.8%) | 5 (99.8%) | 45 (96.6%) |
| NAME | 6062 | 5027 | 82.9% | 479 | 91.3% | 98 (98.4%) | 349 (100.0%) | 98 (98.4%) | 16 (99.9%) | 16 (99.9%) | 16 (99.9%) | 280 (99.0%) |
| PAN | 1553 | 1289 | 83.0% | 0 | 100.0% | 0 (93.9%) | 0 (100.0%) | 0 (93.9%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| PASSPORT | 597 | 460 | 77.1% | 0 | 100.0% | 0 (99.1%) | 0 (100.0%) | 0 (99.1%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| PINCODE | 2411 | 915 | 38.0% | 85 | 91.5% | 52 (83.8%) | 84 (100.0%) | 52 (83.8%) | 33 (99.5%) | 33 (99.5%) | 33 (99.5%) | 64 (100.0%) |
| UAN | 1199 | 945 | 78.8% | 317 | 74.9% | 46 (97.8%) | 311 (99.9%) | 46 (97.8%) | 6 (99.7%) | 6 (99.7%) | 6 (99.7%) | 316 (99.2%) |
| UPI_ID | 810 | 710 | 87.7% | 0 | 100.0% | 0 (97.3%) | 0 (100.0%) | 0 (97.3%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| VID | 314 | 259 | 82.5% | 2 | 99.2% | 1 (99.2%) | 1 (99.2%) | 1 (99.2%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 2 (97.3%) |
| VOTER_ID | 566 | 456 | 80.6% | 6 | 98.7% | 1 (73.7%) | 6 (100.0%) | 1 (73.7%) | 1 (99.8%) | 1 (99.8%) | 1 (99.8%) | 5 (99.8%) |

#### Per data shape (FP after each configuration; TP retention in brackets)

| shape | baseline TP | baseline FP | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear (re-tuned on dev @99%) | A linear +resolve | B onnx |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| card | 322 | 11 | 0 (91.9%) | 8 (100.0%) | 0 (91.9%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 9 (100.0%) |
| code | 789 | 171 | 49 (97.7%) | 139 (100.0%) | 49 (97.7%) | 10 (98.7%) | 10 (98.7%) | 10 (98.7%) | 162 (99.5%) |
| csv | 11128 | 681 | 188 (97.1%) | 635 (100.0%) | 188 (97.1%) | 78 (99.8%) | 78 (99.8%) | 78 (99.8%) | 551 (99.0%) |
| form | 4142 | 800 | 85 (95.4%) | 762 (100.0%) | 83 (95.4%) | 19 (99.6%) | 19 (99.6%) | 18 (99.6%) | 669 (99.4%) |
| json | 2458 | 294 | 78 (98.3%) | 277 (100.0%) | 78 (98.3%) | 22 (99.6%) | 22 (99.6%) | 22 (99.6%) | 271 (99.8%) |
| log | 1786 | 317 | 115 (98.5%) | 308 (100.0%) | 115 (98.5%) | 14 (99.4%) | 14 (99.4%) | 13 (99.4%) | 275 (99.5%) |
| ocr | 1840 | 261 | 54 (94.3%) | 252 (99.9%) | 53 (94.3%) | 21 (99.0%) | 21 (99.0%) | 20 (99.0%) | 237 (99.5%) |
| prose | 3905 | 711 | 183 (93.4%) | 693 (99.9%) | 181 (93.4%) | 62 (98.1%) | 62 (98.1%) | 56 (98.1%) | 653 (99.5%) |

#### False positives by what the span really was (baseline FP → FP after each configuration)

| span really is | baseline FP | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear (re-tuned on dev @99%) | A linear +resolve | B onnx |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| aadhaar | 384 | 65 | 382 | 62 | 20 | 20 | 13 | 348 |
| rrn | 297 | 29 | 295 | 29 | 10 | 10 | 10 | 277 |
| no_truth | 277 | 138 | 277 | 138 | 9 | 9 | 9 | 184 |
| address | 216 | 33 | 86 | 33 | 14 | 14 | 14 | 97 |
| mobile | 199 | 26 | 185 | 25 | 9 | 9 | 7 | 189 |
| ppo | 153 | 23 | 152 | 23 | 9 | 9 | 9 | 144 |
| apaar | 135 | 12 | 132 | 12 | 3 | 3 | 3 | 124 |
| consumer_no | 134 | 24 | 133 | 24 | 5 | 5 | 5 | 123 |
| customer_id | 116 | 18 | 115 | 18 | 8 | 8 | 8 | 111 |
| uan | 115 | 21 | 114 | 20 | 6 | 6 | 6 | 111 |
| tracking | 100 | 14 | 98 | 14 | 9 | 9 | 9 | 99 |
| policy | 95 | 17 | 95 | 17 | 6 | 6 | 6 | 88 |
| order_id | 92 | 14 | 91 | 14 | 8 | 8 | 8 | 88 |
| txn_id | 90 | 10 | 89 | 10 | 2 | 2 | 2 | 81 |
| bank_account | 87 | 8 | 85 | 8 | 4 | 4 | 4 | 83 |
| generic_id | 85 | 83 | 85 | 83 | 17 | 17 | 17 | 85 |
| dummy | 71 | 33 | 63 | 33 | 8 | 8 | 8 | 71 |
| user_id | 67 | 52 | 67 | 52 | 17 | 17 | 17 | 57 |
| device_id | 62 | 8 | 62 | 8 | 5 | 5 | 5 | 56 |
| pan | 61 | 2 | 61 | 2 | 0 | 0 | 0 | 60 |
| pran | 52 | 19 | 52 | 19 | 2 | 2 | 2 | 51 |
| upi_id | 50 | 6 | 50 | 6 | 1 | 1 | 1 | 40 |
| invoice | 48 | 9 | 47 | 9 | 3 | 3 | 3 | 44 |
| ticket | 48 | 2 | 48 | 2 | 1 | 1 | 1 | 45 |
| otp | 41 | 28 | 41 | 28 | 23 | 23 | 23 | 31 |

### FP reduction vs. TP retention target (thresholds from dev, measured on test)

| rules: TP retention target | achieved retention (test) | FP reduction (test) | precision |
|---|---:|---:|---:|
| 99.9% | 99.98% | 5.0% | 89.5% |
| 99.5% | 99.98% | 5.3% | 89.6% |
| 99.0% | 99.98% | 5.3% | 89.6% |
| 98.0% | 99.53% | 13.1% | 90.3% |
| 95.0% | 97.88% | 74.1% | 96.8% |

| linear: TP retention target | achieved retention (test) | FP reduction (test) | precision |
|---|---:|---:|---:|
| 99.9% | 99.96% | 61.1% | 95.4% |
| 99.5% | 99.69% | 87.7% | 98.5% |
| 99.0% | 99.40% | 93.0% | 99.1% |
| 98.0% | 99.23% | 94.3% | 99.3% |
| 95.0% | 99.18% | 94.9% | 99.4% |

**Note:** corpus.jsonl also served as training data for this model (same file hash in the manifest); real-document numbers below are not out-of-sample.

### Keyword gate vs. model gate

The keyword gate also costs recall: values printed without a nearby label (identity-card OCR, bare table cells) are never reported. Scoring every regex+checksum match with the model instead of requiring a keyword recovers some of them.

| candidate set | TP | FP | precision | recall vs. all labelled spans |
|---|---:|---:|---:|---:|
| 0 baseline | 26370 | 3246 | 89.0% | 80.1% |
| A linear | 26212 | 226 | 99.1% | 79.6% |
| regex+checksum only (no gate) | 31939 | 29857 | 51.7% | 97.4% |
| A linear as the gate | 31768 | 3291 | 90.6% | 96.9% |

### Golden set: 48 hand-written documents in styles the generator does not produce

| configuration | TP | FP | precision | TP retention | FP reduction | recall vs. all spans |
|---|---:|---:|---:|---:|---:|---:|
| 0 baseline | 147 | 22 | 87.0% | 100.0% | 0.0% | 94.0% |
| 1 rules @0.5 | 147 | 3 | 98.0% | 100.0% | 86.4% | 94.0% |
| A linear | 145 | 3 | 98.0% | 98.6% | 86.4% | 92.7% |
| A linear as the gate (ungated) | 152 | 6 | 96.2% | – | – | 97.3% |

Every wrong decision (gated candidates):

| doc | claim | value | really | rules | linear | outcome | model explanation |
|---|---|---|---|---:|---:|---|---|
| g007 csv | AADHAAR | `440291873659` | aadhaar | 0.70 | 0.016 | MISSED by linear | same_fbv:-1.73 AADHAAR&same_fbv:-0.93 same_dist=0:+0.93 AADHAAR&r1=aadhaar:-0.80 |
| g016 prose | AADHAAR | `528374619208` | tracking | 0.60 | 0.016 | fp kept by rules | rules: competing label 'awb' (tracking) governs, claim is AADHAAR; but 'aadhaar' attached on the right · model: same_sep=paren:+1.70 AADHAAR&same_sep=paren:+1.07 l1c=awb:-0.97 l1=awb:-0.92 |
| g025 code | AADHAAR | `840162937518` | aadhaar | 0.97 | 0.313 | MISSED by linear | same_dist=0:+0.93 AADHAAR&adj_left=aadhaar:+0.88 l1=bhat:-0.72 groups=12:-0.71 |
| g026 code | PAN | `ABCPE1234F` | dummy | 0.76 | 1.000 | FP kept by both | rules: governing label 'pan' (left) matches claim; test/sample wording nearby · model: l2=test:-0.96 PAN&linelen=0:+0.68 PAN&linedig=1:+0.64 l1=pan:+0.63 |
| g026 code | MOBILE | `9999999999` | dummy | 0.33 | 0.779 | FP kept by linear | dummy:-2.15 MOBILE&gov_same:+1.38 MOBILE&gov=mobile:+1.38 rep=4:-1.22 |
| g026 code | BANK_ACCOUNT | `234567890124` | dummy | 0.33 | 0.136 | FP kept by linear | dummy:-2.15 seqrun=2:-1.71 BANK_ACCOUNT&adj_left=bank_account:+1.01 l2=test:-0.96 |
| g044 prose | AADHAAR | `682739104538` | customer_id | 0.80 | 0.031 | fp kept by rules | rules: governing label 'uid' (left) matches claim · model: same_dist=0:+0.93 AADHAAR&adj_left=aadhaar:+0.88 l12=for_uid:-0.82 AADHAAR&same_weak:-0.81 |

Labelled spans the baseline never proposes (no keyword in the window), and whether the model-as-gate recovers them:

- g031 NAME `Suresh Babu` — recovered by model gate
- g031 NAME `Venkata Rao` — recovered by model gate
- g037 AADHAAR `394850172636` — recovered by model gate
- g037 BANK_ACCOUNT `702619483524` — recovered by model gate
- g037 IFSC `BARB0MUMBAI` — recovered by model gate
- g037 AADHAAR `258391047616` — recovered by model gate
- g037 BANK_ACCOUNT `1234567890123` — not recovered
- g037 IFSC `PUNB0123400` — recovered by model gate
- g041 NAME `Kavita` — not recovered

### Cost per candidate (this container: 4 vCPU Xeon @ 2.1 GHz, single goroutine)

| stage | µs / candidate |
|---|---:|
| context analysis (attrib.Analyze, shared) | 28.9 |
| rule verifier | 0.67 |
| linear verifier (features + dot product + explanation) | 57.2 |
| ONNX transformer verifier (tokenize + ONNX Runtime, B onnx) | 2403.6 |

### Remaining false positives after `A linear` (sample)

- `log [PINCODE truth=otp 0.97] …58,013 INFO ledger - kyc verify atm-pin=«884656» panNumber=OCMPB9618I sta…`
- `log [BANK_ACCOUNT truth=pran 0.12] …ofbirth":"07.10.1989","nps_account_no":"«110944144153»","uid":"2904 4149 8966"}…`
- `form [AADHAAR truth=user_id 0.66] …4 3274 7945    IFSC:#KKBK0817492    UID#«912119555155»…`
- `csv [AADHAAR truth=user_id 0.95] …7393 5628 0090,106464937261,UBIN0227661,«881922294390»,sanaferna@apl,"799141002…`
- `csv [BANK_ACCOUNT truth=consumer_no 0.36] …mber⏎Sushma Babita Chacko,CUST0437000,«549591331756»,AADHAR,"667528693540","1…`
- `prose [MOBILE truth=aadhaar 0.97] …e Aadhaar of applicant field instead of «919886131941».⏎contact_number docume…`
- `csv [AADHAAR truth=tracking 0.54] …AAR;5472 5185 0642;6bdd5493;08569867654;«380209014401»;"PF Account Number no 10…`
- `csv [AADHAAR truth=tracking 0.83] …aar;8317-8568-9367;24146;+91-9681276232;«457180104626»;"Please quote ticket INC…`
- `csv [AADHAAR truth=generic_id 0.64] …yankaratho@airtel;AADHAR;4866 0028 7322;«339029045475»;+91 86443 58789;12792079…`
- `prose [UAN truth=aadhaar 0.88] …UID / PF No : «643589956185» / 108592569365⏎KYC rec…`
- `csv [AADHAAR truth=user_id 0.94] …4869612,8354869612@oksbi,5904-2831-2091,«835835098323»⏎…`
- `ocr [ADDRESS truth=address 0.58] …igues, Door No 154  PankajGarg Street; H«auz Khas, Jaipur - 598467»⏎⏎5155 8669 5821⏎VI…`
- `form [NAME truth=no_truth 0.71] …IMEI DNL411186252 | «CONTACT PERSON TAPAN» MIRZA | number 699078717…`
- `json [BANK_ACCOUNT truth=credit_card 0.54] …": 7847336583715760,"rupaycardnumber": "«5557261171631280»","ref": "Iyengar Kishore…`
- `json [PINCODE truth=otp 0.99] …arnumber": "4095-8379-7245","atm_pin": "«553733»","data": "878965589945",…`

### Example explanations written to the log by `A linear`

- AADHAAR `487508087069` truth=policy score=0.000 keep=false — gov_kind_sep=policy/space:-1.78 AADHAAR&gov_kind_sep=policy/space:-1.52 same_nl:-1.45 groups=12:-0.71
- AADHAAR `5115 3870 7249` truth=aadhaar score=1.000 keep=true — AADHAAR&l12=aadhaar_no:+2.83 AADHAAR&groups=4-4-4:+1.45 groups=4-4-4:+1.45 same_dist=0:+0.93
- AADHAAR `5497 8958 7091` truth=aadhaar score=1.000 keep=true — AADHAAR&groups=4-4-4:+1.45 groups=4-4-4:+1.45 same_dist=0:+0.93 AADHAAR&adj_left=aadhaar:+0.88
- AADHAAR `3562 5756 4111` truth=aadhaar score=1.000 keep=true — same_sep=paren:+1.70 groups=4-4-4:+1.45 AADHAAR&groups=4-4-4:+1.45 AADHAAR&same_sep=paren:+1.07
- AADHAAR `544997756509` truth=aadhaar score=0.981 keep=true — same_dist=0:+0.93 AADHAAR&adj_left=aadhaar:+0.88 l1c=arn:+0.73 groups=12:-0.71
- AADHAAR `492868931060` truth=rrn score=0.000 keep=false — same_nl:-1.45 gov_kind_sep=txn_ref/tab:-0.99 AADHAAR&adj_left=aadhaar:+0.88 same_sep=newline:-0.82

_Generated 2026-11-05T05:35:18Z._
