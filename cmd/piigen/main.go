// piigen writes a synthetic labelled corpus as JSONL.
//
//	go run ./cmd/piigen -n 30000 -seed 7 -out data/corpus.jsonl
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/techiek77-ui/poc1/pkg/pii/synth"
)

func main() {
	n := flag.Int("n", 30000, "number of documents")
	seed := flag.Int64("seed", 7, "random seed")
	out := flag.String("out", "data/corpus.jsonl", "output JSONL path")
	hard := flag.Float64("hardneg", 0.35, "share of 12-digit non-Aadhaar values forced to be Verhoeff-valid")
	flag.Parse()

	synth.SetHardNegativeRate(*hard)
	docs := synth.Generate(synth.Options{N: *n, Seed: *seed})
	b, err := synth.MarshalJSONL(docs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	split := map[string]int{}
	shape := map[string]int{}
	spans := 0
	for _, d := range docs {
		split[d.Split]++
		shape[d.Shape]++
		spans += len(d.Truth)
	}
	fmt.Printf("wrote %d docs (%d truth spans) to %s\nsplits: %v\nshapes: %v\n", len(docs), spans, *out, split, shape)
}
