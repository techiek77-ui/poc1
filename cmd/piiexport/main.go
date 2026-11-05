// piiexport writes the candidate windows the Tier B (transformer) verifier is
// trained on: one JSON line per baseline candidate with its left context,
// value, right context, claimed type, structured key and label.
//
//	go run ./cmd/piiexport -corpus data/corpus.jsonl -out data/tierb
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/eval"
	"github.com/techiek77-ui/poc1/pkg/pii/onnxinput"
)

type row struct {
	Doc      string `json:"doc"`
	Shape    string `json:"shape"`
	Format   string `json:"format"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Left     string `json:"left"`
	Right    string `json:"right"`
	Key      string `json:"key"`
	Text     string `json:"text"` // the exact model input string (see onnxinput)
	Keyword  bool   `json:"keyword_hit"`
	Label    int    `json:"label"`
	Category string `json:"category"`
}

func main() {
	corpusPath := flag.String("corpus", "data/corpus.jsonl", "JSONL corpus")
	out := flag.String("out", "data/tierb", "output directory")
	window := flag.Int("window", 60, "context window")
	flag.Parse()

	docs, err := corpus.Load(*corpusPath)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	gated := detect.New(detect.Options{Window: *window, RequireKeyword: true})
	ungated := detect.New(detect.Options{Window: *window, RequireKeyword: false})
	for _, split := range []string{"train", "dev", "test"} {
		sd := corpus.Split(docs, split)
		for name, det := range map[string]*detect.Detector{"ungated": ungated, "gated": gated} {
			if split == "train" && name == "gated" {
				continue
			}
			items, _ := eval.Collect(sd, det, *window)
			path := filepath.Join(*out, fmt.Sprintf("%s_%s.jsonl", split, name))
			f, err := os.Create(path)
			if err != nil {
				fail(err)
			}
			enc := json.NewEncoder(f)
			enc.SetEscapeHTML(false)
			pos := 0
			for _, it := range items {
				r := row{Doc: it.Doc.ID, Shape: it.Doc.Shape, Format: string(it.Doc.Format), Type: string(it.Cand.Type), Value: it.Cand.Value,
					Left: it.A.Left, Right: it.A.Right, Key: it.A.Header, Text: onnxinput.Build(&it.Cand, it.A), Keyword: it.Cand.KeywordHit, Category: it.Category}
				if it.TP {
					r.Label = 1
					pos++
				}
				if err := enc.Encode(&r); err != nil {
					fail(err)
				}
			}
			f.Close()
			fmt.Printf("%s: %d rows (%d positive)\n", path, len(items), pos)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
