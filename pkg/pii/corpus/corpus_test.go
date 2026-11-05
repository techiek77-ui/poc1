package corpus

import (
	"path/filepath"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
)

func TestLoadDecisionsAndMarkup(t *testing.T) {
	docs, err := LoadAny(filepath.Join("..", "..", "..", "testdata", "sample_decisions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 4 {
		t.Fatalf("got %d docs", len(docs))
	}
	d := docs[1]
	if !d.Partial || d.Source != "real" || len(d.Truth) != 1 {
		t.Fatalf("decision doc not partial/real: %+v", d)
	}
	s := d.Truth[0]
	if s.Type != "" || s.Claim != pii.Aadhaar || s.Kind != "analyst_fp" || d.Text[s.Start:s.End] != "965147382010" {
		t.Fatalf("fp decision span = %+v (%q)", s, d.Text[s.Start:s.End])
	}
	if docs[0].Truth[0].Type != pii.Aadhaar || d.Text[docs[0].Truth[0].Start:docs[0].Truth[0].End] != "2847 1930 5622" {
		t.Fatalf("tp decision span = %+v", docs[0].Truth[0])
	}
	if docs[3].Split != "dev" || docs[3].Format != pii.FormatCSV {
		t.Fatalf("split/format not carried: %+v", docs[3])
	}
	if err := AssignSplits(docs, "train"); err != nil {
		t.Fatal(err)
	}
	if docs[0].Split != "train" || docs[3].Split != "dev" {
		t.Fatalf("AssignSplits must only fill empty splits")
	}
	if err := AssignSplits(docs, "trian"); err == nil {
		t.Fatal("misspelt split mode must be rejected")
	}
	// decisions about the same snippet share a split under hash mode
	for i := range docs {
		docs[i].Split = ""
	}
	if err := AssignSplits(docs, "hash"); err != nil {
		t.Fatal(err)
	}
	if docs[0].Split != docs[1].Split {
		t.Fatalf("same snippet in different splits: %s vs %s", docs[0].Split, docs[1].Split)
	}

	md, err := LoadAny(filepath.Join("..", "..", "..", "testdata", "sample_real.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(md) != 2 || md[0].Source != "real" || md[0].Partial {
		t.Fatalf("markup docs = %+v", md)
	}
	if len(md[1].Truth) != 2 || md[1].Truth[0].Kind != "txn_id" || md[1].Truth[1].Type != pii.Aadhaar {
		t.Fatalf("markup spans = %+v", md[1].Truth)
	}
}
