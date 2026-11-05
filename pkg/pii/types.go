// Package pii holds the core types shared by the detector, the verifiers,
// the synthetic data generator and the evaluation harness.
package pii

// EntityType is a PII category the product reports. The empty string means
// "not PII" and is used for ground-truth spans of distractor values (order
// IDs, transaction references, timestamps...) that look like PII.
type EntityType string

const (
	Aadhaar        EntityType = "AADHAAR"
	VID            EntityType = "VID"
	PAN            EntityType = "PAN"
	GSTIN          EntityType = "GSTIN"
	CreditCard     EntityType = "CREDIT_CARD"
	VoterID        EntityType = "VOTER_ID"
	Passport       EntityType = "PASSPORT"
	DrivingLicence EntityType = "DRIVING_LICENCE"
	IFSC           EntityType = "IFSC"
	Mobile         EntityType = "MOBILE"
	Email          EntityType = "EMAIL"
	UPI            EntityType = "UPI_ID"
	BankAccount    EntityType = "BANK_ACCOUNT"
	UAN            EntityType = "UAN"
	Pincode        EntityType = "PINCODE"
	DOB            EntityType = "DOB"
	Name           EntityType = "NAME"
	Address        EntityType = "ADDRESS"
)

// AllTypes lists every entity type the baseline detector knows about.
var AllTypes = []EntityType{
	Aadhaar, VID, PAN, GSTIN, CreditCard, VoterID, Passport, DrivingLicence,
	IFSC, Mobile, Email, UPI, BankAccount, UAN, Pincode, DOB, Name, Address,
}

// Format is what the product knows about a file before scanning it, from its
// extension or MIME type. It changes how context is extracted: in a CSV the
// label of a value is its column header, in JSON it is the key.
type Format string

const (
	FormatText Format = "text"
	FormatCSV  Format = "csv"
	FormatJSON Format = "json"
	FormatLog  Format = "log"
)

// TruthSpan is a labelled span in a synthetic document. Type is empty for
// values that are not PII; Kind names the generator that produced the value
// (e.g. "bank_account", "rrn", "order_id") and drives the false-positive
// breakdown in the evaluation.
type TruthSpan struct {
	Start int        `json:"start"`
	End   int        `json:"end"`
	Type  EntityType `json:"type"`
	Kind  string     `json:"kind"`
}

// Doc is one synthetic file.
type Doc struct {
	ID     string      `json:"id"`
	Split  string      `json:"split"` // train, dev, test
	Format Format      `json:"format"`
	Shape  string      `json:"shape"` // prose, form, csv, json, log, code, ocr, card
	Text   string      `json:"text"`
	Truth  []TruthSpan `json:"truth"`
}

// Candidate is one detection emitted by the baseline detector: a regex match
// that passed the entity's structural/checksum validation. KeywordHit tells
// whether the keyword gate (same-type keyword within the window, or in the
// CSV header) passed; the baseline product only reports gated candidates, but
// the verifiers can also score ungated ones.
type Candidate struct {
	DocID       string     `json:"doc_id"`
	Start       int        `json:"start"`
	End         int        `json:"end"`
	Value       string     `json:"value"`
	Type        EntityType `json:"type"`
	ChecksumOK  bool       `json:"checksum_ok"`
	KeywordHit  bool       `json:"keyword_hit"`
	KeywordDist int        `json:"keyword_dist"` // chars from value to nearest same-type keyword, -1 if none
	HeaderHit   bool       `json:"header_hit"`   // CSV only: column header carries a same-type keyword
	Header      string     `json:"header"`       // CSV only: normalised column header
	Column      int        `json:"column"`       // CSV only
}

// Verdict is a verifier's decision about a candidate. Score is in [0,1]
// (probability that the candidate really is the claimed entity type); Keep is
// the thresholded decision; Reason is a short human-readable explanation meant
// for the product's log.
type Verdict struct {
	Score  float64 `json:"score"`
	Keep   bool    `json:"keep"`
	Reason string  `json:"reason"`
}
