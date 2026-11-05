package eval

import (
	"path/filepath"
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
