package synth

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii"
)

// baseTime anchors every generated timestamp so that generation is
// deterministic for a given seed.
var baseTime = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

// Builder accumulates text and truth spans.
type Builder struct {
	sb    strings.Builder
	truth []pii.TruthSpan
	vals  []string // the exact text written for each truth span (tests)
}

func (b *Builder) S(s string) { b.sb.WriteString(s) }

// V writes a value and records it as a truth span of the field's type/kind.
// An address also gets a nested PINCODE span: the PIN code inside it is PII
// in its own right, and a PINCODE detection on it must count as correct.
func (b *Builder) V(val string, typ pii.EntityType, kind string) {
	st := b.sb.Len()
	b.sb.WriteString(val)
	b.truth = append(b.truth, pii.TruthSpan{Start: st, End: b.sb.Len(), Type: typ, Kind: kind})
	b.vals = append(b.vals, val)
	if typ == pii.Address {
		if loc := pinInAddress.FindStringIndex(val); loc != nil {
			b.truth = append(b.truth, pii.TruthSpan{Start: st + loc[0], End: st + loc[1], Type: pii.Pincode, Kind: "pincode"})
			b.vals = append(b.vals, val[loc[0]:loc[1]])
		}
	}
}

var pinInAddress = regexp.MustCompile(`\b[1-9]\d{5}\b`)

func (b *Builder) Len() int { return b.sb.Len() }

// Inst is a field with a chosen label and value.
type Inst struct {
	F     *Field
	Label string
	Value string
}

// ctx carries per-document state.
type ctx struct {
	r     *rand.Rand
	split string
	p     *Person
	b     *Builder
	shape string
}

func (c *ctx) label(f *Field) string {
	l := pickLabel(c.r, f, c.split)
	switch c.r.Intn(10) {
	case 0, 1:
		return strings.ToLower(l)
	case 2:
		return strings.ToUpper(l)
	default:
		return l
	}
}

func (c *ctx) inst(f *Field) Inst { return Inst{F: f, Label: c.label(f), Value: f.Gen(c.r, c.p)} }

func (c *ctx) instFor(p *Person, f *Field) Inst {
	return Inst{F: f, Label: c.label(f), Value: f.Gen(c.r, p)}
}

func (c *ctx) randomDate() string {
	t := baseTime.Add(-time.Duration(c.r.Intn(2000)) * 24 * time.Hour)
	switch c.r.Intn(4) {
	case 0:
		return t.Format("02/01/2006")
	case 1:
		return t.Format("2 Jan 2006")
	case 2:
		return t.Format("2006-01-02")
	default:
		return t.Format("02-01-2006")
	}
}

// fill renders a template with the given field instances.
func (c *ctx) fill(tpl string, insts []Inst) {
	b := c.b
	i := 0
	for i < len(tpl) {
		if tpl[i] != '{' {
			j := strings.IndexByte(tpl[i:], '{')
			if j < 0 {
				b.S(tpl[i:])
				return
			}
			b.S(tpl[i : i+j])
			i += j
			continue
		}
		k := strings.IndexByte(tpl[i:], '}')
		if k < 0 {
			b.S(tpl[i:])
			return
		}
		tok := tpl[i+1 : i+k]
		i += k + 1
		switch {
		case len(tok) == 2 && (tok[0] == 'L' || tok[0] == 'K') && tok[1] >= '1' && tok[1] <= '3':
			n := int(tok[1] - '1')
			if n < len(insts) {
				if tok[0] == 'K' {
					b.S(identifier(c.r, insts[n].Label))
				} else {
					b.S(insts[n].Label)
				}
			}
		case len(tok) == 2 && tok[0] == 'V' && tok[1] >= '1' && tok[1] <= '3':
			n := int(tok[1] - '1')
			if n < len(insts) {
				b.V(insts[n].Value, insts[n].F.Type, insts[n].F.Kind)
			}
		case len(tok) == 3 && tok[0] == 'V' && tok[1] >= '1' && tok[1] <= '3' && (tok[2] == 'b' || tok[2] == 'c'):
			// another value of the same field (a second person / another record)
			n := int(tok[1] - '1')
			if n < len(insts) {
				other := NewPerson(c.r)
				b.V(insts[n].F.Gen(c.r, other), insts[n].F.Type, insts[n].F.Kind)
			}
		case tok == "NAME":
			b.V(c.p.nameFmt(c.r), pii.Name, "name")
		case tok == "CITY":
			b.S(c.p.City)
		case tok == "BANK":
			b.S(c.p.BankName)
		case tok == "COMPANY":
			b.S(c.p.Company)
		case tok == "DATE":
			b.V(c.randomDate(), "", "date")
		case strings.HasPrefix(tok, "D:"):
			f := FieldByKind[tok[2:]]
			if f != nil {
				b.V(f.Gen(c.r, c.p), f.Type, f.Kind)
			}
		default:
			b.S("{" + tok + "}")
		}
	}
}

// identifier turns a label into a code identifier (snake_case, camelCase...).
func identifier(r *rand.Rand, label string) string {
	words := strings.FieldsFunc(strings.ToLower(label), func(c rune) bool {
		return !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9')
	})
	if len(words) == 0 {
		return "field"
	}
	switch r.Intn(6) {
	case 0, 1:
		return strings.Join(words, "_")
	case 2:
		out := words[0]
		for _, w := range words[1:] {
			out += strings.ToUpper(w[:1]) + w[1:]
		}
		return out
	case 3:
		return strings.ToUpper(strings.Join(words, "_"))
	case 4:
		return strings.Join(words, "")
	default:
		return strings.Join(words, "-")
	}
}

// headerStyle renders a column header from a label in one consistent style per CSV.
func headerStyle(style int, label string) string {
	words := strings.FieldsFunc(strings.ToLower(label), func(c rune) bool {
		return !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9')
	})
	if len(words) == 0 {
		return label
	}
	switch style {
	case 0:
		return strings.Join(words, "_")
	case 1:
		return strings.Join(words, " ")
	case 2:
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
		return strings.Join(words, " ")
	case 3:
		return strings.ToUpper(strings.Join(words, "_"))
	case 4:
		out := words[0]
		for _, w := range words[1:] {
			out += strings.ToUpper(w[:1]) + w[1:]
		}
		return out
	default:
		return label
	}
}

// ---- field selection -----------------------------------------------------

var piiFields, disFields []*Field

func init() {
	for _, f := range Fields {
		if f.Type != "" {
			piiFields = append(piiFields, f)
		} else if f.Kind != "dummy" {
			disFields = append(disFields, f)
		}
	}
}

func weightedPick(r *rand.Rand, fs []*Field, used map[string]bool) *Field {
	total := 0
	for _, f := range fs {
		if !used[f.Kind] {
			total += f.Weight
		}
	}
	if total == 0 {
		return nil
	}
	x := r.Intn(total)
	for _, f := range fs {
		if used[f.Kind] {
			continue
		}
		x -= f.Weight
		if x < 0 {
			return f
		}
	}
	return nil
}

var colliders = []string{"bank_account", "rrn", "customer_id", "ppo", "apaar", "consumer_no", "mobile", "uan", "tracking", "order_id", "txn_id", "policy", "generic_id", "device_id"}

// chooseFields picks the PII and distractor fields for a document.
func (c *ctx) chooseFields(nPII, nDis int, forceAadhaar bool) []*Field {
	used := map[string]bool{}
	var out []*Field
	if forceAadhaar || c.r.Float64() < 0.45 {
		out = append(out, FieldByKind["aadhaar"])
		used["aadhaar"] = true
		nPII--
	}
	for i := 0; i < nPII; i++ {
		if f := weightedPick(c.r, piiFields, used); f != nil {
			out = append(out, f)
			used[f.Kind] = true
		}
	}
	if c.r.Float64() < 0.5 { // a collider next to the PII
		k := colliders[c.r.Intn(len(colliders))]
		if !used[k] {
			out = append(out, FieldByKind[k])
			used[k] = true
			nDis--
		}
	}
	for i := 0; i < nDis; i++ {
		if f := weightedPick(c.r, disFields, used); f != nil {
			out = append(out, f)
			used[f.Kind] = true
		}
	}
	c.r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (c *ctx) insts(fs []*Field) []Inst {
	out := make([]Inst, len(fs))
	for i, f := range fs {
		out[i] = c.inst(f)
	}
	return out
}

// ---- shapes --------------------------------------------------------------

func (c *ctx) prose() {
	b := c.b
	email := c.r.Intn(2) == 0
	if email {
		if c.r.Intn(2) == 0 {
			c.fill(pickPooled(c.r, subjects, c.split), nil)
			b.S("\n\n")
		}
		c.fill(pickPooled(c.r, emailOpeners, c.split), nil)
		b.S("\n\n")
	}
	nSent := 1 + c.r.Intn(4)
	for s := 0; s < nSent; s++ {
		switch c.r.Intn(10) {
		case 0, 1, 2:
			c.fill(pickPooled(c.r, filler, c.split), nil)
		case 3, 4:
			fs := c.chooseFields(1+c.r.Intn(2), 0, false)
			c.fill(pickPooled(c.r, keywordOnly, c.split), c.insts(fs))
		default:
			fs := c.chooseFields(1+c.r.Intn(2), c.r.Intn(3), false)
			var tpl string
			switch len(fs) {
			case 0:
				continue
			case 1:
				tpl = pickPooled(c.r, prose1, c.split)
				if fs[0].Kind == "mobile" && c.r.Intn(3) == 0 {
					tpl = pickPooled(c.r, mobileCtx, c.split)
				}
			case 2:
				tpl = pickPooled(c.r, prose2, c.split)
				if c.r.Intn(5) == 0 {
					tpl = pickPooled(c.r, hardLayouts2, c.split)
				}
			default:
				tpl = pickPooled(c.r, prose3, c.split)
				if c.r.Intn(5) == 0 {
					tpl = pickPooled(c.r, hardLayouts3, c.split)
				}
				if len(fs) > 3 {
					fs = fs[:3]
				}
			}
			c.fill(tpl, c.insts(fs))
		}
		b.S(pick(c.r, []string{" ", " ", "\n", "\n\n", " "}))
	}
	if email {
		b.S("\n")
		c.fill(pickPooled(c.r, emailClosers, c.split), nil)
	}
}

func (c *ctx) form() {
	b := c.b
	if c.r.Intn(2) == 0 {
		b.S(pick(c.r, sectionHeaders) + "\n")
	}
	fs := c.chooseFields(2+c.r.Intn(4), c.r.Intn(4), false)
	insts := c.insts(fs)
	if c.r.Intn(7) == 0 { // aligned table: a row of labels, then a row of values
		colSep := pick(c.r, []string{"\t", "  |  ", "    ", " | "})
		for i, in := range insts {
			if i > 0 {
				b.S(colSep)
			}
			b.S(in.Label)
		}
		b.S("\n")
		for i, in := range insts {
			if i > 0 {
				b.S(colSep)
			}
			b.V(in.Value, in.F.Type, in.F.Kind)
		}
		b.S("\n")
		return
	}
	sep := pick(c.r, []string{": ", ": ", " : ", ":", " - ", " – ", "  ", "\t", " = ", ": \t", "#", " "})
	joiner := pick(c.r, []string{"\n", "\n", "\n", " | ", ", ", "    ", "; ", "  ", "\n"})
	for i, in := range insts {
		if i > 0 {
			if c.r.Intn(4) == 0 && i%3 == 0 {
				b.S("\n" + pick(c.r, sectionHeaders) + "\n")
			} else {
				b.S(joiner)
			}
		}
		b.S(in.Label)
		if c.r.Intn(8) == 0 { // label on one line, value on the next
			b.S("\n")
		} else {
			b.S(sep)
		}
		b.V(in.Value, in.F.Type, in.F.Kind)
	}
	if c.r.Intn(3) == 0 {
		b.S("\n")
		c.fill(pickPooled(c.r, filler, c.split), nil)
	}
}

func (c *ctx) csv() {
	b := c.b
	delim := pick(c.r, []string{",", ",", ",", "\t", ";", "|"})
	style := c.r.Intn(6)
	fs := c.chooseFields(2+c.r.Intn(4), c.r.Intn(4), false)
	// a name column most of the time
	hasName := false
	for _, f := range fs {
		if f.Kind == "name" {
			hasName = true
		}
	}
	if !hasName && c.r.Intn(3) > 0 {
		fs = append([]*Field{FieldByKind["name"]}, fs...)
	}
	labelsFor := make([]string, len(fs))
	idTypeCol := -1 // "id_type,id_number" pairs: the type name sits in the cell before the value
	for i, f := range fs {
		labelsFor[i] = headerStyle(style, pickLabel(c.r, f, c.split))
		if c.r.Intn(6) == 0 { // ambiguous header
			labelsFor[i] = pick(c.r, []string{"id", "number", "ref", "value", fmt.Sprintf("col%d", i+1), fmt.Sprintf("field_%d", i+1), "no", "data", "identifier"})
		}
		if idTypeCol < 0 && c.r.Intn(4) == 0 && (f.Kind == "aadhaar" || f.Kind == "pan" || f.Kind == "voter_id" || f.Kind == "passport" || f.Kind == "driving_licence") {
			idTypeCol = i
			labelsFor[i] = headerStyle(style, pick(c.r, []string{"ID Number", "Document No", "KYC Number", "Proof Number", "ID Value"}))
		}
	}
	idTypeName := func(f *Field) string {
		switch f.Kind {
		case "aadhaar":
			return pick(c.r, []string{"AADHAAR", "Aadhaar", "AADHAR", "UID"})
		case "pan":
			return pick(c.r, []string{"PAN", "Pan Card"})
		case "voter_id":
			return pick(c.r, []string{"VOTER ID", "EPIC", "Voter Id"})
		case "passport":
			return pick(c.r, []string{"PASSPORT", "Passport"})
		default:
			return pick(c.r, []string{"DL", "DRIVING LICENCE", "Driving License"})
		}
	}
	remarks := c.r.Intn(3) == 0
	if c.r.Intn(10) > 0 { // header row (sometimes missing)
		hdr := append([]string{}, labelsFor...)
		if idTypeCol >= 0 {
			hdr = append(hdr[:idTypeCol], append([]string{headerStyle(style, pick(c.r, []string{"ID Type", "Document Type", "KYC Type", "Proof Type"}))}, hdr[idTypeCol:]...)...)
		}
		b.S(strings.Join(hdr, delim))
		if remarks {
			b.S(delim + pick(c.r, []string{"remarks", "notes", "description", "comments", "Remarks"}))
		}
		b.S("\n")
	}
	rows := 1 + c.r.Intn(6)
	for row := 0; row < rows; row++ {
		p := c.p
		if row > 0 {
			p = NewPerson(c.r)
		}
		for i, f := range fs {
			if i > 0 {
				b.S(delim)
			}
			if i == idTypeCol {
				b.S(idTypeName(f) + delim)
			}
			v := f.Gen(c.r, p)
			quoted := strings.Contains(v, delim) || (c.r.Intn(5) == 0 && delim == ",")
			if quoted {
				b.S(`"`)
			}
			b.V(v, f.Type, f.Kind)
			if quoted {
				b.S(`"`)
			}
		}
		if remarks {
			b.S(delim + `"`)
			old := c.p
			c.p = p
			if c.r.Intn(2) == 0 {
				rf := c.chooseFields(1, c.r.Intn(2), false)
				tpl := prose1
				if len(rf) >= 2 {
					tpl = prose2
				}
				c.fill(pickPooled(c.r, tpl, c.split), c.insts(rf))
			} else {
				c.fill(pickPooled(c.r, filler, c.split), nil)
			}
			c.p = old
			b.S(`"`)
		}
		b.S("\n")
	}
}

func (c *ctx) jsonDoc() {
	b := c.b
	fs := c.chooseFields(2+c.r.Intn(4), c.r.Intn(4), false)
	pretty := c.r.Intn(3) > 0
	nl, ind := "", ""
	if pretty {
		nl, ind = "\n", "  "
	}
	wrapKey := pick(c.r, []string{"", "", "customer", "kyc", "data", "payload", "applicant", "record", "user", "beneficiary"})
	n := 1
	if c.r.Intn(4) == 0 {
		n = 2 + c.r.Intn(2)
	}
	depth := 0
	open := func(s string) { b.S(strings.Repeat(ind, depth) + s + nl); depth++ }
	closeB := func(s string) { depth--; b.S(strings.Repeat(ind, depth) + s) }
	if n > 1 {
		open("[")
	}
	for k := 0; k < n; k++ {
		p := c.p
		if k > 0 {
			p = NewPerson(c.r)
		}
		open("{")
		if wrapKey != "" {
			b.S(strings.Repeat(ind, depth) + `"` + wrapKey + `": `)
			b.S("{" + nl)
			depth++
		}
		meta := c.r.Intn(2) == 0
		for i, f := range fs {
			key := identifier(c.r, pickLabel(c.r, f, c.split))
			if c.r.Intn(8) == 0 {
				key = pick(c.r, []string{"id", "value", "number", "ref", "no", "data"})
			}
			v := f.Gen(c.r, p)
			b.S(strings.Repeat(ind, depth) + `"` + key + `": `)
			numeric := isAllDigits(v) && c.r.Intn(3) == 0
			if !numeric {
				b.S(`"`)
			}
			b.V(v, f.Type, f.Kind)
			if !numeric {
				b.S(`"`)
			}
			if i < len(fs)-1 || meta {
				b.S(",")
			}
			b.S(nl)
		}
		if meta {
			b.S(strings.Repeat(ind, depth) + `"status": "` + pick(c.r, []string{"OK", "PENDING", "VERIFIED", "FAILED"}) + `"` + nl)
		}
		if wrapKey != "" {
			closeB("}")
			b.S(nl)
		}
		closeB("}")
		if k < n-1 {
			b.S(",")
		}
		b.S(nl)
	}
	if n > 1 {
		closeB("]")
	}
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (c *ctx) logDoc() {
	b := c.b
	lines := 1 + c.r.Intn(5)
	svc := pick(c.r, services)
	t := baseTime.Add(-time.Duration(c.r.Intn(100000)) * time.Second)
	for i := 0; i < lines; i++ {
		t = t.Add(time.Duration(c.r.Intn(5000)) * time.Millisecond)
		switch c.r.Intn(4) {
		case 0:
			b.S(t.Format("2006-01-02T15:04:05.000Z") + " ")
		case 1:
			b.S(t.Format("2006-01-02 15:04:05,000") + " ")
		case 2:
			b.S(fmt.Sprintf("[%s] ", t.Format("02/Jan/2006:15:04:05 -0700")))
		default:
			b.S(fmt.Sprint(t.UnixMilli()) + " ")
		}
		b.S(pick(c.r, logLevels) + " ")
		if c.r.Intn(2) == 0 {
			b.S("[" + svc + "] ")
		} else {
			b.S(svc + " - ")
		}
		if c.r.Intn(5) == 0 {
			b.S(fmt.Sprintf("[req-%08x] ", c.r.Uint32()))
		}
		switch c.r.Intn(6) {
		case 0:
			c.fill(pickPooled(c.r, filler, c.split), nil)
		case 1:
			// JSON payload inside the log line
			old := c.shape
			c.shape = "log"
			c.jsonLine()
			c.shape = old
		default:
			fs := c.chooseFields(1+c.r.Intn(2), c.r.Intn(3), false)
			if len(fs) > 3 {
				fs = fs[:3]
			}
			insts := c.insts(fs)
			tpl := pickPooled(c.r, logMessages, c.split)
			c.fill(tpl, insts)
		}
		b.S("\n")
	}
}

// jsonLine writes a compact JSON object (used inside log lines).
func (c *ctx) jsonLine() {
	b := c.b
	fs := c.chooseFields(1+c.r.Intn(3), c.r.Intn(3), false)
	b.S("{")
	for i, f := range fs {
		if i > 0 {
			b.S(",")
		}
		b.S(`"` + identifier(c.r, pickLabel(c.r, f, c.split)) + `":`)
		v := f.Gen(c.r, c.p)
		if isAllDigits(v) && c.r.Intn(3) == 0 {
			b.V(v, f.Type, f.Kind)
		} else {
			b.S(`"`)
			b.V(v, f.Type, f.Kind)
			b.S(`"`)
		}
	}
	b.S("}")
}

func (c *ctx) code() {
	b := c.b
	fs := c.chooseFields(2+c.r.Intn(3), c.r.Intn(3), false)
	// in fixtures, Aadhaar-like values are often dummies
	dummy := c.r.Intn(5) < 2
	if dummy {
		for i, f := range fs {
			if f.Kind == "aadhaar" {
				fs[i] = FieldByKind["dummy"]
			}
		}
	}
	keys := make([]string, len(fs))
	vals := make([]string, len(fs))
	for i, f := range fs {
		keys[i] = identifier(c.r, pickLabel(c.r, f, c.split))
		vals[i] = f.Gen(c.r, c.p)
	}
	switch c.r.Intn(6) {
	case 0: // YAML
		b.S(pick(c.r, []string{"customer:\n", "test_user:\n", "fixture:\n", "kyc:\n", ""}))
		for i, f := range fs {
			b.S("  " + keys[i] + ": ")
			q := c.r.Intn(2) == 0
			if q {
				b.S(`"`)
			}
			b.V(vals[i], f.Type, f.Kind)
			if q {
				b.S(`"`)
			}
			b.S("\n")
		}
	case 1: // Python dict
		b.S(pick(c.r, []string{"payload = {", "TEST_CUSTOMER = {", "data = {", "row = {"}) + "\n")
		for i, f := range fs {
			b.S(`    "` + keys[i] + `": "`)
			b.V(vals[i], f.Type, f.Kind)
			b.S(`",` + "\n")
		}
		b.S("}\n")
	case 2: // SQL
		b.S("INSERT INTO " + pick(c.r, []string{"customers", "kyc_records", "applicants", "accounts", "tmp_import"}) + " (" + strings.Join(keys, ", ") + ") VALUES (")
		for i, f := range fs {
			if i > 0 {
				b.S(", ")
			}
			b.S("'")
			b.V(vals[i], f.Type, f.Kind)
			b.S("'")
		}
		b.S(");\n")
	case 3: // properties / env
		for i, f := range fs {
			b.S(strings.ToUpper(keys[i]) + "=")
			b.V(vals[i], f.Type, f.Kind)
			b.S("\n")
		}
	case 4: // Java/Go constants
		for i, f := range fs {
			b.S(pick(c.r, []string{"private static final String ", "const ", "var ", "String "}) + strings.ToUpper(keys[i]) + ` = "`)
			b.V(vals[i], f.Type, f.Kind)
			b.S(`";` + "\n")
		}
	default: // curl / HTTP
		b.S("curl -X POST https://api.example.in/v1/kyc -d '{")
		for i, f := range fs {
			if i > 0 {
				b.S(",")
			}
			b.S(`"` + keys[i] + `":"`)
			b.V(vals[i], f.Type, f.Kind)
			b.S(`"`)
		}
		b.S("}'\n")
	}
}

// card renders OCR-like text of identity documents and bank passbooks. Real
// Aadhaar cards do not print the word "Aadhaar" next to the number, which is
// why the keyword gate misses them.
func (c *ctx) card() {
	b := c.b
	p := c.p
	switch c.r.Intn(4) {
	case 0: // Aadhaar front
		b.S(pick(c.r, []string{"Government of India\n", "GOVERNMENT OF INDIA\n", "भारत सरकार\nGovernment of India\n"}))
		if c.r.Intn(2) == 0 {
			b.S("Unique Identification Authority of India\n")
		}
		b.V(p.Full, pii.Name, "name")
		b.S("\n")
		if c.r.Intn(2) == 0 {
			b.S(pick(c.r, []string{"S/O: ", "D/O: ", "C/O: ", "W/O: "}))
			b.V(p.Father, pii.Name, "father_name")
			b.S("\n")
		}
		b.S(pick(c.r, []string{"DOB: ", "Date of Birth: ", "DOB : ", "Year of Birth: "}))
		b.V(p.dobFmt(c.r), pii.DOB, "dob")
		b.S("\n" + strings.ToUpper(p.Gender) + "\n\n")
		b.V(groupDigits(p.Aadhaar, " ", 4), pii.Aadhaar, "aadhaar")
		b.S("\n")
		if c.r.Intn(2) == 0 {
			b.S("VID : ")
			b.V(groupDigits(p.VID, " ", 4), pii.VID, "vid")
			b.S("\n")
		}
		b.S(pick(c.r, []string{"Mera Aadhaar, Meri Pehchaan\n", "मेरा आधार, मेरी पहचान\n", "Aadhaar - Aam Aadmi ka Adhikar\n", ""}))
	case 1: // Aadhaar back
		b.S("Unique Identification Authority of India\nAddress:\n")
		b.S(pick(c.r, []string{"S/O: ", "D/O: ", "W/O: "}))
		b.V(p.Father, pii.Name, "father_name")
		b.S(", ")
		b.V(p.addressFmt(c.r), pii.Address, "address")
		b.S("\n\n")
		b.V(groupDigits(p.Aadhaar, " ", 4), pii.Aadhaar, "aadhaar")
		b.S("\n")
		if c.r.Intn(2) == 0 {
			b.S("VID : ")
			b.V(groupDigits(p.VID, " ", 4), pii.VID, "vid")
			b.S("\n")
		}
		b.S("help@uidai.gov.in  www.uidai.gov.in  1947\n")
	case 2: // PAN card
		b.S("INCOME TAX DEPARTMENT\nGOVT. OF INDIA\n")
		if c.r.Intn(2) == 0 {
			b.S("Permanent Account Number Card\n")
		}
		b.V(p.PAN, pii.PAN, "pan")
		b.S("\n" + pick(c.r, []string{"Name\n", "NAME\n", ""}))
		b.V(strings.ToUpper(p.Full), pii.Name, "name")
		b.S("\n" + pick(c.r, []string{"Father's Name\n", "FATHER'S NAME\n"}))
		b.V(strings.ToUpper(p.Father), pii.Name, "father_name")
		b.S("\n" + pick(c.r, []string{"Date of Birth\n", "DATE OF BIRTH\n"}))
		b.V(p.DOB.Format("02/01/2006"), pii.DOB, "dob")
		b.S("\nSignature\n")
	default: // bank passbook / statement header
		b.S(strings.ToUpper(p.BankName) + "\n" + pick(c.r, []string{"Savings Account Statement\n", "PASSBOOK\n", "Statement of Account\n"}))
		b.S(pick(c.r, []string{"Account Holder: ", "Name: ", "Customer Name : "}))
		b.V(p.Full, pii.Name, "name")
		b.S("\n" + pick(c.r, []string{"Account No: ", "A/c No. ", "Account Number : "}))
		b.V(p.Account, pii.BankAccount, "bank_account")
		b.S("\nIFSC: ")
		b.V(p.IFSC, pii.IFSC, "ifsc")
		b.S("\n" + pick(c.r, []string{"CIF No: ", "Customer ID: ", "CRN: "}))
		b.V(p.CustomerID, "", "customer_id")
		b.S("\nBranch: " + p.City + "\n")
		if c.r.Intn(2) == 0 {
			b.S("Aadhaar Seeded: " + pick(c.r, []string{"Yes", "No", "YES"}) + "\n")
		}
		b.S("Address: ")
		b.V(p.addressFmt(c.r), pii.Address, "address")
		b.S("\n\nDate        Particulars                          Ref No         Debit      Credit     Balance\n")
		for i := 0; i < 1+c.r.Intn(4); i++ {
			b.V(c.randomDate(), "", "date")
			b.S("  " + pick(c.r, []string{"UPI/", "IMPS/", "NEFT/", "ATM WDL/", "POS/"}))
			b.V(p.nameFmt(c.r), pii.Name, "name")
			b.S("  ")
			b.V(rrn(c.r), "", "rrn")
			b.S("  ")
			b.V(FieldByKind["amount"].Gen(c.r, p), "", "amount")
			b.S("  ")
			b.V(FieldByKind["amount"].Gen(c.r, p), "", "amount")
			b.S("\n")
		}
	}
}

// ---- OCR noise -------------------------------------------------------------

var ocrSub = map[byte][]byte{
	'0': {'O', 'o'}, '1': {'l', 'I', '|'}, '5': {'S'}, '8': {'B'}, '2': {'Z'}, '6': {'b'}, '7': {'T'},
	'O': {'0'}, 'o': {'0'}, 'l': {'1'}, 'I': {'1', 'l'}, 'S': {'5'}, 's': {'5'}, 'B': {'8'}, 'Z': {'2'}, 'G': {'6'},
	'e': {'c'}, 'c': {'e'}, 'a': {'o'}, 'u': {'v'}, 'n': {'r'}, 'h': {'b'},
}

// ocrNoise corrupts text the way OCR does and remaps truth spans. Letters are
// corrupted more than digits (labels get mangled more often than values).
func ocrNoise(r *rand.Rand, text string, truth []pii.TruthSpan, letterRate, digitRate float64) (string, []pii.TruthSpan) {
	var out strings.Builder
	mapping := make([]int, len(text)+1)
	for i := 0; i < len(text); i++ {
		mapping[i] = out.Len()
		ch := text[i]
		isDigit := ch >= '0' && ch <= '9'
		rate := letterRate
		if isDigit {
			rate = digitRate
		}
		if ch >= 0x80 { // keep UTF-8 sequences intact
			out.WriteByte(ch)
			continue
		}
		switch {
		case (ch == ':' || ch == ',') && r.Float64() < 0.15:
			out.WriteByte(pick(r, []byte{' ', '.', ';', ch}))
		case ch == ' ' && r.Float64() < 0.03:
			// dropped or doubled space
			if r.Intn(2) == 0 {
				out.WriteString("  ")
			}
		case r.Float64() < rate:
			if subs, ok := ocrSub[ch]; ok {
				out.WriteByte(subs[r.Intn(len(subs))])
			} else {
				out.WriteByte(ch)
			}
		default:
			out.WriteByte(ch)
			if isDigit && r.Float64() < 0.004 {
				out.WriteByte(' ') // split a digit run
			}
		}
	}
	mapping[len(text)] = out.Len()
	nt := make([]pii.TruthSpan, len(truth))
	for i, t := range truth {
		nt[i] = pii.TruthSpan{Start: mapping[t.Start], End: mapping[t.End], Type: t.Type, Kind: t.Kind}
	}
	s := out.String()
	for i := range nt { // trim spans that gained a trailing space
		for nt[i].End > nt[i].Start && s[nt[i].End-1] == ' ' {
			nt[i].End--
		}
	}
	return s, nt
}

// ---- document assembly -------------------------------------------------------

// Options for generation.
type Options struct {
	N    int
	Seed int64
	// Shape weights (sum arbitrary)
	Weights map[string]int
}

var defaultWeights = map[string]int{"prose": 26, "form": 20, "csv": 15, "json": 10, "log": 12, "code": 5, "ocr": 8, "card": 4}

// Generate produces N documents. Splits are assigned by index: 70% train,
// 10% dev, 20% test. Label phrasings and sentence templates are pooled so
// that dev/test contain phrasings unseen in train.
func Generate(opts Options) []pii.Doc {
	w := opts.Weights
	if w == nil {
		w = defaultWeights
	}
	shapes := []string{"prose", "form", "csv", "json", "log", "code", "ocr", "card"}
	total := 0
	for _, s := range shapes {
		total += w[s]
	}
	docs := make([]pii.Doc, 0, opts.N)
	for i := 0; i < opts.N; i++ {
		r := rand.New(rand.NewSource(opts.Seed*1_000_003 + int64(i)))
		split := "train"
		switch i % 10 {
		case 7:
			split = "dev"
		case 8, 9:
			split = "test"
		}
		x := r.Intn(total)
		shape := shapes[0]
		for _, s := range shapes {
			x -= w[s]
			if x < 0 {
				shape = s
				break
			}
		}
		docs = append(docs, generateOne(r, i, split, shape))
	}
	return docs
}

func generateOne(r *rand.Rand, i int, split, shape string) pii.Doc {
	c := &ctx{r: r, split: split, p: NewPerson(r), b: &Builder{}, shape: shape}
	format := pii.FormatText
	switch shape {
	case "prose":
		c.prose()
	case "form":
		c.form()
	case "csv":
		c.csv()
		format = pii.FormatCSV
	case "json":
		c.jsonDoc()
		format = pii.FormatJSON
	case "log":
		c.logDoc()
		format = pii.FormatLog
	case "code":
		c.code()
	case "ocr":
		if r.Intn(2) == 0 {
			c.form()
		} else {
			c.prose()
		}
	case "card":
		c.card()
	}
	text := c.b.sb.String()
	truth := c.b.truth
	if shape == "ocr" || (shape == "card" && r.Intn(3) > 0) {
		text, truth = ocrNoise(r, text, truth, 0.03, 0.008)
		if shape == "card" {
			shape = "ocr"
		}
	}
	return pii.Doc{ID: fmt.Sprintf("d%06d", i), Split: split, Format: format, Shape: shape, Text: text, Truth: truth, Nested: true}
}

// MarshalJSONL encodes docs one per line.
func MarshalJSONL(docs []pii.Doc) ([]byte, error) {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	for i := range docs {
		if err := enc.Encode(&docs[i]); err != nil {
			return nil, err
		}
	}
	return []byte(sb.String()), nil
}
