//go:build onnx

package onnx

import (
	"fmt"
	"os"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/eval"
	"github.com/techiek77-ui/poc1/pkg/pii/onnxinput"
)

// TestTokenizerOnCorpus encodes every dev-split candidate window and reports
// inputs the Go tokenizer cannot handle (PII_CORPUS and PII_ORT_LIB must be set).
func TestTokenizerOnCorpus(t *testing.T) {
	path := os.Getenv("PII_CORPUS")
	if path == "" || os.Getenv("PII_ORT_LIB") == "" {
		t.Skip("PII_CORPUS / PII_ORT_LIB not set")
	}
	if err := Init(os.Getenv("PII_ORT_LIB")); err != nil {
		t.Fatal(err)
	}
	v, err := New("../../../models/tier_b")
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	docs, err := corpus.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	items, _ := eval.Collect(corpus.Split(docs, "dev"), detect.New(detect.DefaultOptions()), 60)
	bad := 0
	for _, it := range items {
		text := onnxinput.Build(&it.Cand, it.A)
		if _, _, _, err := v.Encode(text); err != nil {
			bad++
			if bad <= 5 {
				fmt.Printf("FAIL %q\n   -> %v\n", text, err)
			}
		}
	}
	fmt.Printf("%d of %d windows failed to tokenize\n", bad, len(items))
}
