// Package attrib analyses the context of one candidate: which label governs
// the value (label-to-value attribution), which competing labels are nearby,
// what the number looks like, and what the structured context (CSV header,
// SQL column, JSON key) says. The rule verifier decides directly from this
// analysis and the ML verifier turns it into features, so both share one
// definition of "context".
package attrib

import (
	"regexp"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/gazetteer"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

// Side says where a label sits relative to the candidate.
type Side string

const (
	Left   Side = "left"
	Right  Side = "right"
	Header Side = "header" // CSV column header
	Key    Side = "key"    // SQL INSERT column name (JSON keys are left labels with a "jsonkey" separator)
)

// LabelRef is a label in the candidate's context.
type LabelRef struct {
	Hit             labels.Hit
	Side            Side
	Dist            int    // characters between label and value (0 for header/key)
	Intervening     int    // other value-like spans between label and candidate
	Separator       string // normalised text between an adjacent label and the value
	Newline         bool   // a line break between label and value
	FollowedByValue bool   // right-side label that itself introduces another value ("..., bank acc 9000...")
}

// Structured reports whether the label comes from a header or key.
func (r *LabelRef) Structured() bool { return r.Side == Header || r.Side == Key }

// Soft reports whether the label is generic or weak.
func (r *LabelRef) Soft() bool { return r.Hit.Kind == labels.KGeneric || r.Hit.Weak }

// NumShape describes a digit string.
type NumShape struct {
	Digits      string
	Len         int
	Groups      string  // e.g. "4-4-4", "12"
	First       byte    // first digit
	RepeatRatio float64 // share of the most frequent digit
	Sequential  bool    // ascending/descending run over the whole string
	SeqRun      int     // longest ascending/descending run of consecutive digits
	RepeatGroup bool    // the same 3- or 4-digit group repeated throughout
	Palindrome  bool
	LeadingZero bool
	Mobile91    bool // 12 digits: 91 + [6-9]xxxxxxxxx
	StartsWith1 bool
	LuhnOK      bool
	VerhoeffOK  bool
}

// Dummy reports a test-data-like digit pattern: one digit repeated, a long
// ascending/descending run (2345678901..), a repeated group (1234 1234 1234)
// or a long palindrome.
func (s *NumShape) Dummy() bool {
	return s.RepeatRatio >= 0.9 || s.Sequential || s.SeqRun >= 8 || s.RepeatGroup || (s.Palindrome && s.Len >= 10)
}

// Analysis is the context of one candidate.
type Analysis struct {
	Window       int
	Left, Right  string
	LeftTokens   []string // nearest first, OCR-normalised, lowercase
	RightTokens  []string
	Labels       []LabelRef
	Governing    *LabelRef // the label attributed to this value, nil if none
	SameNearest  *LabelRef // nearest label of the claimed kind (any side)
	CompNearest  *LabelRef // nearest non-soft label of another kind with no intervening value
	Shape        *NumShape // nil for non-numeric candidates
	Format       pii.Format
	Header       string // CSV header or SQL column, if any
	HeaderKinds  []labels.Kind
	OtherValues  int // value-like spans in the window other than the candidate
	LabelCount   int
	LineLen      int
	LineDigitPct float64
	TestContext  bool // test/sample/dummy/fixture wording in the window
	NameInfo     *NameInfo
}

// NameInfo describes a NAME or ADDRESS candidate span.
type NameInfo struct {
	Words      int
	DictFirst  bool
	DictLast   bool
	DictAny    int
	AllCaps    bool
	Title      bool
	StopWords  int
	AddrCues   int
	HasPIN     bool
	HasCity    bool
	HasState   bool
	HasDigits  bool
	HasComma   bool
	SpanLength int
	NextWord   string // lowercase word right after the span ("road" after a person's name)
}

var (
	valueRe   = regexp.MustCompile(`\d(?:[\d \-/]{2,}\d)?|\b[A-Z0-9]*\d[A-Z0-9]*\b|\S+@\S+`)
	insertRe  = regexp.MustCompile(`(?is)insert\s+into\s+[\w."` + "`" + `\[\]]+\s*\(([^)]*)\)\s*values\s*\(`)
	testWords = map[string]bool{"test": true, "tests": true, "testing": true, "sample": true, "dummy": true, "example": true,
		"fixture": true, "fixtures": true, "mock": true, "fake": true, "demo": true, "placeholder": true, "xxxx": true,
		"tmp": true, "temp": true, "staging": true, "sandbox": true, "lorem": true, "foo": true, "bar": true}
	tightRight = map[string]bool{"paren": true, "dash": true, "colon": true, "verb": true, "hash": true}
)

// Analyze builds the context of a candidate. window is the number of
// characters on each side (the product uses 60).
func Analyze(v *detect.DocView, c *pii.Candidate, window int) *Analysis {
	text := v.Doc.Text
	from, to := max(0, c.Start-window), min(len(text), c.End+window)
	a := &Analysis{Window: window, Left: text[from:c.Start], Right: text[c.End:to], Format: v.Doc.Format}
	kind := labels.TypeKind[c.Type]
	textClaim := c.Type == pii.Name || c.Type == pii.Address

	lt := labels.Tokenize(a.Left)
	for i := len(lt) - 1; i >= 0 && len(a.LeftTokens) < 8; i-- {
		a.LeftTokens = append(a.LeftTokens, lt[i].Norm)
	}
	for _, tk := range labels.Tokenize(a.Right) {
		if len(a.RightTokens) >= 8 {
			break
		}
		a.RightTokens = append(a.RightTokens, tk.Norm)
	}
	for _, t := range append(append([]string{}, a.LeftTokens...), a.RightTokens...) {
		if testWords[t] {
			a.TestContext = true
			break
		}
	}

	// other value-like spans in the window (excluding the candidate itself)
	var values [][2]int
	for _, loc := range valueRe.FindAllStringIndex(text[from:to], -1) {
		s, e := loc[0]+from, loc[1]+from
		if e <= c.Start || s >= c.End {
			if isValueLike(text[s:e]) {
				values = append(values, [2]int{s, e})
			}
		}
	}
	a.OtherValues = len(values)
	between := func(x, y int) int {
		if x > y {
			x, y = y, x
		}
		n := 0
		for _, vs := range values {
			if vs[0] >= x && vs[1] <= y {
				n++
			}
		}
		return n
	}

	for _, h := range v.LabelsIn(from, to) {
		if h.End > c.Start && h.Start < c.End {
			continue // overlaps the value itself
		}
		ref := LabelRef{Hit: h}
		if h.End <= c.Start {
			ref.Side = Left
			ref.Dist = c.Start - h.End
			ref.Intervening = between(h.End, c.Start)
			gap := text[h.End:c.Start]
			ref.Newline = strings.ContainsAny(gap, "\n\r")
			if ref.Intervening == 0 {
				ref.Separator = sepClass(gap)
			}
		} else {
			ref.Side = Right
			ref.Dist = h.Start - c.End
			ref.Intervening = between(c.End, h.Start)
			gap := text[c.End:h.Start]
			ref.Newline = strings.ContainsAny(gap, "\n\r")
			if ref.Intervening == 0 {
				ref.Separator = sepClass(gap)
			}
			for _, vs := range values {
				if vs[0] >= h.End && vs[0]-h.End <= 12 && strings.Trim(text[h.End:vs[0]], " \t:=-#.()/|,;'\"") == "" {
					ref.FollowedByValue = true
					break
				}
			}
		}
		a.Labels = append(a.Labels, ref)
	}
	a.LabelCount = len(a.Labels)

	// structured context: CSV header, else SQL INSERT column
	if v.CSV != nil {
		if _, hdr, ok := v.CSV.ColumnAt(c.Start); ok {
			a.Header = hdr
			for _, h := range labels.Find(hdr) {
				a.HeaderKinds = append(a.HeaderKinds, h.Kind)
				a.Labels = append(a.Labels, LabelRef{Hit: h, Side: Header})
			}
		}
	} else if col, ok := sqlColumnFor(text, c.Start); ok {
		a.Header = col
		for _, h := range labels.Find(col) {
			a.HeaderKinds = append(a.HeaderKinds, h.Kind)
			a.Labels = append(a.Labels, LabelRef{Hit: h, Side: Key})
		}
	}

	a.Governing = governing(a, textClaim)
	for i := range a.Labels {
		ref := &a.Labels[i]
		if textClaim && !textKind(ref.Hit.Kind) {
			continue
		}
		if ref.Hit.Kind == kind {
			if a.SameNearest == nil || rank(ref) < rank(a.SameNearest) {
				a.SameNearest = ref
			}
		} else if !ref.Soft() && ref.Intervening == 0 {
			if a.CompNearest == nil || rank(ref) < rank(a.CompNearest) {
				a.CompNearest = ref
			}
		}
	}

	if numericValue(c.Value) {
		if d := checksum.Digits(c.Value); len(d) >= 4 {
			a.Shape = shapeOf(c.Value, d)
		}
	}

	ls := strings.LastIndexAny(text[:c.Start], "\n\r") + 1
	le := c.End + strings.IndexAny(text[c.End:]+"\n", "\n\r")
	line := text[ls:le]
	a.LineLen = len(line)
	nd := 0
	for i := 0; i < len(line); i++ {
		if line[i] >= '0' && line[i] <= '9' {
			nd++
		}
	}
	if len(line) > 0 {
		a.LineDigitPct = float64(nd) / float64(len(line))
	}
	if textClaim {
		a.NameInfo = nameInfo(text, c)
	}
	return a
}

// isValueLike decides whether a span between a label and a candidate counts
// as another value. Short pure-digit runs (years, counts, "12 digit", error
// codes) are not: they would otherwise break attribution.
func isValueLike(s string) bool {
	if strings.ContainsAny(s, "@") {
		return true
	}
	digits, letters := 0, 0
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digits++
		case s[i] >= 'A' && s[i] <= 'Z' || s[i] >= 'a' && s[i] <= 'z':
			letters++
		}
	}
	if letters > 0 {
		return digits >= 2 && len(s) >= 5 // alphanumeric identifiers
	}
	return digits >= 6 // numeric values; 4-5 digit years/codes/counts are not
}

func textKind(k labels.Kind) bool {
	return k == labels.KName || k == labels.KAddress || k == labels.KGeneric
}

func numericValue(s string) bool {
	digits := 0
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digits++
		case s[i] == ' ' || s[i] == '-' || s[i] == '+':
		default:
			return false
		}
	}
	return digits > 0
}

// rank orders label refs: structured first, then left-adjacent by distance,
// then right-adjacent, then anything with intervening values.
func rank(r *LabelRef) int {
	switch {
	case r.Structured():
		return 0
	case r.Intervening == 0 && r.Side == Left:
		return 10 + r.Dist
	case r.Intervening == 0 && r.Side == Right:
		return 200 + r.Dist
	default:
		return 1000 + 100*r.Intervening + r.Dist
	}
}

// governing picks the label attributed to the value: a header/key if it
// carries a label; else the nearest label on the left with no other value in
// between; else a label immediately to the right joined by a tight separator
// ("9000... (Aadhaar)", "9000... is my Aadhaar"); else nothing. A soft label
// (generic or weak) immediately left is skipped in favour of a specific label
// that directly precedes it ("Permanent Account Number Card ABCPE1234F",
// "Aadhaar Card No. 9000..."). For NAME/ADDRESS claims only name, address and
// generic labels can govern: a numeric-ID label never "governs" a person's
// name that happens to follow it.
func governing(a *Analysis, textClaim bool) *LabelRef {
	eligible := func(r *LabelRef) bool { return !textClaim || textKind(r.Hit.Kind) }
	var best *LabelRef
	for i := range a.Labels {
		r := &a.Labels[i]
		if r.Structured() && eligible(r) && (best == nil || len(r.Hit.Phrase) > len(best.Hit.Phrase)) {
			best = r
		}
	}
	if best != nil {
		return best
	}
	for i := range a.Labels {
		r := &a.Labels[i]
		if r.Side == Left && r.Intervening == 0 && eligible(r) && (best == nil || r.Dist < best.Dist) {
			best = r
		}
	}
	if best != nil && best.Soft() {
		for i := range a.Labels {
			r := &a.Labels[i]
			if r.Side == Left && r.Intervening == 0 && !r.Soft() && eligible(r) && r.Hit.End <= best.Hit.Start && best.Hit.Start-r.Hit.End <= 3 {
				return r
			}
		}
	}
	if best != nil {
		return best
	}
	for i := range a.Labels {
		r := &a.Labels[i]
		if r.Side == Right && r.Intervening == 0 && r.Dist <= 25 && !r.Newline && !r.FollowedByValue && tightRight[r.Separator] && eligible(r) && (best == nil || r.Dist < best.Dist) {
			best = r
		}
	}
	return best
}

// sqlColumnFor maps a value inside "INSERT INTO t (c1, c2, ...) VALUES (...)"
// to its column name by counting top-level commas.
func sqlColumnFor(text string, pos int) (string, bool) {
	start := max(0, pos-4000)
	seg := text[start:pos]
	locs := insertRe.FindAllStringSubmatchIndex(seg, -1)
	if len(locs) == 0 {
		return "", false
	}
	m := locs[len(locs)-1]
	cols := strings.Split(seg[m[2]:m[3]], ",")
	idx, depth := 0, 0
	var inQ byte
	for i := start + m[1]; i < pos; i++ {
		ch := text[i]
		if inQ != 0 {
			if ch == inQ {
				inQ = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inQ = ch
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return "", false // the VALUES list closed before our value
			}
			depth--
		case ',':
			if depth == 0 {
				idx++
			}
		case ';':
			return "", false
		}
	}
	if idx < len(cols) {
		return strings.Trim(strings.TrimSpace(cols[idx]), "`\"[]"), true
	}
	return "", false
}

// sepClass normalises the text between a label and its value.
func sepClass(gap string) string {
	g := strings.TrimSpace(gap)
	if g == "" {
		if strings.Contains(gap, "\t") {
			return "tab"
		}
		return "space"
	}
	gl := strings.ToLower(g)
	for _, w := range []string{"no.", "no", "number", "num", "#", "is", "was", "of", "as", "the", "my", "-", ":", "="} {
		gl = strings.TrimSpace(strings.TrimPrefix(gl, w))
	}
	switch {
	case strings.ContainsAny(g, "\n\r"):
		return "newline"
	case strings.Contains(g, "\":") || strings.Contains(g, "\" :"):
		return "jsonkey"
	case strings.HasPrefix(g, ":"):
		return "colon"
	case strings.HasPrefix(g, "="):
		return "equals"
	case strings.HasPrefix(g, "-") || strings.HasPrefix(g, "–"):
		return "dash"
	case strings.HasPrefix(g, "#"):
		return "hash"
	case strings.HasPrefix(g, "("):
		return "paren"
	case strings.HasPrefix(g, ",") || strings.HasPrefix(g, ";") || strings.HasPrefix(g, "|"):
		return "comma"
	case strings.HasPrefix(gl, "is") || strings.HasPrefix(gl, "as"):
		return "verb"
	case len(gl) == 0:
		return "words"
	case len(gl) > 15:
		return "far"
	}
	return "other"
}

func shapeOf(raw, d string) *NumShape {
	s := &NumShape{Digits: d, Len: len(d), First: d[0]}
	var parts []string
	cur := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] >= '0' && raw[i] <= '9' {
			cur++
		} else if cur > 0 {
			parts = append(parts, itoa(cur))
			cur = 0
		}
	}
	if cur > 0 {
		parts = append(parts, itoa(cur))
	}
	s.Groups = strings.Join(parts, "-")
	counts := [10]int{}
	for i := 0; i < len(d); i++ {
		counts[d[i]-'0']++
	}
	mx := 0
	for _, c := range counts {
		if c > mx {
			mx = c
		}
	}
	s.RepeatRatio = float64(mx) / float64(len(d))
	asc, desc := true, true
	runA, runD, best := 1, 1, 1
	for i := 1; i < len(d); i++ {
		if (int(d[i])-int(d[i-1])+10)%10 == 1 {
			runA++
		} else {
			asc = false
			runA = 1
		}
		if (int(d[i-1])-int(d[i])+10)%10 == 1 {
			runD++
		} else {
			desc = false
			runD = 1
		}
		best = max(best, runA, runD)
	}
	s.Sequential = asc || desc
	s.SeqRun = best
	for _, g := range []int{3, 4} {
		if len(d) >= 2*g && len(d)%g == 0 {
			same := true
			for i := g; i < len(d); i++ {
				if d[i] != d[i%g] {
					same = false
					break
				}
			}
			if same {
				s.RepeatGroup = true
			}
		}
	}
	s.Palindrome = true
	for i := 0; i < len(d)/2; i++ {
		if d[i] != d[len(d)-1-i] {
			s.Palindrome = false
			break
		}
	}
	s.LeadingZero = d[0] == '0'
	s.Mobile91 = len(d) == 12 && d[0] == '9' && d[1] == '1' && d[2] >= '6'
	s.StartsWith1 = d[0] == '1'
	s.LuhnOK = checksum.Luhn(d)
	s.VerhoeffOK = checksum.Verhoeff(d)
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

var pinInside = regexp.MustCompile(`\b[1-9]\d{5}\b`)

func nameInfo(text string, c *pii.Candidate) *NameInfo {
	ni := &NameInfo{SpanLength: c.End - c.Start}
	words := strings.Fields(c.Value)
	ni.Words = len(words)
	if len(words) > 0 {
		ni.DictFirst = gazetteer.IsFirstName(words[0]) || gazetteer.IsSurname(words[0])
		last := words[len(words)-1]
		ni.DictLast = gazetteer.IsSurname(last) || gazetteer.IsFirstName(last)
	}
	ni.AllCaps = true
	for _, w := range words {
		if gazetteer.IsNameWord(w) {
			ni.DictAny++
		}
		if gazetteer.IsNameStop(w) {
			ni.StopWords++
		}
		if strings.ToUpper(w) != w {
			ni.AllCaps = false
		}
		if gazetteer.IsCity(strings.Trim(w, ",")) {
			ni.HasCity = true
		}
	}
	for _, tk := range labels.Tokenize(c.Value) {
		if gazetteer.AddressCues[tk.Text] {
			ni.AddrCues++
		}
	}
	for _, st := range gazetteer.States {
		if strings.Contains(c.Value, st) {
			ni.HasState = true
			break
		}
	}
	ni.HasPIN = pinInside.MatchString(c.Value)
	ni.HasDigits = strings.ContainsAny(c.Value, "0123456789")
	ni.HasComma = strings.Contains(c.Value, ",")
	pre := strings.TrimSpace(text[max(0, c.Start-8):c.Start])
	pre = strings.TrimSuffix(pre, ".")
	if i := strings.LastIndexAny(pre, " \n\t,"); i >= 0 {
		pre = pre[i+1:]
	}
	ni.Title = gazetteer.IsTitle(pre)
	if toks := labels.Tokenize(text[c.End:min(len(text), c.End+20)]); len(toks) > 0 && toks[0].Start <= 2 {
		ni.NextWord = toks[0].Text
	}
	return ni
}
