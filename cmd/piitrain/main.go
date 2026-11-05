// piitrain trains the Tier A linear verifier and writes a versioned model
// artifact (weights + per-entity thresholds + manifest).
//
//	go run ./cmd/piitrain -corpus data/corpus.jsonl -out models/tier_a_lr.json
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/eval"
	"github.com/techiek77-ui/poc1/pkg/pii/verify"
)

func main() {
	corpusPath := flag.String("corpus", "data/corpus.jsonl", "JSONL corpus from piigen")
	out := flag.String("out", "models/tier_a_lr.json", "model artifact path")
	window := flag.Int("window", 60, "context window (chars each side)")
	retain := flag.Float64("retain", 0.99, "share of baseline true positives to keep when choosing thresholds")
	epochs := flag.Int("epochs", 8, "training epochs")
	lr := flag.Float64("lr", 0.1, "learning rate")
	l2 := flag.Float64("l2", 1e-6, "L2 regularisation")
	minCount := flag.Int("mincount", 3, "minimum feature count")
	posw := flag.Float64("posw", 1.0, "weight of positive examples")
	prune := flag.Float64("prune", 0.0, "drop weights with |w| below this after training (smaller artifact)")
	flag.Parse()

	docs, err := corpus.Load(*corpusPath)
	if err != nil {
		fail(err)
	}
	train, dev := corpus.Split(docs, "train"), corpus.Split(docs, "dev")
	fmt.Printf("corpus: %d docs (train %d, dev %d)\n", len(docs), len(train), len(dev))

	// Train on ungated candidates so the model also learns what a missing
	// keyword means; thresholds are chosen on the product's gated candidates.
	ungated := detect.New(detect.Options{Window: *window, RequireKeyword: false})
	gated := detect.New(detect.Options{Window: *window, RequireKeyword: true})

	t0 := time.Now()
	trainItems, _ := eval.Collect(train, ungated, *window)
	fmt.Printf("train candidates: %d (%.1fs)\n", len(trainItems), time.Since(t0).Seconds())
	examples := make([]verify.Example, len(trainItems))
	pos := 0
	for i, it := range trainItems {
		examples[i] = verify.Example{Feats: verify.Features(&it.Cand, it.A), Label: it.TP, Type: it.Cand.Type}
		if it.TP {
			pos++
		}
	}
	fmt.Printf("positives: %d, negatives: %d\n", pos, len(examples)-pos)

	o := verify.DefaultTrainOptions()
	o.Epochs, o.LR, o.L2, o.MinCount, o.PosW = *epochs, *lr, *l2, *minCount, float32(*posw)
	t0 = time.Now()
	m := verify.Train(examples, o)
	fmt.Printf("trained %d weights in %.1fs\n", len(m.Weights), time.Since(t0).Seconds())
	if *prune > 0 {
		before := len(m.Weights)
		for f, w := range m.Weights {
			if w < float32(*prune) && w > -float32(*prune) {
				delete(m.Weights, f)
			}
		}
		fmt.Printf("pruned |w|<%g: %d -> %d weights\n", *prune, before, len(m.Weights))
	}

	devItems, _ := eval.Collect(dev, gated, *window)
	lin := &verify.Linear{M: m}
	devScores, _, _ := eval.Scores(devItems, lin)
	th, def := eval.ChooseThresholds(devItems, devScores, *retain, 30)
	m.Thresholds, m.DefaultThreshold = th, def
	truth := eval.TruthCounts(dev)
	base := eval.Evaluate(devItems, eval.Config{Name: "baseline", Scores: ones(len(devItems))}, truth, 0)
	res := eval.Evaluate(devItems, eval.Config{Name: "linear", Scores: devScores, Thresholds: th, Default: def}, truth, 0)
	fmt.Printf("dev: baseline TP %d FP %d | linear TP %d FP %d | retention %.2f%% | FP reduction %.1f%%\n",
		base.Total.TPBase, base.Total.FPBase, res.Total.TP, res.Total.FP, 100*res.Total.Retention(), 100*res.Total.FPReduction())

	m.TrainingData = fmt.Sprintf("%s sha256:%s", filepath.Base(*corpusPath), fileHash(*corpusPath))
	m.Notes = map[string]any{
		"window":              *window,
		"retention_target":    *retain,
		"dev_tp_retention":    res.Total.Retention(),
		"dev_fp_reduction":    res.Total.FPReduction(),
		"dev_precision":       res.Total.Precision(),
		"train_examples_pos":  pos,
		"train_examples_neg":  len(examples) - pos,
		"optimizer":           fmt.Sprintf("adagrad lr=%g l2=%g epochs=%d mincount=%d posw=%g prune=%g", *lr, *l2, *epochs, *minCount, *posw, *prune),
		"candidate_generator": "regex+checksum, ungated for training, gated for thresholds",
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail(err)
	}
	if err := m.Save(*out); err != nil {
		fail(err)
	}
	fmt.Printf("saved %s (version %s, schema %d)\n", *out, m.Version, m.SchemaVersion)
}

func ones(n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = 1
	}
	return s
}

func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "?"
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
