package golden_test

import (
	"path/filepath"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/eval"
	"github.com/techiek77-ui/poc1/pkg/pii/golden"
	"github.com/techiek77-ui/poc1/pkg/pii/verify"
)

// TestRulesOnGoldenSet is the regression guard for the rule verifier: on the
// hand-written documents it must keep every true positive and remove most of
// the baseline's false positives. It needs no trained model.
func TestRulesOnGoldenSet(t *testing.T) {
	docs, err := golden.LoadFile(filepath.Join("..", "..", "..", "testdata", "golden.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) < 40 {
		t.Fatalf("only %d golden docs", len(docs))
	}
	det := detect.New(detect.DefaultOptions())
	items, _ := eval.Collect(docs, det, 60)
	scores, reasons, _ := eval.Scores(items, verify.Rules{})
	truth := eval.TruthCounts(docs)
	base := eval.Evaluate(items, eval.Config{Name: "baseline", Scores: ones(len(items)), Default: 0.5}, truth, 0)
	res := eval.Evaluate(items, eval.Config{Name: "rules", Scores: scores, Default: 0.5}, truth, 0)
	t.Logf("baseline TP %d FP %d; rules TP %d FP %d (retention %.1f%%, FP reduction %.1f%%)",
		base.Total.TPBase, base.Total.FPBase, res.Total.TP, res.Total.FP, 100*res.Total.Retention(), 100*res.Total.FPReduction())
	for i, it := range items {
		if it.TP && scores[i] < 0.5 {
			t.Errorf("rules dropped a true positive: %s %s %q (%s)", it.Doc.ID, it.Cand.Type, it.Cand.Value, reasons[i])
		}
	}
	if res.Total.FPReduction() < 0.8 {
		t.Errorf("rules FP reduction %.1f%% < 80%%", 100*res.Total.FPReduction())
	}
	// the brief's example, both orders
	for _, text := range []string{"aadhar 738192045614, bank acc 509347182063", "bank acc 509347182063, aadhar 738192045614"} {
		doc := &pii.Doc{ID: "c", Format: pii.FormatText, Text: text}
		v := detect.NewDocView(doc)
		kept := map[string]bool{}
		for _, c := range det.Detect(v) {
			s, _ := verify.Rules{}.Score(v, &c, analyzeFor(v, &c))
			if s >= 0.5 {
				kept[string(c.Type)+":"+c.Value] = true
			}
		}
		if !kept["AADHAAR:738192045614"] || !kept["BANK_ACCOUNT:509347182063"] || kept["AADHAAR:509347182063"] || kept["BANK_ACCOUNT:738192045614"] {
			t.Errorf("%q: kept %v", text, kept)
		}
	}
}

func ones(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = 1
	}
	return s
}
