// Package detect is the baseline detector: regular expressions per entity
// type, structural/checksum validation, and the keyword gate (a same-type
// label within a character window, or in the CSV column header). It
// reproduces the product's current pipeline so that the verifiers can be
// measured against it.
package detect

import (
	"regexp"
	"sort"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

// Options configures the baseline.
type Options struct {
	Window         int              // characters before and after the value searched for a keyword (product: 60)
	RequireKeyword bool             // product behaviour: drop candidates without a keyword hit
	Types          []pii.EntityType // nil = all
}

// DefaultOptions mirrors the product: 60-char window, keyword required.
func DefaultOptions() Options { return Options{Window: 60, RequireKeyword: true} }

// Spec describes how one entity type is matched.
type Spec struct {
	Type     pii.EntityType
	Kind     labels.Kind
	Re       *regexp.Regexp
	Validate func(match string) bool // structural / checksum validation
	Numeric  bool                    // candidate is a digit string: reject when embedded in a longer digit run
	// Reject, when set, drops a match because of what surrounds it.
	Reject func(text string, st, en int) bool
}

// Specs are the regexes and validators of the baseline. They are deliberately
// close to what public DLP tools use for Indian identifiers.
var Specs = []Spec{
	{Type: pii.Aadhaar, Kind: labels.KAadhaar, Numeric: true,
		Re: regexp.MustCompile(`\b\d{4}[ -]?\d{4}[ -]?\d{4}\b`),
		Validate: func(m string) bool {
			d := checksum.Digits(m)
			return len(d) == 12 && d[0] >= '2' && checksum.Verhoeff(d) && consistentSeparators(m)
		}},
	{Type: pii.VID, Kind: labels.KVID, Numeric: true,
		Re: regexp.MustCompile(`\b\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{4}\b`),
		Validate: func(m string) bool {
			d := checksum.Digits(m)
			return len(d) == 16 && d[0] >= '2' && checksum.Verhoeff(d) && consistentSeparators(m)
		}},
	{Type: pii.PAN, Kind: labels.KPAN,
		Re:       regexp.MustCompile(`\b[A-Z]{3}[ABCFGHLJPT][A-Z]\d{4}[A-Z]\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.GSTIN, Kind: labels.KGSTIN,
		Re:       regexp.MustCompile(`\b\d{2}[A-Z]{3}[ABCFGHLJPT][A-Z]\d{4}[A-Z][1-9A-Z]Z[0-9A-Z]\b`),
		Validate: func(m string) bool { return checksum.GSTIN(m) && stateCodeOK(m[:2]) }},
	{Type: pii.CreditCard, Kind: labels.KCreditCard, Numeric: true,
		Re: regexp.MustCompile(`\b\d(?:[ -]?\d){12,18}\b`),
		Validate: func(m string) bool {
			d := checksum.Digits(m)
			return len(d) >= 13 && len(d) <= 19 && checksum.Luhn(d) && cardBIN(d) && consistentSeparators(m)
		}},
	{Type: pii.VoterID, Kind: labels.KVoterID,
		Re:       regexp.MustCompile(`\b[A-Z]{3}\d{7}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.Passport, Kind: labels.KPassport,
		Re:       regexp.MustCompile(`\b[A-PR-WY][1-9]\d{6}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.DrivingLicence, Kind: labels.KDL,
		Re:       regexp.MustCompile(`\b[A-Z]{2}[ -]?\d{2}[ -]?(?:19|20)\d{2}[ -]?\d{7}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.IFSC, Kind: labels.KIFSC,
		Re:       regexp.MustCompile(`\b[A-Z]{4}0[A-Z0-9]{6}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.Mobile, Kind: labels.KMobile, Numeric: true,
		Re: regexp.MustCompile(`(?:\+91[ -]?|\b91[ -]?|\b0)?[6-9]\d{4}[ -]?\d{5}\b`),
		Validate: func(m string) bool {
			d := checksum.Digits(m)
			return len(d) == 10 || (len(d) == 11 && d[0] == '0') || (len(d) == 12 && strings.HasPrefix(d, "91"))
		}},
	{Type: pii.Email, Kind: labels.KEmail,
		Re:       regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.UPI, Kind: labels.KUPI,
		Re:       regexp.MustCompile(`\b[A-Za-z0-9.\-_]{2,}@(?:ok[a-z]{3,}|ybl|ibl|axl|paytm|upi|apl|yapl|rapl|icici|sbi|hdfcbank|axisbank|kotak|airtel|freecharge|jio|fbl|idfcbank|yesbank|barodampay|pnb|unionbank|indus|dbs|rbl|federal|boi|cnrb|aubank|waicici|waaxis|wahdfcbank|wasbi|ptyes|ptaxis|pthdfc|ptsbi)\b`),
		Validate: func(m string) bool { return true },
		// "care@axisbank.com" is an e-mail whose domain starts with a UPI handle
		Reject: func(text string, st, en int) bool {
			return en+1 < len(text) && text[en] == '.' && isAlpha(text[en+1])
		}},
	{Type: pii.BankAccount, Kind: labels.KBankAccount, Numeric: true,
		Re:       regexp.MustCompile(`\b\d{9,18}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.UAN, Kind: labels.KUAN, Numeric: true,
		Re:       regexp.MustCompile(`\b\d{12}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.Pincode, Kind: labels.KPincode, Numeric: true,
		Re:       regexp.MustCompile(`\b[1-9]\d{5}\b`),
		Validate: func(m string) bool { return true }},
	{Type: pii.DOB, Kind: labels.KDOB,
		Re:       regexp.MustCompile(`(?i)\b(?:(?:0?[1-9]|[12]\d|3[01])[/\-.](?:0?[1-9]|1[0-2])[/\-.](?:19|20)\d{2}|(?:19|20)\d{2}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01])|(?:0?[1-9]|[12]\d|3[01])(?:st|nd|rd|th)?[ -](?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*,?[ -](?:19|20)\d{2})\b`),
		Validate: func(m string) bool { return true }},
}

func consistentSeparators(m string) bool {
	space, dash := strings.Count(m, " "), strings.Count(m, "-")
	return space == 0 || dash == 0
}

func stateCodeOK(s string) bool {
	n := int(s[0]-'0')*10 + int(s[1]-'0')
	return n >= 1 && n <= 38 || n == 97 || n == 99
}

// cardBIN accepts the major networks' prefixes (Visa, Mastercard incl. 2-series,
// Amex, RuPay, Discover, Maestro, JCB, Diners).
func cardBIN(d string) bool {
	switch {
	case d[0] == '4':
		return len(d) == 13 || len(d) == 16 || len(d) == 19
	case d[0] == '5' && d[1] >= '1' && d[1] <= '5':
		return len(d) == 16
	case strings.HasPrefix(d, "2"):
		n := atoi(d[:4])
		return n >= 2221 && n <= 2720 && len(d) == 16
	case strings.HasPrefix(d, "34") || strings.HasPrefix(d, "37"):
		return len(d) == 15
	case strings.HasPrefix(d, "60") || strings.HasPrefix(d, "65") || strings.HasPrefix(d, "81") || strings.HasPrefix(d, "82") || strings.HasPrefix(d, "508"):
		return len(d) == 16
	case strings.HasPrefix(d, "35"):
		return len(d) == 16
	case strings.HasPrefix(d, "36") || strings.HasPrefix(d, "38"):
		return len(d) == 14 || len(d) == 16
	case strings.HasPrefix(d, "50") || (d[0] == '5' && d[1] >= '6') || strings.HasPrefix(d, "6"):
		return len(d) >= 12 && len(d) <= 19
	}
	return false
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// Detector runs the baseline.
type Detector struct {
	opts  Options
	specs []Spec
}

// New builds a detector.
func New(opts Options) *Detector {
	d := &Detector{opts: opts}
	want := map[pii.EntityType]bool{}
	for _, t := range opts.Types {
		want[t] = true
	}
	for _, s := range Specs {
		if opts.Types == nil || want[s.Type] {
			d.specs = append(d.specs, s)
		}
	}
	return d
}

// DocView caches per-document analysis shared by the detector and verifiers.
type DocView struct {
	Doc      *pii.Doc
	Tokens   []labels.Token
	Labels   []labels.Hit
	CSV      *CSVLayout // nil unless Format == csv
	maxLabel int        // longest label hit in bytes (LabelsIn's search bound)
}

// NewDocView tokenizes and label-scans a document once.
func NewDocView(doc *pii.Doc) *DocView {
	v := &DocView{Doc: doc}
	v.Tokens = labels.Tokenize(doc.Text)
	v.Labels = labels.FindTokens(v.Tokens)
	for _, h := range v.Labels {
		if n := h.End - h.Start; n > v.maxLabel {
			v.maxLabel = n
		}
	}
	if doc.Format == pii.FormatCSV {
		v.CSV = ParseCSVLayout(doc.Text)
	}
	return v
}

// LabelsIn returns label hits overlapping [from,to).
func (v *DocView) LabelsIn(from, to int) []labels.Hit {
	// Labels are sorted by Start and none is longer than v.maxLabel bytes, so
	// the first hit that can overlap [from,to) starts at or after
	// from-maxLabel; binary search there instead of scanning the document.
	i := sort.Search(len(v.Labels), func(i int) bool { return v.Labels[i].Start >= from-v.maxLabel })
	var out []labels.Hit
	for ; i < len(v.Labels) && v.Labels[i].Start < to; i++ {
		if h := v.Labels[i]; h.End > from {
			out = append(out, h)
		}
	}
	return out
}

// Detect returns the baseline candidates for a document. With
// RequireKeyword=false it returns every validated regex match and sets
// KeywordHit on each, which lets a verifier replace the keyword gate.
func (d *Detector) Detect(v *DocView) []pii.Candidate {
	text := v.Doc.Text
	var out []pii.Candidate
	for _, s := range d.specs {
		for _, loc := range s.Re.FindAllStringIndex(text, -1) {
			st, en := loc[0], loc[1]
			m := text[st:en]
			if st > 0 && isAlnum(text[st-1]) { // manual left boundary (the mobile regex may start with '+')
				continue
			}
			if s.Numeric && embeddedInDigitRun(text, st, en) {
				continue
			}
			if !s.Validate(m) {
				continue
			}
			if s.Reject != nil && s.Reject(text, st, en) {
				continue
			}
			c := pii.Candidate{DocID: v.Doc.ID, Start: st, End: en, Value: m, Type: s.Type, ChecksumOK: true, KeywordDist: -1, Column: -1}
			d.gate(v, s.Kind, &c)
			if d.opts.RequireKeyword && !c.KeywordHit {
				continue
			}
			out = append(out, c)
		}
	}
	out = append(out, d.namesAndAddresses(v)...)
	return out
}

// gate applies the keyword window and the CSV header rule.
func (d *Detector) gate(v *DocView, kind labels.Kind, c *pii.Candidate) {
	from, to := c.Start-d.opts.Window, c.End+d.opts.Window
	for _, h := range v.LabelsIn(from, to) {
		if h.Kind != kind {
			continue
		}
		dist := 0
		if h.End <= c.Start {
			dist = c.Start - h.End
		} else if h.Start >= c.End {
			dist = h.Start - c.End
		}
		if !c.KeywordHit || dist < c.KeywordDist {
			c.KeywordDist = dist
		}
		c.KeywordHit = true
	}
	if v.CSV != nil {
		if col, hdr, ok := v.CSV.ColumnAt(c.Start); ok {
			c.Column, c.Header = col, hdr
			for _, h := range labels.Find(hdr) {
				if h.Kind == kind {
					c.HeaderHit = true
					c.KeywordHit = true
					if c.KeywordDist < 0 {
						c.KeywordDist = 0
					}
				}
			}
		}
	}
}

func isAlnum(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func isAlpha(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

// digitRunLen counts consecutive digits ending at i (inclusive) when back is
// true, or starting at i when back is false.
func digitRunLen(text string, i int, back bool) int {
	n := 0
	for i >= 0 && i < len(text) && isDigit(text[i]) {
		n++
		if back {
			i--
		} else {
			i++
		}
	}
	return n
}

// embeddedInDigitRun reports whether the match is a slice of a longer
// number written with single separators ("9000 0000 0002 1234" contains an
// Aadhaar-shaped prefix).
func embeddedInDigitRun(text string, st, en int) bool {
	// A neighbouring group of >= 3 digits across a single separator means the
	// match is a slice of a longer grouped number; a lone "10 " ("Sl No 10
	// 9876543210") or "2 " is not.
	sep := func(b byte) bool { return b == ' ' || b == '-' || b == '/' }
	if st >= 2 && sep(text[st-1]) && digitRunLen(text, st-2, true) >= 3 {
		return true
	}
	if en+1 < len(text) && sep(text[en]) && digitRunLen(text, en+1, false) >= 3 {
		return true
	}
	return false
}
