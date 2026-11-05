# Results on the held-out test split

Corpus `corpus.jsonl`: 30000 documents (6000 in test). Window 60 chars. Model `tier_a_lr.json` (version 20261104T170332Z, trained 2026-11-04T17:03:32Z).

Baseline = regex + checksum + same-type keyword within 60 chars (or CSV header). Every verifier only re-scores baseline detections. Thresholds for the ML configurations are chosen on the dev split to retain ≥ 99% of the baseline's true positives per entity type; the rule verifier uses its natural 0.5 threshold.

### Gated candidates (the product's detections)

| configuration | detections kept | TP | FP | precision | TP retention | FP reduction |
|---|---:|---:|---:|---:|---:|---:|
| 0 baseline | 30271 | 26544 | 3727 | 87.7% | 100.00% | 0.0% |
| 1 rules @0.5 | 26994 | 25917 | 1077 | 96.0% | 97.64% | 71.1% |
| 1 rules @99% | 27908 | 26485 | 1423 | 94.9% | 99.78% | 61.8% |
| 1 rules @0.5 +resolve | 26990 | 25916 | 1074 | 96.0% | 97.63% | 71.2% |
| A linear | 26522 | 26430 | 92 | 99.7% | 99.57% | 97.5% |
| A linear +resolve | 26522 | 26430 | 92 | 99.7% | 99.57% | 97.5% |

#### Per entity type (baseline FP → FP after each configuration; TP retention in brackets)

| type | truth spans | baseline TP | baseline recall | baseline FP | baseline precision | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear +resolve |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| AADHAAR | 5789 | 5016 | 86.6% | 679 | 88.1% | 149 (93.5%) | 191 (99.6%) | 148 (93.5%) | 65 (98.7%) | 65 (98.7%) |
| ADDRESS | 1708 | 2481 | 98.3% | 382 | 86.7% | 358 (96.7%) | 382 (100.0%) | 358 (96.7%) | 3 (99.6%) | 3 (99.6%) |
| BANK_ACCOUNT | 2558 | 2196 | 85.8% | 1383 | 61.4% | 191 (99.7%) | 191 (99.7%) | 190 (99.7%) | 9 (99.3%) | 9 (99.3%) |
| CREDIT_CARD | 991 | 851 | 85.9% | 2 | 99.8% | 0 (99.9%) | 0 (99.9%) | 0 (99.9%) | 0 (100.0%) | 0 (100.0%) |
| DOB | 1426 | 1159 | 81.3% | 56 | 95.4% | 41 (99.7%) | 41 (99.7%) | 41 (99.7%) | 0 (99.9%) | 0 (99.9%) |
| DRIVING_LICENCE | 582 | 483 | 83.0% | 0 | 100.0% | 0 (99.8%) | 0 (99.8%) | 0 (99.8%) | 0 (100.0%) | 0 (100.0%) |
| EMAIL | 1396 | 1075 | 77.0% | 0 | 100.0% | 0 (99.8%) | 0 (99.8%) | 0 (99.8%) | 0 (100.0%) | 0 (100.0%) |
| GSTIN | 768 | 630 | 82.0% | 0 | 100.0% | 0 (99.8%) | 0 (99.8%) | 0 (99.8%) | 0 (100.0%) | 0 (100.0%) |
| IFSC | 1109 | 945 | 85.2% | 0 | 100.0% | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| MOBILE | 2323 | 1890 | 81.4% | 62 | 96.8% | 8 (99.5%) | 8 (99.5%) | 8 (99.5%) | 3 (99.7%) | 3 (99.7%) |
| NAME | 5823 | 5146 | 77.0% | 445 | 92.0% | 77 (98.6%) | 356 (100.0%) | 77 (98.6%) | 8 (99.9%) | 8 (99.9%) |
| PAN | 1544 | 1248 | 80.8% | 0 | 100.0% | 0 (95.0%) | 0 (99.6%) | 0 (95.0%) | 0 (100.0%) | 0 (100.0%) |
| PASSPORT | 620 | 474 | 76.5% | 0 | 100.0% | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| PINCODE | 787 | 642 | 81.6% | 376 | 63.1% | 217 (99.4%) | 217 (99.4%) | 217 (99.4%) | 3 (98.1%) | 3 (98.1%) |
| UAN | 1123 | 971 | 86.5% | 339 | 74.1% | 35 (99.8%) | 35 (99.8%) | 34 (99.8%) | 1 (99.7%) | 1 (99.7%) |
| UPI_ID | 809 | 652 | 80.6% | 0 | 100.0% | 0 (99.5%) | 0 (99.5%) | 0 (99.5%) | 0 (100.0%) | 0 (100.0%) |
| VID | 332 | 264 | 79.5% | 2 | 99.2% | 1 (100.0%) | 1 (100.0%) | 1 (100.0%) | 0 (100.0%) | 0 (100.0%) |
| VOTER_ID | 528 | 421 | 79.7% | 1 | 99.8% | 0 (87.2%) | 1 (99.8%) | 0 (87.2%) | 0 (100.0%) | 0 (100.0%) |

#### Per data shape (FP after each configuration; TP retention in brackets)

| shape | baseline TP | baseline FP | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear +resolve |
|---|---:|---:|---:|---:|---:|---:|---:|
| card | 323 | 17 | 6 (93.2%) | 6 (100.0%) | 6 (93.2%) | 0 (100.0%) | 0 (100.0%) |
| code | 787 | 177 | 78 (98.0%) | 86 (99.6%) | 78 (98.0%) | 22 (98.6%) | 22 (98.6%) |
| csv | 10897 | 948 | 368 (97.8%) | 440 (99.7%) | 367 (97.8%) | 37 (99.8%) | 37 (99.8%) |
| form | 4475 | 898 | 145 (97.8%) | 281 (99.9%) | 144 (97.8%) | 4 (99.5%) | 4 (99.5%) |
| json | 2459 | 299 | 84 (98.7%) | 101 (100.0%) | 84 (98.7%) | 7 (99.8%) | 7 (99.8%) |
| log | 1740 | 347 | 155 (97.8%) | 172 (99.9%) | 155 (97.8%) | 11 (99.5%) | 11 (99.5%) |
| ocr | 1909 | 284 | 76 (97.5%) | 120 (99.8%) | 76 (97.5%) | 0 (99.1%) | 0 (99.1%) |
| prose | 3954 | 757 | 165 (96.7%) | 217 (99.7%) | 164 (96.7%) | 11 (99.3%) | 11 (99.3%) |

#### False positives by what the span really was (baseline FP → FP after each configuration)

| span really is | baseline FP | 1 rules @0.5 | 1 rules @99% | 1 rules @0.5 +resolve | A linear | A linear +resolve |
|---|---:|---:|---:|---:|---:|---:|
| address | 520 | 209 | 301 | 209 | 2 | 2 |
| aadhaar | 443 | 77 | 83 | 76 | 3 | 3 |
| no_truth | 311 | 182 | 307 | 182 | 5 | 5 |
| rrn | 277 | 30 | 31 | 30 | 7 | 7 |
| mobile | 212 | 37 | 38 | 36 | 3 | 3 |
| customer_id | 141 | 19 | 20 | 19 | 2 | 2 |
| ppo | 141 | 8 | 10 | 8 | 4 | 4 |
| apaar | 135 | 14 | 14 | 14 | 4 | 4 |
| uan | 125 | 26 | 26 | 26 | 0 | 0 |
| consumer_no | 120 | 17 | 17 | 17 | 1 | 1 |
| txn_id | 105 | 18 | 20 | 18 | 3 | 3 |
| tracking | 103 | 9 | 10 | 9 | 2 | 2 |
| policy | 96 | 16 | 23 | 16 | 2 | 2 |
| bank_account | 88 | 20 | 25 | 19 | 3 | 3 |
| generic_id | 87 | 69 | 74 | 69 | 10 | 10 |
| order_id | 78 | 8 | 20 | 8 | 2 | 2 |
| device_id | 70 | 3 | 12 | 3 | 0 | 0 |
| dummy | 70 | 51 | 51 | 51 | 22 | 22 |
| upi_id | 65 | 7 | 7 | 7 | 0 | 0 |
| user_id | 59 | 36 | 36 | 36 | 11 | 11 |
| invoice | 55 | 25 | 27 | 25 | 0 | 0 |
| pan | 55 | 3 | 55 | 3 | 0 | 0 |
| ticket | 49 | 4 | 12 | 4 | 2 | 2 |
| credit_card | 44 | 15 | 15 | 15 | 0 | 0 |
| pran | 42 | 20 | 20 | 20 | 0 | 0 |

### FP reduction vs. TP retention target (thresholds from dev, measured on test)

| rules: TP retention target | achieved retention (test) | FP reduction (test) | precision |
|---|---:|---:|---:|
| 99.9% | 99.92% | 47.7% | 93.2% |
| 99.5% | 99.80% | 61.5% | 94.9% |
| 99.0% | 99.78% | 61.8% | 94.9% |
| 98.0% | 99.51% | 69.3% | 95.8% |
| 95.0% | 98.99% | 69.9% | 95.9% |

| linear: TP retention target | achieved retention (test) | FP reduction (test) | precision |
|---|---:|---:|---:|
| 99.9% | 99.91% | 93.1% | 99.0% |
| 99.5% | 99.76% | 96.5% | 99.5% |
| 99.0% | 99.57% | 97.5% | 99.7% |
| 98.0% | 99.46% | 97.8% | 99.7% |
| 95.0% | 99.46% | 97.8% | 99.7% |

### Keyword gate vs. model gate

The keyword gate also costs recall: values printed without a nearby label (identity-card OCR, bare table cells) are never reported. Scoring every regex+checksum match with the model instead of requiring a keyword recovers some of them.

| candidate set | TP | FP | precision | recall vs. all labelled spans |
|---|---:|---:|---:|---:|
| 0 baseline | 26544 | 3727 | 87.7% | 83.0% |
| A linear | 26430 | 92 | 99.7% | 82.7% |
| regex+checksum only (no gate) | 30347 | 31143 | 49.4% | 95.6% |
| A linear as the gate | 30198 | 2245 | 93.1% | 95.1% |

### Golden set: 48 hand-written documents in styles the generator does not produce

| configuration | TP | FP | precision | TP retention | FP reduction | recall vs. all spans |
|---|---:|---:|---:|---:|---:|---:|
| 0 baseline | 147 | 52 | 73.9% | 100.0% | 0.0% | 94.0% |
| 1 rules @0.5 | 147 | 7 | 95.5% | 100.0% | 86.5% | 94.0% |
| A linear | 145 | 3 | 98.0% | 98.6% | 94.2% | 92.7% |
| A linear as the gate (ungated) | 151 | 6 | 96.2% | – | – | 96.7% |

Every wrong decision (gated candidates):

| doc | claim | value | really | rules | linear | outcome | model explanation |
|---|---|---|---|---:|---:|---|---|
| g007 csv | BANK_ACCOUNT | `440291873659` | aadhaar | 0.60 | 0.000 | fp kept by rules | rules: generic label 'number' governs · model: BANK_ACCOUNT&same_nl:-1.43 same_nl:-1.22 BANK_ACCOUNT&l1=kumar:-1.00 BANK_ACCOUNT&luhn:-0.67 |
| g016 prose | AADHAAR | `528374619208` | tracking | 0.60 | 0.000 | fp kept by rules | rules: competing label 'awb' (tracking) governs, claim is AADHAAR; but 'aadhaar' attached on the right · model: l1=awb:-1.28 AADHAAR&same_sep=paren:+1.17 l1c=awb:-1.00 AADHAAR&gov_kind_sep=tracking/space:-0.96 |
| g018 prose | AADHAAR | `793018462512` | bank_account | 0.60 | 0.000 | fp kept by rules | rules: competing label 'loan account number' (bank_account) governs, claim is AADHAAR; but 'aadhaar' attached on the right · model: AADHAAR&gov=bank_account:-1.52 AADHAAR&same_sep=paren:+1.17 AADHAAR&gov_kind_sep=bank_account/colon:-1.09 same_sep=paren:+0.86 |
| g025 code | AADHAAR | `840162937518` | aadhaar | 0.97 | 0.293 | MISSED by linear | gov_same:+0.65 AADHAAR&fmt=text:-0.63 fmt=text:-0.57 fmt=text:-0.57 |
| g026 code | PAN | `ABCPE1234F` | dummy | 0.76 | 1.000 | FP kept by both | rules: governing label 'pan' (left) matches claim; test/sample wording nearby · model: PAN&nvals=1:+1.28 l2=test:-0.87 l2c=est:-0.78 PAN&testctx:+0.74 |
| g026 code | MOBILE | `9999999999` | dummy | 0.33 | 0.657 | FP kept by linear | rep=4:-1.58 MOBILE&same_sep=equals:+1.54 MOBILE&gov_kind_sep=mobile/equals:+1.54 MOBILE&l1=mobile:+1.04 |
| g026 code | BANK_ACCOUNT | `234567890124` | dummy | 0.76 | 0.950 | FP kept by both | rules: governing label 'account' (left) matches claim; test/sample wording nearby · model: l2=test:-0.87 l2c=est:-0.78 l1c=acc:+0.67 BANK_ACCOUNT&l1=account:+0.66 |
| g038 prose | MOBILE | `919876543216` | mobile | 0.60 | 0.028 | MISSED by linear | MOBILE&m91:+0.94 MOBILE&len=12:+0.94 m91:-0.77 same_fbv:-0.77 |
| g044 prose | AADHAAR | `682739104538` | customer_id | 0.80 | 0.169 | fp kept by rules | rules: governing label 'uid' (left) matches claim · model: l1=uid:-0.94 AADHAAR&l1=uid:-0.73 gov_same:+0.65 rb=password:+0.63 |
| g048 prose | BANK_ACCOUNT | `965147382010` | aadhaar | 0.60 | 0.055 | fp kept by rules | rules: no governing label; 'account' within window (d=8) · model: same_fbv:-0.77 BANK_ACCOUNT&r2=account:-0.65 BANK_ACCOUNT&adj_right=bank_account:-0.63 BANK_ACCOUNT&len=12:-0.57 |

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
| context analysis (attrib.Analyze, shared) | 28.5 |
| rule verifier | 0.56 |
| linear verifier (features + dot product + explanation) | 40.7 |

### Remaining false positives after `A linear` (sample)

- `csv [AADHAAR truth=rrn 0.35] …84 to CA Number 306286154 completed."⏎«631215403430»,anilbhati@paytm,"5928579…`
- `code [AADHAAR truth=dummy 0.98] …payload = {⏎    "aadhaar_no": "«7234 1234 1230»",⏎    "officeAddress":…`
- `json [AADHAAR truth=rrn 0.69] …ccount_no": "71926347767",⏎    "no": "«315577130705»",⏎    "aadhaar_number_…`
- `json [MOBILE truth=generic_id 0.99] …: "797679","ppo": "902479049966","no": "«09197214949»","status": "PENDING"}},{…`
- `json [AADHAAR truth=customer_id 0.70] …0372552516307"}},{"customer": {"data": "«971914115080»","adharNumber": "7631425…`
- `csv [AADHAAR truth=bank_account 0.47] …tra;23KVDPR4940J2Z5;210909342267;299231;«277112379292»;954069810026;09 November…`
- `csv [PINCODE truth=count 0.68] …; 470498"⏎Anna Mandal,28 January 1993,«110200»,"BARB0309279",9184128316…`
- `prose [BANK_ACCOUNT truth=txn_id 0.99] …ts.⏎⏎0026386854382976 (acct_no) and «33620123631» (Refund ID) submitted.�…`
- `prose [NAME truth=no_truth 0.96] …0166865045, Ticket No c9bea55f-d100 and «Contact Person Sushma» Venkatesan.⏎⏎pf no s…`
- `log [AADHAAR truth=customer_id 0.34] …6253 not found while linking customerid=«974988308882»⏎…`
- `csv [NAME truth=ifsc 0.71] …No,Vpa,Loan Account No,Applicant Name⏎«UCBA»0FY8011,786671668,3149743…`
- `log [AADHAAR truth=bank_account 0.91] …er] {"bill-number":"697655559699","ac":"«235312193188»","AADHAAR_CARD_NO":"8498…`
- `code [AADHAAR truth=dummy 0.91] … "103656414839";⏎String AADHAAR_NO = "«5234 1234 1236»";⏎String ACCOUNTNUMBER…`
- `code [AADHAAR truth=dummy 0.58] …y Vignesh Qureshi";⏎var AADHAAR_NO = "«9876 5432 1096»";⏎private static final…`
- `csv [AADHAAR truth=rrn 0.99] …5;CUST3640518;"7568 6874 5715 (uid) and «454898731222» (UTR No) submitted."⏎0…`

### Example explanations written to the log by `A linear`

- AADHAAR `549789587091` truth=aadhaar score=0.776 keep=true — l2c=uid:+0.72 gov_same:+0.65 AADHAAR&fmt=text:-0.63 fmt=text:-0.57
- AADHAAR `3562 5756 4111` truth=aadhaar score=1.000 keep=true — AADHAAR&gov_kind_sep=aadhaar/space:+2.60 AADHAAR&l1=card:+1.37 gov_kind_sep=aadhaar/space:+1.09 groups=4-4-4:+1.04
- AADHAAR `544997756509` truth=aadhaar score=0.799 keep=true — l2c=uid:+0.72 AADHAAR&comp=pan:+0.72 gov_same:+0.65 AADHAAR&fmt=text:-0.63
- AADHAAR `5702 0591 2494` truth=aadhaar score=1.000 keep=true — AADHAAR&gov_kind_sep=aadhaar/space:+2.60 AADHAAR&l2=aadhar:+2.02 AADHAAR&l12=aadhar_number:+1.27 gov_kind_sep=aadhaar/space:+1.09
- AADHAAR `3531 8567 9123` truth=aadhaar score=1.000 keep=true — AADHAAR&groups=4-4-4:+1.04 groups=4-4-4:+1.04 l2c=652:+0.75 gov_same:+0.65
- AADHAAR `401467251301` truth=aadhaar score=0.888 keep=true — rb=423:-0.66 gov_same:+0.65 AADHAAR&fmt=text:-0.63 fmt=text:-0.57

_Generated 2026-11-04T17:03:51Z._
