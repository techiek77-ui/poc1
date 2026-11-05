# Results on the held-out test split

Corpus `corpus.jsonl`: 30000 documents (6000 in test). Window 60 chars. Model `tier_a_lr.json` (version 20261105T030257Z, trained 2026-11-05T03:02:57Z).

Baseline = regex + checksum + same-type keyword within 60 chars (or CSV header). Every verifier only re-scores baseline detections. Thresholds for the ML configurations are chosen on the dev split to retain ≥ 99% of the baseline's true positives per entity type; the rule verifier uses its natural 0.5 threshold.

### Gated candidates (the product's detections)

| configuration | detections kept | TP | FP | precision | TP retention | FP reduction |
|---|---:|---:|---:|---:|---:|---:|
| 0 baseline | 29561 | 25788 | 3773 | 87.2% | 100.00% | 0.0% |
| 1 rules @0.5 | 26123 | 25060 | 1063 | 95.9% | 97.18% | 71.8% |
| 1 rules @99% | 27139 | 25721 | 1418 | 94.8% | 99.74% | 62.4% |
| 1 rules @0.5 +resolve | 26117 | 25058 | 1059 | 95.9% | 97.17% | 71.9% |
| A linear | 25764 | 25685 | 79 | 99.7% | 99.60% | 97.9% |
| A linear +resolve | 25764 | 25685 | 79 | 99.7% | 99.60% | 97.9% |
| B onnx | 27139 | 24695 | 2444 | 91.0% | 95.76% | 35.2% |

#### Per entity type (baseline FP → FP after each configuration; TP retention in brackets)

| type | truth spans | baseline TP | baseline recall | baseline FP | baseline precision | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear +resolve | B onnx |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| AADHAAR | 5761 | 4973 | 86.3% | 719 | 87.4% | 136 (92.0%) | 182 (99.3%) | 135 (91.9%) | 45 (98.5%) | 45 (98.5%) | 458 (89.8%) |
| ADDRESS | 1662 | 2419 | 98.3% | 399 | 85.8% | 367 (96.6%) | 399 (100.0%) | 367 (96.6%) | 8 (99.8%) | 8 (99.8%) | 338 (98.7%) |
| BANK_ACCOUNT | 2554 | 2178 | 85.3% | 1412 | 60.7% | 209 (100.0%) | 209 (100.0%) | 206 (100.0%) | 8 (99.6%) | 8 (99.6%) | 976 (96.7%) |
| CREDIT_CARD | 982 | 845 | 86.0% | 3 | 99.6% | 0 (99.6%) | 0 (99.6%) | 0 (99.6%) | 0 (100.0%) | 0 (100.0%) | 3 (100.0%) |
| DOB | 1408 | 1148 | 81.5% | 51 | 95.7% | 36 (99.7%) | 36 (99.7%) | 36 (99.7%) | 0 (99.8%) | 0 (99.8%) | 23 (99.6%) |
| DRIVING_LICENCE | 578 | 482 | 83.4% | 0 | 100.0% | 0 (99.8%) | 0 (99.8%) | 0 (99.8%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| EMAIL | 1373 | 1054 | 76.8% | 0 | 100.0% | 0 (99.7%) | 0 (99.7%) | 0 (99.7%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| GSTIN | 760 | 603 | 79.3% | 0 | 100.0% | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| IFSC | 1069 | 902 | 84.4% | 0 | 100.0% | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| MOBILE | 2314 | 1886 | 81.5% | 63 | 96.8% | 8 (99.6%) | 8 (99.6%) | 8 (99.6%) | 3 (99.7%) | 3 (99.7%) | 42 (98.9%) |
| NAME | 5987 | 4625 | 77.1% | 440 | 91.3% | 64 (98.2%) | 340 (100.0%) | 64 (98.2%) | 5 (99.8%) | 5 (99.8%) | 184 (90.4%) |
| PAN | 1533 | 1251 | 81.6% | 0 | 100.0% | 0 (94.3%) | 0 (99.8%) | 0 (94.3%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| PASSPORT | 600 | 454 | 75.7% | 0 | 100.0% | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| PINCODE | 776 | 641 | 82.6% | 339 | 65.4% | 203 (99.5%) | 203 (99.5%) | 203 (99.5%) | 9 (99.5%) | 9 (99.5%) | 141 (99.7%) |
| UAN | 1138 | 984 | 86.5% | 343 | 74.2% | 40 (99.9%) | 40 (99.9%) | 40 (99.9%) | 1 (100.0%) | 1 (100.0%) | 276 (99.5%) |
| UPI_ID | 805 | 663 | 82.4% | 0 | 100.0% | 0 (99.2%) | 0 (99.2%) | 0 (99.2%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| VID | 316 | 254 | 80.4% | 3 | 98.8% | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 3 (100.0%) |
| VOTER_ID | 549 | 426 | 77.6% | 1 | 99.8% | 0 (85.2%) | 1 (99.8%) | 0 (85.2%) | 0 (100.0%) | 0 (100.0%) | 0 (97.9%) |

#### Per data shape (FP after each configuration; TP retention in brackets)

| shape | baseline TP | baseline FP | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear +resolve | B onnx |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| card | 320 | 17 | 6 (91.9%) | 6 (98.8%) | 6 (91.9%) | 0 (100.0%) | 0 (100.0%) | 9 (100.0%) |
| code | 761 | 177 | 53 (97.6%) | 64 (99.6%) | 53 (97.6%) | 12 (98.7%) | 12 (98.7%) | 118 (97.6%) |
| csv | 10385 | 966 | 376 (97.3%) | 444 (99.7%) | 376 (97.3%) | 33 (99.7%) | 33 (99.7%) | 622 (93.2%) |
| form | 4424 | 909 | 129 (97.4%) | 280 (99.9%) | 127 (97.4%) | 4 (99.7%) | 4 (99.7%) | 584 (97.7%) |
| json | 2422 | 301 | 88 (98.2%) | 99 (100.0%) | 88 (98.2%) | 6 (99.5%) | 6 (99.5%) | 220 (98.4%) |
| log | 1664 | 353 | 158 (96.9%) | 174 (99.8%) | 158 (96.9%) | 8 (99.4%) | 8 (99.4%) | 174 (95.7%) |
| ocr | 1886 | 272 | 68 (97.1%) | 108 (99.7%) | 67 (97.1%) | 4 (99.2%) | 4 (99.2%) | 186 (98.1%) |
| prose | 3926 | 778 | 185 (96.5%) | 243 (99.7%) | 184 (96.4%) | 12 (99.5%) | 12 (99.5%) | 531 (96.7%) |

#### False positives by what the span really was (baseline FP → FP after each configuration)

| span really is | baseline FP | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear +resolve | B onnx |
|---|---:|---:|---:|---:|---:|---:|---:|
| address | 483 | 189 | 272 | 189 | 2 | 2 | 150 |
| aadhaar | 441 | 70 | 75 | 69 | 4 | 4 | 317 |
| no_truth | 313 | 173 | 308 | 173 | 4 | 4 | 191 |
| rrn | 292 | 34 | 37 | 34 | 2 | 2 | 181 |
| mobile | 211 | 44 | 47 | 42 | 2 | 2 | 150 |
| ppo | 158 | 12 | 14 | 12 | 3 | 3 | 105 |
| customer_id | 149 | 21 | 22 | 21 | 1 | 1 | 121 |
| apaar | 138 | 14 | 15 | 14 | 2 | 2 | 101 |
| consumer_no | 125 | 26 | 26 | 26 | 2 | 2 | 86 |
| uan | 123 | 22 | 22 | 22 | 2 | 2 | 79 |
| txn_id | 115 | 20 | 22 | 20 | 1 | 1 | 77 |
| tracking | 110 | 9 | 10 | 9 | 2 | 2 | 79 |
| generic_id | 97 | 81 | 87 | 81 | 9 | 9 | 83 |
| bank_account | 93 | 24 | 31 | 23 | 2 | 2 | 69 |
| policy | 82 | 11 | 14 | 11 | 1 | 1 | 54 |
| order_id | 81 | 10 | 27 | 10 | 3 | 3 | 59 |
| dummy | 70 | 27 | 27 | 27 | 12 | 12 | 54 |
| invoice | 70 | 36 | 40 | 36 | 1 | 1 | 55 |
| upi_id | 65 | 8 | 8 | 8 | 1 | 1 | 46 |
| device_id | 61 | 1 | 9 | 1 | 0 | 0 | 44 |
| user_id | 60 | 41 | 41 | 41 | 10 | 10 | 23 |
| credit_card | 52 | 14 | 14 | 14 | 0 | 0 | 43 |
| pan | 50 | 0 | 50 | 0 | 0 | 0 | 43 |
| ticket | 49 | 3 | 9 | 3 | 1 | 1 | 37 |
| pran | 48 | 17 | 17 | 17 | 0 | 0 | 41 |

### FP reduction vs. TP retention target (thresholds from dev, measured on test)

| rules: TP retention target | achieved retention (test) | FP reduction (test) | precision |
|---|---:|---:|---:|
| 99.9% | 99.92% | 7.6% | 88.1% |
| 99.5% | 99.78% | 61.4% | 94.6% |
| 99.0% | 99.74% | 62.4% | 94.8% |
| 98.0% | 99.41% | 69.7% | 95.7% |
| 95.0% | 99.10% | 70.6% | 95.8% |

| linear: TP retention target | achieved retention (test) | FP reduction (test) | precision |
|---|---:|---:|---:|
| 99.9% | 99.92% | 94.4% | 99.2% |
| 99.5% | 99.74% | 97.2% | 99.6% |
| 99.0% | 99.60% | 97.9% | 99.7% |
| 98.0% | 99.59% | 97.9% | 99.7% |
| 95.0% | 99.54% | 98.1% | 99.7% |

### Keyword gate vs. model gate

The keyword gate also costs recall: values printed without a nearby label (identity-card OCR, bare table cells) are never reported. Scoring every regex+checksum match with the model instead of requiring a keyword recovers some of them.

| candidate set | TP | FP | precision | recall vs. all labelled spans |
|---|---:|---:|---:|---:|
| 0 baseline | 25788 | 3773 | 87.2% | 82.9% |
| A linear | 25685 | 79 | 99.7% | 82.5% |
| regex+checksum only (no gate) | 29601 | 30789 | 49.0% | 95.5% |
| A linear as the gate | 29443 | 2049 | 93.5% | 95.0% |

### Golden set: 48 hand-written documents in styles the generator does not produce

| configuration | TP | FP | precision | TP retention | FP reduction | recall vs. all spans |
|---|---:|---:|---:|---:|---:|---:|
| 0 baseline | 147 | 52 | 73.9% | 100.0% | 0.0% | 94.0% |
| 1 rules @0.5 | 147 | 6 | 96.1% | 100.0% | 88.5% | 94.0% |
| A linear | 147 | 3 | 98.0% | 100.0% | 94.2% | 94.0% |
| A linear as the gate (ungated) | 153 | 6 | 96.2% | – | – | 98.0% |

Every wrong decision (gated candidates):

| doc | claim | value | really | rules | linear | outcome | model explanation |
|---|---|---|---|---:|---:|---|---|
| g007 csv | BANK_ACCOUNT | `440291873659` | aadhaar | 0.70 | 0.003 | fp kept by rules | rules: generic label 'number' governs · model: BANK_ACCOUNT&same_nl:-1.54 same_nl:-1.28 same_sep=newline:-0.90 BANK_ACCOUNT&adj_left=bank_account:+0.84 |
| g016 prose | AADHAAR | `528374619208` | tracking | 0.60 | 0.013 | fp kept by rules | rules: competing label 'awb' (tracking) governs, claim is AADHAAR; but 'aadhaar' attached on the right · model: same_sep=paren:+1.43 AADHAAR&same_sep=paren:+1.16 AADHAAR&l1=awb:-0.84 AADHAAR&fmt=text:-0.78 |
| g018 prose | AADHAAR | `793018462512` | bank_account | 0.60 | 0.003 | fp kept by rules | rules: competing label 'loan account number' (bank_account) governs, claim is AADHAAR; but 'aadhaar' attached on the right · model: same_sep=paren:+1.43 AADHAAR&same_sep=paren:+1.16 AADHAAR&gov=bank_account:-1.15 AADHAAR&gov_kind_sep=bank_account/colon:-1.04 |
| g026 code | PAN | `ABCPE1234F` | dummy | 0.76 | 1.000 | FP kept by both | rules: governing label 'pan' (left) matches claim; test/sample wording nearby · model: l2c=est:-1.03 l2=test:-0.83 gov_same:+0.82 PAN&nvals=1:+0.80 |
| g026 code | MOBILE | `9999999999` | dummy | 0.33 | 0.670 | FP kept by linear | rep=4:-1.63 MOBILE&gov_same:+1.12 MOBILE&gov=mobile:+1.12 l2c=est:-1.03 |
| g026 code | BANK_ACCOUNT | `234567890124` | dummy | 0.33 | 0.963 | FP kept by linear | l2c=est:-1.03 BANK_ACCOUNT&adj_left=bank_account:+0.84 l2=test:-0.83 gov_same:+0.82 |
| g044 prose | AADHAAR | `682739104538` | customer_id | 0.80 | 0.066 | fp kept by rules | rules: governing label 'uid' (left) matches claim · model: l1=uid:-0.96 gov_same:+0.82 AADHAAR&fmt=text:-0.78 AADHAAR&l1=uid:-0.74 |
| g048 prose | BANK_ACCOUNT | `965147382010` | aadhaar | 0.60 | 0.033 | fp kept by rules | rules: no governing label; 'account' within window (d=8) · model: l1=and:+0.76 same_fbv:-0.75 BANK_ACCOUNT&l1=and:-0.74 fmt=text:-0.67 |

Labelled spans the baseline never proposes (no keyword in the window), and whether the model-as-gate recovers them:

- g031 NAME `Suresh Babu` — recovered by model gate
- g031 NAME `Venkata Rao` — recovered by model gate
- g037 AADHAAR `394850172636` — recovered by model gate
- g037 BANK_ACCOUNT `702619483524` — not recovered
- g037 IFSC `BARB0MUMBAI` — recovered by model gate
- g037 AADHAAR `258391047616` — recovered by model gate
- g037 BANK_ACCOUNT `1234567890123` — not recovered
- g037 IFSC `PUNB0123400` — recovered by model gate
- g041 NAME `Kavita` — not recovered

### Cost per candidate (this container: 4 vCPU Xeon @ 2.1 GHz, single goroutine)

| stage | µs / candidate |
|---|---:|
| context analysis (attrib.Analyze, shared) | 24.8 |
| rule verifier | 0.53 |
| linear verifier (features + dot product + explanation) | 42.2 |
| ONNX transformer verifier (tokenize + ONNX Runtime, B onnx) | 2349.8 |

### Remaining false positives after `A linear` (sample)

- `log [AADHAAR truth=user_id 0.98] …G [upi-gateway] session created for uid «910589800587»; aadhar=2517 2542 6807�…`
- `json [MOBILE truth=generic_id 0.99] …976","pensionNo": "590247904996","no": "«09197214949»"}},{"record": {"rmn": "+…`
- `json [AADHAAR truth=customer_id 0.98] … 251630 58977"}},{"customer": {"data": "«971914115080»","aadhaar_no_as_per_card…`
- `prose [NAME truth=no_truth 0.89] …0166865045, Ticket No c9bea55f-d100 and «Contact Person Sushma» Venkatesan.⏎⏎pf no s…`
- `csv [AADHAAR truth=generic_id 0.95] …aadhaar no;userid;dl no⏎Pathak Danish;«479714484008»;842405115139;7479 8338 0…`
- `prose [ADDRESS truth=aadhaar 0.78] …with status 200 in 120ms. Javed Bajpai, «Registration No MH54XE3504, Aadhaar Card No 5979 2676 7061 and PIN Code 579165».⏎⏎⏎Regards,⏎Supp…`
- `json [AADHAAR truth=ppo 0.61] …Nagar, Jaipur - 241505",⏎    "data": "«443578002819»",⏎    "applicant_aadha…`
- `log [AADHAAR truth=bank_account 0.73] …er] {"bill-number":"697655559699","ac":"«235312193188»","AADHAAR_CARD_NO":"8498…`
- `code [AADHAAR truth=dummy 0.95] …": "103656414839",⏎    "aadhaar_no": "«5234 1234 1236»",⏎    "accountnumber":…`
- `code [AADHAAR truth=dummy 0.93] …y Vignesh Qureshi";⏎var AADHAAR_NO = "«9876 5432 1096»";⏎private static final…`
- `ocr [ADDRESS truth=aadhaar 0.54] …ymentId 908958249234, IMEI 8C;4D:D1:AD:E«2.7B and Aadhar Card No. 8974-3946-5287. As discussed on the call today. AadhaarNo copy attached (scanned 891651» pages).⏎Mismatch: Paye…`
- `prose [ADDRESS truth=name 1.00] …n pending⏎⏎Dear Tripathi Alok,⏎⏎«Alok Tripathi - One Time Password 3271, pin_code 143808, Akola Processed 926491» records in 367923 ms.⏎…`
- `prose [ADDRESS truth=name 0.99] …n pending⏎⏎Dear Tripathi Alok,⏎⏎«Alok Tripathi - One Time Password 3271, pin_code 143808, Akola Processed 926491 records in 367923» ms.⏎⏎PF No 103637815…`
- `csv [PINCODE truth=otp 0.46] …zip code,mpin,mobile,comments⏎608648,"«797912»",+91-9409694824,"For UID…`
- `code [AADHAAR truth=dummy 0.82] …kyc:⏎  aadhaar_no: «4234 1234 1235»⏎  mastercard-no: 4253 …`

### Example explanations written to the log by `A linear`

- AADHAAR `549789587091` truth=aadhaar score=0.958 keep=true — gov_same:+0.82 AADHAAR&fmt=text:-0.78 AADHAAR&adj_right=upi_id:+0.69 fmt=text:-0.67
- AADHAAR `3562 5756 4111` truth=aadhaar score=1.000 keep=true — AADHAAR&gov_kind_sep=aadhaar/space:+3.60 AADHAAR&l1=card:+1.64 groups=4-4-4:+1.02 AADHAAR&groups=4-4-4:+1.02
- AADHAAR `544997756509` truth=aadhaar score=0.738 keep=true — gov_same:+0.82 AADHAAR&fmt=text:-0.78 fmt=text:-0.67 groups=12:-0.53
- AADHAAR `478692868930` truth=rrn score=0.001 keep=false — gov_kind_sep=txn_ref/colon:-1.28 l12=ref_no:-1.21 AADHAAR&l12=ref_no:-1.16 AADHAAR&fmt=text:-0.78
- AADHAAR `5702 0591 2494` truth=aadhaar score=0.993 keep=true — groups=4-4-4:+1.02 AADHAAR&groups=4-4-4:+1.02 gov_same:+0.82 AADHAAR&fmt=text:-0.78
- AADHAAR `3531 8567 9123` truth=aadhaar score=1.000 keep=true — AADHAAR&groups=4-4-4:+1.02 groups=4-4-4:+1.02 gov_same:+0.82 AADHAAR&same_side=header:+0.71

_Generated 2026-11-05T03:07:54Z._
