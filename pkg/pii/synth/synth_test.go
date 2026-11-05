package synth

import (
	"strings"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
)

func TestGenerateIsDeterministicAndAligned(t *testing.T) {
	a := Generate(Options{N: 300, Seed: 3})
	b := Generate(Options{N: 300, Seed: 3})
	if len(a) != len(b) {
		t.Fatal("length mismatch")
	}
	shapes := map[string]int{}
	for i := range a {
		if a[i].Text != b[i].Text {
			t.Fatalf("doc %d differs between runs", i)
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
	for _, s := range []string{"prose", "form", "csv", "json", "log", "code", "ocr"} {
		if shapes[s] == 0 {
			t.Errorf("shape %s never generated", s)
		}
	}
}

func TestPoolsAreDisjoint(t *testing.T) {
	seenTrain, seenTest := map[string]bool{}, map[string]bool{}
	for _, l := range Fields[0].Labels {
		switch Pool(l) {
		case "train":
			seenTrain[l] = true
		case "test":
			seenTest[l] = true
		}
	}
	for l := range seenTrain {
		if seenTest[l] {
			t.Fatalf("%q in both pools", l)
		}
	}
	if len(seenTest) == 0 {
		t.Log("warning: no test-only Aadhaar labels; held-out phrasing coverage comes from templates")
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
