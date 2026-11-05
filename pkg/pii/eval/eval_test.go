package eval

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
)

// Partially labelled documents must only yield the candidates an analyst
// judged, with the claim respected.
func TestCollectPartialDocuments(t *testing.T) {
	docs, err := corpus.LoadAny(filepath.Join("..", "..", "..", "testdata", "sample_decisions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	items, _ := Collect(docs, detect.New(detect.DefaultOptions()), 60)
	for _, it := range items {
		if it.Cand.Type != pii.Aadhaar {
			t.Fatalf("a candidate of another type leaked from a partial doc: %+v", it.Cand)
		}
		if it.TruthIdx < 0 {
			t.Fatalf("unjudged candidate kept: %+v", it.Cand)
		}
	}
	tp, fp := 0, 0
	for _, it := range items {
		if it.TP {
			tp++
		} else {
			fp++
			if it.Category != "analyst_fp" {
				t.Fatalf("fp category = %q", it.Category)
			}
		}
	}
	if tp != 2 || fp != 2 {
		t.Fatalf("expected 2 tp + 2 fp judged candidates, got %d + %d", tp, fp)
	}
}

func TestMatchSemantics(t *testing.T) {
	text := "Address: 12, MG Road, Indiranagar, Bengaluru 560038 end"
	addr := pii.TruthSpan{Start: 9, End: 51, Type: pii.Address, Kind: "address"}
	// a PIN detection inside an address: unknown unless nesting is labelled
	doc := &pii.Doc{ID: "d", Text: text, Truth: []pii.TruthSpan{addr}}
	c := &pii.Candidate{Start: 45, End: 51, Type: pii.Pincode}
	tp, _, cat, unknown := match(doc, c)
	if tp || !unknown || cat != "inside_address" {
		t.Fatalf("PIN inside unlabelled-nesting address: tp=%v cat=%s unknown=%v", tp, cat, unknown)
	}
	doc.Nested = true
	if _, _, cat, unknown = match(doc, c); unknown || cat != "address" {
		t.Fatalf("with nesting labelled, an unlabelled PIN is a real negative: cat=%s unknown=%v", cat, unknown)
	}
	doc.Truth = append(doc.Truth, pii.TruthSpan{Start: 45, End: 51, Type: pii.Pincode, Kind: "pincode"})
	if tp, _, _, _ = match(doc, c); !tp {
		t.Fatal("nested PIN span must make the PIN detection a true positive")
	}
	// an over-wide detection covering the whole address plus 100 bytes is not a TP
	wide := &pii.Candidate{Start: 0, End: 55, Type: pii.Address}
	if tp, _, _, _ = match(doc, wide); !tp {
		t.Fatal("55-byte detection of a 42-byte address covers >= half each way: TP")
	}
	doc2 := &pii.Doc{ID: "e", Text: strings.Repeat("x", 200) + text, Truth: []pii.TruthSpan{{Start: 209, End: 251, Type: pii.Address}}}
	if tp, _, _, _ = match(doc2, &pii.Candidate{Start: 0, End: 251, Type: pii.Address}); tp {
		t.Fatal("a 251-byte detection of a 42-byte address is a fragment-grab, not a TP")
	}
}
