// piieval compares the baseline, the rule verifier and the trained linear
// verifier on the held-out test split and writes a markdown report.
//
//	go run ./cmd/piieval -corpus data/corpus.jsonl -model models/tier_a_lr.json -out docs/RESULTS.md
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/eval"
	"github.com/techiek77-ui/poc1/pkg/pii/golden"
	"github.com/techiek77-ui/poc1/pkg/pii/verify"
)

func main() {
	var corpora corpus.Paths
	flag.Var(&corpora, "corpus", "corpus file (piigen JSONL, markup .txt, or decision JSONL); repeatable")
	realSplit := flag.String("real-split", "", "split for real documents that carry none: train|dev|test|hash (default: what the model was trained with, else test)")
	modelPath := flag.String("model", "models/tier_a_lr.json", "Tier A model artifact")
	window := flag.Int("window", 60, "context window")
	retain := flag.Float64("retain", 0.99, "TP retention target for threshold selection")
	out := flag.String("out", "docs/RESULTS.md", "markdown report path")
	nEx := flag.Int("examples", 15, "remaining-FP examples to print per configuration")
	split := flag.String("split", "test", "split to evaluate")
	goldenPath := flag.String("golden", "testdata/golden.txt", "hand-written golden set ('' to skip)")
	dump := flag.String("dump", "", "write every test candidate with scores to this JSONL file (error analysis)")
	flag.Parse()

	if len(corpora) == 0 {
		corpora = corpus.Paths{"data/corpus.jsonl"}
	}
	m, err := verify.Load(*modelPath)
	if err != nil {
		fail(err)
	}
	if err := m.CheckWindow(*window); err != nil {
		fail(err)
	}
	if *realSplit == "" {
		*realSplit = "test"
		if s, ok := m.Notes["real_split"].(string); ok && s != "" {
			*realSplit = s
		}
	}
	var trainedOn []string
	for _, p := range corpora {
		if h := fileHash(p); h != "" && strings.Contains(m.TrainingData, "sha256:"+h) {
			trainedOn = append(trainedOn, filepath.Base(p))
		}
	}
	docs, err := corpus.LoadAll(corpora)
	if err != nil {
		fail(err)
	}
	if err := corpus.AssignSplits(docs, *realSplit); err != nil {
		fail(err)
	}
	dev, test := corpus.Split(docs, "dev"), corpus.Split(docs, *split)
	corpusPath := new(string)
	*corpusPath = corpora.String()
	gated := detect.New(detect.Options{Window: *window, RequireKeyword: true})
	ungated := detect.New(detect.Options{Window: *window, RequireKeyword: false})

	devItems, _ := eval.Collect(dev, gated, *window)
	testItems, analyzeDur := eval.Collect(test, gated, *window)
	truthTest := eval.TruthCounts(test)

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Results on the held-out %s split\n\n", *split)
	fmt.Fprintf(&sb, "Corpus `%s`: %d documents (%d in %s). Window %d chars. Model `%s` (version %s, trained %s).\n\n",
		filepath.Base(*corpusPath), len(docs), len(test), *split, *window, filepath.Base(*modelPath), m.Version, m.TrainedAt)
	fmt.Fprintf(&sb, "Baseline = regex + checksum + same-type keyword within %d chars (or CSV header). "+
		"Every verifier only re-scores baseline detections. Thresholds for the ML configurations are chosen on the dev split to retain ≥ %.0f%% of the baseline's true positives per entity type; the rule verifier uses its natural 0.5 threshold.\n\n", *window, 100**retain)

	// --- configurations --------------------------------------------------------
	base := eval.Config{Name: "0 baseline", Scores: ones(len(testItems)), Default: 0.5}

	rules := verify.Rules{}
	rulesDev, _, _ := eval.Scores(devItems, rules)
	rulesTest, _, rulesDur := eval.Scores(testItems, rules)
	cfgRules := eval.Config{Name: "1 rules @0.5", Scores: rulesTest, Default: 0.5}
	thR, defR := eval.ChooseThresholds(devItems, rulesDev, *retain, 100)
	cfgRulesT := eval.Config{Name: fmt.Sprintf("1 rules @%.0f%%", 100**retain), Scores: rulesTest, Thresholds: thR, Default: defR}

	lin := &verify.Linear{M: m}
	linDev, _, _ := eval.Scores(devItems, lin)
	linTest, linReasons, linDur := eval.Scores(testItems, lin)
	thL, defL := eval.ChooseThresholds(devItems, linDev, *retain, 100)
	// The headline row uses the thresholds shipped in the artifact (the operating
	// point a deployment would run); the re-tuned row shows what this dev split
	// would choose, which differs when -retain/-window or the dev data differ.
	cfgLin := eval.Config{Name: "A linear", Scores: linTest, Thresholds: m.Thresholds, Default: m.DefaultThreshold}
	cfgLinT := eval.Config{Name: fmt.Sprintf("A linear (re-tuned on dev @%.0f%%)", 100**retain), Scores: linTest, Thresholds: thL, Default: defL}
	cfgLinR := eval.Config{Name: "A linear +resolve", Scores: linTest, Thresholds: m.Thresholds, Default: m.DefaultThreshold, Resolve: true}
	cfgRulesR := eval.Config{Name: "1 rules @0.5 +resolve", Scores: rulesTest, Default: 0.5, Resolve: true}

	cfgs := []eval.Config{base, cfgRules, cfgRulesT, cfgRulesR, cfgLin, cfgLinT, cfgLinR}
	// optional Tier B (ONNX transformer), only with -tags onnx and -onnx <dir>
	bVer, closeB := extraVerifier()
	defer closeB()
	var bTest []float64
	var bDur time.Duration
	if bVer != nil {
		bDev, bDevReasons, _ := eval.Scores(devItems, bVer)
		var bReasons []string
		bTest, bReasons, bDur = eval.Scores(testItems, bVer)
		if n := eval.CountErrors(bDevReasons) + eval.CountErrors(bReasons); n > 0 {
			fmt.Fprintf(&sb, "\n**Warning:** the ONNX verifier failed on %d of %d candidates (scored 0.5); its row below is not a measurement of the model.\n\n", n, len(devItems)+len(testItems))
			fmt.Fprintf(os.Stderr, "warning: ONNX verifier errors on %d candidates\n", n)
		}
		thB, defB := eval.ChooseThresholds(devItems, bDev, *retain, 100)
		cfgs = append(cfgs, eval.Config{Name: "B onnx", Scores: bTest, Thresholds: thB, Default: defB})
	}
	var results []*eval.Result
	for _, cfg := range cfgs {
		results = append(results, eval.Evaluate(testItems, cfg, truthTest, *nEx))
	}
	sb.WriteString(eval.Markdown(results, "Gated candidates (the product's detections)"))

	// --- retention sweep ---------------------------------------------------------
	sb.WriteString("\n### FP reduction vs. TP retention target (thresholds from dev, measured on test)\n\n")
	targets := []float64{0.999, 0.995, 0.99, 0.98, 0.95}
	sb.WriteString(eval.Sweep(devItems, testItems, rulesDev, rulesTest, targets, truthTest, "rules"))
	sb.WriteString("\n")
	sb.WriteString(eval.Sweep(devItems, testItems, linDev, linTest, targets, truthTest, "linear"))

	// --- replacing the keyword gate ----------------------------------------------
	devU, _ := eval.Collect(dev, ungated, *window)
	testU, _ := eval.Collect(test, ungated, *window)
	linDevU, _, _ := eval.Scores(devU, lin)
	linTestU, _, _ := eval.Scores(testU, lin)
	thU, defU := eval.ChooseThresholds(devU, linDevU, *retain, 100)
	if len(trainedOn) > 0 {
		fmt.Fprintf(&sb, "\n**Note:** %s also served as training data for this model (same file hash in the manifest); real-document numbers below are not out-of-sample.\n", strings.Join(trainedOn, ", "))
	}
	baseU := eval.Evaluate(testU, eval.Config{Name: "regex+checksum only (no gate)", Scores: ones(len(testU)), Default: 0.5}, truthTest, 0)
	linU := eval.Evaluate(testU, eval.Config{Name: "A linear as the gate", Scores: linTestU, Thresholds: thU, Default: defU}, truthTest, 0)
	baseG := results[0]
	linG := results[4] // "A linear" (shipped thresholds)
	sb.WriteString("\n### Keyword gate vs. model gate\n\n")
	sb.WriteString("The keyword gate also costs recall: values printed without a nearby label (identity-card OCR, bare table cells) are never reported. Scoring every regex+checksum match with the model instead of requiring a keyword recovers some of them.\n\n")
	sb.WriteString("| candidate set | TP | FP | precision | recall vs. all labelled spans |\n|---|---:|---:|---:|---:|\n")
	for _, r := range []*eval.Result{baseG, linG, baseU, linU} {
		fmt.Fprintf(&sb, "| %s | %d | %d | %.1f%% | %.1f%% |\n", r.Name, r.Total.TP, r.Total.FP, 100*r.Total.Precision(), 100*r.OverallRecall())
	}

	// --- real documents in the evaluated split ----------------------------------------
	if real := corpus.Real(test); len(real) > 0 {
		rItems, _ := eval.Collect(real, gated, *window)
		rTruth := eval.TruthCounts(real)
		rRules, _, _ := eval.Scores(rItems, rules)
		rLin, _, _ := eval.Scores(rItems, lin)
		var rr []*eval.Result
		for _, cfg := range []eval.Config{
			{Name: "0 baseline", Scores: ones(len(rItems)), Default: 0.5},
			{Name: "1 rules @0.5", Scores: rRules, Default: 0.5},
			{Name: "A linear", Scores: rLin, Thresholds: m.Thresholds, Default: m.DefaultThreshold},
		} {
			rr = append(rr, eval.Evaluate(rItems, cfg, rTruth, *nEx))
		}
		fmt.Fprintf(&sb, "\n### Real documents in the %s split (%d documents, %d judged candidates)\n\n", *split, len(real), len(rItems))
		sb.WriteString("Thresholds are the ones chosen on dev; partially labelled documents only count the candidates an analyst judged.\n\n")
		sb.WriteString(eval.Markdown(rr, "Real data"))
	}

	// --- golden set ------------------------------------------------------------------
	if *goldenPath != "" {
		sb.WriteString(goldenSection(*goldenPath, m, m.Thresholds, m.DefaultThreshold, *window))
	}
	if *dump != "" {
		if err := dumpItems(*dump, testItems, rulesTest, linTest, linReasons); err != nil {
			fail(err)
		}
	}

	// --- latency -------------------------------------------------------------------
	n := float64(len(testItems))
	sb.WriteString("\n### Cost per candidate (this container: 4 vCPU Xeon @ 2.1 GHz, single goroutine)\n\n")
	sb.WriteString("| stage | µs / candidate |\n|---|---:|\n")
	fmt.Fprintf(&sb, "| context analysis (attrib.Analyze, shared) | %.1f |\n", float64(analyzeDur.Microseconds())/n)
	fmt.Fprintf(&sb, "| rule verifier | %.2f |\n", float64(rulesDur.Microseconds())/n)
	fmt.Fprintf(&sb, "| linear verifier (features + dot product + explanation) | %.1f |\n", float64(linDur.Microseconds())/n)
	if bVer != nil {
		fmt.Fprintf(&sb, "| ONNX transformer verifier (tokenize + ONNX Runtime, %s) | %.1f |\n", bVer.Name(), float64(bDur.Microseconds())/n)
	}

	// --- examples --------------------------------------------------------------------
	sb.WriteString("\n### Remaining false positives after `A linear` (sample)\n\n")
	for _, e := range results[4].Examples {
		fmt.Fprintf(&sb, "- `%s`\n", e)
	}
	sb.WriteString("\n### Example explanations written to the log by `A linear`\n\n")
	shown := 0
	for i, it := range testItems {
		if shown >= 6 {
			break
		}
		if it.Cand.Type != pii.Aadhaar {
			continue
		}
		fmt.Fprintf(&sb, "- %s `%s` truth=%s score=%.3f keep=%v — %s\n", it.Cand.Type, it.Cand.Value, it.Category, linTest[i], linTest[i] >= m.Threshold(it.Cand.Type), linReasons[i])
		shown++
	}
	fmt.Fprintf(&sb, "\n_Generated %s._\n", time.Now().UTC().Format(time.RFC3339))

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(*out, []byte(sb.String()), 0o644); err != nil {
		fail(err)
	}
	fmt.Println(sb.String())
}

// goldenSection evaluates the hand-written documents and lists every wrong
// decision, because with ~50 documents the individual cases matter more than
// the aggregate.
func goldenSection(path string, m *verify.Model, th map[pii.EntityType]float64, def float64, window int) string {
	docs, err := golden.LoadFile(path)
	if err != nil {
		return fmt.Sprintf("\n### Golden set\n\ncould not load %s: %v\n", path, err)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n### Golden set: %d hand-written documents in styles the generator does not produce\n\n", len(docs))
	gated := detect.New(detect.Options{Window: window, RequireKeyword: true})
	ungated := detect.New(detect.Options{Window: window, RequireKeyword: false})
	items, _ := eval.Collect(docs, gated, window)
	itemsU, _ := eval.Collect(docs, ungated, window)
	truth := eval.TruthCounts(docs)
	rules := verify.Rules{}
	lin := &verify.Linear{M: m}
	rs, rr, _ := eval.Scores(items, rules)
	ls, lr, _ := eval.Scores(items, lin)
	lsU, _, _ := eval.Scores(itemsU, lin)
	cfgs := []eval.Config{
		{Name: "0 baseline", Scores: ones(len(items)), Default: 0.5},
		{Name: "1 rules @0.5", Scores: rs, Default: 0.5},
		{Name: "A linear", Scores: ls, Thresholds: th, Default: def},
	}
	var results []*eval.Result
	for _, c := range cfgs {
		results = append(results, eval.Evaluate(items, c, truth, 0))
	}
	sb.WriteString("| configuration | TP | FP | precision | TP retention | FP reduction | recall vs. all spans |\n|---|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range results {
		fmt.Fprintf(&sb, "| %s | %d | %d | %.1f%% | %.1f%% | %.1f%% | %.1f%% |\n", r.Name, r.Total.TP, r.Total.FP, 100*r.Total.Precision(), 100*r.Total.Retention(), 100*r.Total.FPReduction(), 100*r.OverallRecall())
	}
	rU := eval.Evaluate(itemsU, eval.Config{Name: "A linear as the gate (ungated)", Scores: lsU, Thresholds: th, Default: def}, truth, 0)
	fmt.Fprintf(&sb, "| %s | %d | %d | %.1f%% | – | – | %.1f%% |\n", rU.Name, rU.Total.TP, rU.Total.FP, 100*rU.Total.Precision(), 100*rU.OverallRecall())

	sb.WriteString("\nEvery wrong decision (gated candidates):\n\n")
	sb.WriteString("| doc | claim | value | really | rules | linear | outcome | model explanation |\n|---|---|---|---|---:|---:|---|---|\n")
	for i, it := range items {
		keepR := rs[i] >= 0.5
		t := th[it.Cand.Type]
		if _, ok := th[it.Cand.Type]; !ok {
			t = def
		}
		keepL := ls[i] >= t
		var outcome string
		switch {
		case it.TP && !keepL && !keepR:
			outcome = "MISSED by both"
		case it.TP && !keepL:
			outcome = "MISSED by linear"
		case it.TP && !keepR:
			outcome = "missed by rules"
		case !it.TP && keepL && keepR:
			outcome = "FP kept by both"
		case !it.TP && keepL:
			outcome = "FP kept by linear"
		case !it.TP && keepR:
			outcome = "fp kept by rules"
		default:
			continue
		}
		val := strings.ReplaceAll(it.Cand.Value, "|", "\\|")
		if len(val) > 30 {
			val = val[:27] + "..."
		}
		why := lr[i]
		if !keepR && it.TP || keepR && !it.TP {
			why = "rules: " + rr[i] + " · model: " + lr[i]
		}
		fmt.Fprintf(&sb, "| %s %s | %s | `%s` | %s | %.2f | %.3f | %s | %s |\n", it.Doc.ID, it.Doc.Shape, it.Cand.Type, val, it.Category, rs[i], ls[i], outcome, strings.ReplaceAll(why, "|", "\\|"))
	}
	// what the keyword gate misses entirely
	sb.WriteString("\nLabelled spans the baseline never proposes (no keyword in the window), and whether the model-as-gate recovers them:\n\n")
	type key struct {
		id  string
		idx int
	}
	proposed := map[key]bool{}
	for _, it := range items {
		if it.TP {
			proposed[key{it.Doc.ID, it.TruthIdx}] = true
		}
	}
	recovered := map[key]bool{}
	for i, it := range itemsU {
		t := th[it.Cand.Type]
		if _, ok := th[it.Cand.Type]; !ok {
			t = def
		}
		if it.TP && lsU[i] >= t {
			recovered[key{it.Doc.ID, it.TruthIdx}] = true
		}
	}
	for _, d := range docs {
		for i, ts := range d.Truth {
			if ts.Type == "" || proposed[key{d.ID, i}] {
				continue
			}
			rec := "not recovered"
			if recovered[key{d.ID, i}] {
				rec = "recovered by model gate"
			}
			fmt.Fprintf(&sb, "- %s %s `%s` — %s\n", d.ID, ts.Type, d.Text[ts.Start:ts.End], rec)
		}
	}
	return sb.String()
}

type dumpRow struct {
	Doc, Shape, Format, Type, Value, Category, Left, Right, Reason string
	TP                                                             bool
	Rules, Linear                                                  float64
}

func dumpItems(path string, items []eval.Item, rules, lin []float64, reasons []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i, it := range items {
		if err := enc.Encode(dumpRow{Doc: it.Doc.ID, Shape: it.Doc.Shape, Format: string(it.Doc.Format), Type: string(it.Cand.Type), Value: it.Cand.Value,
			Category: it.Category, Left: it.A.Left, Right: it.A.Right, Reason: reasons[i], TP: it.TP, Rules: rules[i], Linear: lin[i]}); err != nil {
			return err
		}
	}
	return nil
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
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
