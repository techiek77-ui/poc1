package synth

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
)

func TestGenerateIsDeterministicAndAligned(t *testing.T) {
	a := Generate(Options{N: 800, Seed: 3})
	b := Generate(Options{N: 800, Seed: 3})
	if len(a) != len(b) {
		t.Fatal("length mismatch")
	}
	shapes := map[string]int{}
	for i := range a {
		if a[i].Text != b[i].Text || len(a[i].Truth) != len(b[i].Truth) {
			t.Fatalf("doc %d differs between runs", i)
		}
		for j := range a[i].Truth {
			if a[i].Truth[j] != b[i].Truth[j] {
				t.Fatalf("doc %d span %d differs between runs", i, j)
			}
		}
		if !a[i].Nested {
			t.Fatalf("doc %d: generated documents label nested PII", i)
		}
		shapes[a[i].Shape]++
		for _, s := range a[i].Truth {
			if s.Start < 0 || s.End > len(a[i].Text) || s.End <= s.Start {
				t.Fatalf("doc %s: bad span %+v", a[i].ID, s)
			}
			v := a[i].Text[s.Start:s.End]
			if strings.TrimSpace(v) != v {
				t.Fatalf("doc %s (%s): span %+v has surrounding whitespace: %q", a[i].ID, a[i].Shape, s, v)
			}
			if s.Kind == "aadhaar" && a[i].Shape != "ocr" { // OCR noise may corrupt digits: a realistic baseline miss
				d := checksum.Digits(v)
				if len(d) != 12 || !checksum.Verhoeff(d) || d[0] < '2' {
					t.Fatalf("aadhaar truth %q is not a valid Aadhaar", v)
				}
				if s.Type != pii.Aadhaar {
					t.Fatalf("aadhaar truth has type %q", s.Type)
				}
			}
		}
	}
	for _, s := range []string{"prose", "form", "csv", "json", "log", "code", "ocr", "card"} {
		if shapes[s] == 0 {
			t.Errorf("shape %s never generated", s)
		}
	}
}

// Every field must have a label allowed in every split, and every PII field a
// phrasing held out for dev and one held out for test (by normalised form, so
// "Aadhaar No." cannot stand in for "aadhaar_no").
func TestLabelPoolCoverage(t *testing.T) {
	for _, f := range Fields {
		allowed := map[string]int{}
		heldOut := map[string]int{}
		for _, l := range f.Labels {
			p := LabelPool(f, l)
			for _, sp := range []string{"train", "dev", "test"} {
				if p == "" || p == sp {
					allowed[sp]++
				}
			}
			if p == "dev" || p == "test" {
				heldOut[p]++
			}
		}
		for _, sp := range []string{"train", "dev", "test"} {
			if allowed[sp] == 0 {
				t.Errorf("field %s: no label allowed for split %s", f.Kind, sp)
			}
		}
		if f.Type != "" && (heldOut["dev"] == 0 || heldOut["test"] == 0) {
			t.Errorf("field %s: needs a held-out phrasing for dev and test (have dev=%d test=%d of %d labels)", f.Kind, heldOut["dev"], heldOut["test"], len(f.Labels))
		}
	}
	if Norm("Aadhaar No.") != Norm("aadhaar_no") || Norm("Aadhaar No.") != Norm("AADHAAR NO") {
		t.Fatal("normalisation must fold case and punctuation")
	}
	for _, list := range [][]string{prose1, prose2, prose3, hardLayouts2, hardLayouts3, keywordOnly, filler, mobileCtx, logMessages, subjects, emailOpeners, emailClosers} {
		for _, sp := range []string{"train", "dev", "test"} {
			ok := 0
			for _, s := range list {
				if allowed(s, sp) {
					ok++
				}
			}
			if ok == 0 {
				t.Errorf("a template list has nothing allowed for %s", sp)
			}
		}
	}
}

// Spans must be exactly the text written for them before OCR noise, and the
// OCR image of it afterwards (allowed substitutions only).
func TestSpansMatchValues(t *testing.T) {
	strip := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r == ' ' || r == ':' || r == ',' || r == '.' || r == ';' {
				return -1
			}
			return r
		}, s)
	}
	shapes := []string{"prose", "form", "card", "csv", "log", "json", "code"}
	checked := 0
	for seed := int64(0); seed < 1500; seed++ {
		r := rand.New(rand.NewSource(seed))
		shape := shapes[seed%int64(len(shapes))]
		c := &ctx{r: r, split: "train", p: NewPerson(r), b: &Builder{}, shape: shape}
		switch shape {
		case "prose":
			c.prose()
		case "form":
			c.form()
		case "card":
			c.card()
		case "csv":
			c.csv()
		case "log":
			c.logDoc()
		case "json":
			c.jsonDoc()
		default:
			c.code()
		}
		text, truth, vals := c.b.sb.String(), c.b.truth, c.b.vals
		if len(truth) != len(vals) {
			t.Fatalf("seed %d: %d spans but %d values", seed, len(truth), len(vals))
		}
		for i, ts := range truth {
			if got := text[ts.Start:ts.End]; got != vals[i] {
				t.Fatalf("seed %d %s: span %q != value %q", seed, shape, got, vals[i])
			}
		}
		noised, nt := ocrNoise(r, text, truth, 0.03, 0.008)
		for i := range truth {
			o, n := strip(vals[i]), strip(noised[nt[i].Start:nt[i].End])
			ok := len(o) == len(n)
			for k := 0; ok && k < len(o); k++ {
				if o[k] == n[k] {
					continue
				}
				found := false
				for _, b := range ocrSub[o[k]] {
					if b == n[k] {
						found = true
					}
				}
				ok = found
			}
			if v := noised[nt[i].Start:nt[i].End]; v == "" || strings.TrimSpace(v) != v {
				ok = false
			}
			if !ok {
				t.Fatalf("seed %d %s: OCR span %q is not the image of %q", seed, shape, noised[nt[i].Start:nt[i].End], vals[i])
			}
			checked++
		}
	}
	if checked < 10000 {
		t.Fatalf("only %d spans checked", checked)
	}
}

func TestHardNegativesExist(t *testing.T) {
	docs := Generate(Options{N: 2000, Seed: 11})
	collide := 0
	for _, d := range docs {
		for _, s := range d.Truth {
			if s.Type == pii.Aadhaar {
				continue
			}
			v := checksum.Digits(d.Text[s.Start:s.End])
			if len(v) == 12 && v[0] >= '2' && checksum.Verhoeff(v) && len(d.Text[s.Start:s.End]) == 12 {
				collide++
			}
		}
	}
	if collide < 100 {
		t.Fatalf("expected many Aadhaar-shaped non-Aadhaar values, got %d", collide)
	}
}
