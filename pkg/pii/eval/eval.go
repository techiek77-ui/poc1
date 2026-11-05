// Package eval scores detector configurations against the synthetic ground
// truth: it matches candidates to truth spans, selects operating points that
// retain a target share of the baseline's true positives, and reports the
// false-positive reduction per entity, per data shape and per false-positive
// category.
package eval

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/verify"
)

// Item is one baseline candidate with its ground-truth outcome.
type Item struct {
	Doc      *pii.Doc
	View     *detect.DocView
	Cand     pii.Candidate
	A        *attrib.Analysis
	TP       bool
	TruthIdx int    // index into Doc.Truth of the matched span, -1 if none
	Category string // truth kind the span overlaps, or "no_truth"
}

// Collect runs the detector over docs and labels every candidate. It returns
// the items and the time spent in context analysis (attrib.Analyze), which
// is the per-candidate cost every verifier pays.
func Collect(docs []pii.Doc, det *detect.Detector, window int) ([]Item, time.Duration) {
	var items []Item
	var analyze time.Duration
	for i := range docs {
		doc := &docs[i]
		v := detect.NewDocView(doc)
		for _, c := range det.Detect(v) {
			it := Item{Doc: doc, View: v, Cand: c}
			var unknown bool
			it.TP, it.TruthIdx, it.Category, unknown = match(doc, &c)
			if unknown {
				continue // PII inside PII of another type, in a corpus that does not label nesting
			}
			if doc.Partial {
				// only what an analyst actually judged is usable
				if it.TruthIdx < 0 {
					continue
				}
				if cl := doc.Truth[it.TruthIdx].Claim; cl != "" && cl != c.Type {
					continue
				}
			}
			t0 := time.Now()
			it.A = attrib.Analyze(v, &it.Cand, window)
			analyze += time.Since(t0)
			items = append(items, it)
		}
	}
	return items, analyze
}

// match decides what a detection is. It is a true positive when a labelled
// span of the claimed type covers at least half of the detection and the
// detection covers at least half of that span (so neither a fragment nor a
// wildly over-wide span counts). Otherwise the largest-overlap span gives the
// false-positive category; a detection lying entirely inside PII of another
// type, in a corpus that does not label nested PII, is "unknown".
func match(doc *pii.Doc, c *pii.Candidate) (tp bool, idx int, category string, unknown bool) {
	clen := c.End - c.Start
	best, bestOv := -1, 0
	for i, t := range doc.Truth {
		ov := min(c.End, t.End) - max(c.Start, t.Start)
		if ov <= 0 {
			continue
		}
		if t.Type == c.Type && ov*2 >= clen && ov*2 >= t.End-t.Start && (best < 0 || doc.Truth[best].Type != c.Type || ov > bestOv) {
			if best < 0 || doc.Truth[best].Type != c.Type {
				best, bestOv = i, ov
			} else if ov > bestOv {
				best, bestOv = i, ov
			}
		}
	}
	if best >= 0 {
		return true, best, doc.Truth[best].Kind, false
	}
	for i, t := range doc.Truth {
		ov := min(c.End, t.End) - max(c.Start, t.Start)
		if ov > bestOv {
			best, bestOv = i, ov
		}
	}
	if best < 0 {
		return false, -1, "no_truth", false
	}
	t := doc.Truth[best]
	if bestOv*2 < min(clen, t.End-t.Start) {
		return false, -1, "no_truth", false
	}
	if t.Type != "" && t.Type != c.Type && !doc.Nested && t.Start <= c.Start && t.End >= c.End {
		return false, best, "inside_" + strings.ToLower(string(t.Type)), true
	}
	return false, best, t.Kind, false
}

// TruthCounts counts labelled PII spans per type.
func TruthCounts(docs []pii.Doc) map[pii.EntityType]int {
	m := map[pii.EntityType]int{}
	for _, d := range docs {
		for _, t := range d.Truth {
			if t.Type != "" {
				m[t.Type]++
			}
		}
	}
	return m
}

// Scores runs a verifier over items. Baseline (nil verifier) scores 1.
func Scores(items []Item, ver verify.Verifier) ([]float64, []string, time.Duration) {
	scores := make([]float64, len(items))
	reasons := make([]string, len(items))
	start := time.Now()
	for i := range items {
		if ver == nil {
			scores[i] = 1
			continue
		}
		scores[i], reasons[i] = ver.Score(items[i].View, &items[i].Cand, items[i].A)
	}
	return scores, reasons, time.Since(start)
}

// MaxThreshold caps every chosen threshold (see verify.MaxThreshold).
var MaxThreshold = verify.MaxThreshold

// CountErrors returns how many reasons report a verifier failure.
func CountErrors(reasons []string) int {
	n := 0
	for _, r := range reasons {
		if strings.HasPrefix(r, verify.ErrorReasonPrefix) {
			n++
		}
	}
	return n
}

// ChooseThresholds picks, per entity type, the highest threshold (capped at
// MaxThreshold) that keeps at least `retain` of that type's true-positive
// candidates on dev, with a one-standard-deviation margin so the target also
// holds on unseen data (the plain quantile is a dev-set property: with the
// 1st percentile of n dev TPs the expected hold-out retention is below the
// target about half the time). Types with fewer than minN true positives fall
// back to the pooled threshold.
func ChooseThresholds(items []Item, scores []float64, retain float64, minN int) (map[pii.EntityType]float64, float64) {
	byType := map[pii.EntityType][]float64{}
	var all []float64
	for i, it := range items {
		if it.TP {
			byType[it.Cand.Type] = append(byType[it.Cand.Type], scores[i])
			all = append(all, scores[i])
		}
	}
	pick := func(s []float64) float64 {
		if len(s) == 0 {
			return 0.5
		}
		sort.Float64s(s)
		n := float64(len(s))
		margin := math.Ceil(math.Sqrt(n * retain * (1 - retain)))
		k := int(math.Floor((1-retain)*n) - margin)
		if k < 0 {
			k = 0
		}
		if k >= len(s) {
			k = len(s) - 1
		}
		// threshold just below the k-th smallest TP score so that it is kept
		return min(math.Nextafter(s[k], 0), MaxThreshold)
	}
	def := pick(all)
	out := map[pii.EntityType]float64{}
	for t, s := range byType {
		if len(s) >= minN {
			out[t] = pick(s)
		} else {
			out[t] = def
		}
	}
	return out, def
}

// Config names a scoring setup.
type Config struct {
	Name       string
	Scores     []float64
	Thresholds map[pii.EntityType]float64
	Default    float64
	Resolve    bool // drop a lower-scoring claim when the same span is claimed as two types
}

func (c *Config) keep(it *Item, i int) bool {
	th, ok := c.Thresholds[it.Cand.Type]
	if !ok {
		th = c.Default
	}
	return c.Scores[i] >= th
}

// Cell is a confusion-count bucket.
type Cell struct {
	TP, FP, TPBase, FPBase int
}

func (c Cell) Precision() float64 {
	if c.TP+c.FP == 0 {
		return 0
	}
	return float64(c.TP) / float64(c.TP+c.FP)
}
func (c Cell) BasePrecision() float64 {
	if c.TPBase+c.FPBase == 0 {
		return 0
	}
	return float64(c.TPBase) / float64(c.TPBase+c.FPBase)
}
func (c Cell) Retention() float64 {
	if c.TPBase == 0 {
		return 1
	}
	return float64(c.TP) / float64(c.TPBase)
}
func (c Cell) FPReduction() float64 {
	if c.FPBase == 0 {
		return 0
	}
	return 1 - float64(c.FP)/float64(c.FPBase)
}

// Result aggregates a config over items.
type Result struct {
	Name     string
	Total    Cell
	ByType   map[pii.EntityType]*Cell
	ByShape  map[string]*Cell
	ByCat    map[string]*Cell // FP categories only (truth kind of suppressed/kept FPs)
	Truth    map[pii.EntityType]int
	Covered  map[pii.EntityType]int     // truth spans with at least one kept true-positive detection
	Recall   map[pii.EntityType]float64 // covered / truth spans
	Kept     int
	Examples []string
}

// OverallRecall is covered truth spans over all truth spans.
func (r *Result) OverallRecall() float64 {
	c, n := 0, 0
	for t, k := range r.Truth {
		n += k
		c += r.Covered[t]
	}
	if n == 0 {
		return 0
	}
	return float64(c) / float64(n)
}

// Evaluate applies a config to items.
func Evaluate(items []Item, cfg Config, truth map[pii.EntityType]int, maxExamples int) *Result {
	res := &Result{Name: cfg.Name, ByType: map[pii.EntityType]*Cell{}, ByShape: map[string]*Cell{}, ByCat: map[string]*Cell{}, Truth: truth, Covered: map[pii.EntityType]int{}, Recall: map[pii.EntityType]float64{}}
	keep := make([]bool, len(items))
	type tkey struct {
		id  string
		idx int
	}
	covered := map[tkey]bool{}
	for i := range items {
		keep[i] = cfg.keep(&items[i], i)
	}
	if cfg.Resolve {
		// same span claimed as several types: keep the best if it wins clearly
		type key struct {
			id   string
			s, e int
		}
		groups := map[key][]int{}
		for i, it := range items {
			if keep[i] {
				k := key{it.Doc.ID, it.Cand.Start, it.Cand.End}
				groups[k] = append(groups[k], i)
			}
		}
		for _, g := range groups {
			if len(g) < 2 {
				continue
			}
			sort.Slice(g, func(a, b int) bool { return cfg.Scores[g[a]] > cfg.Scores[g[b]] })
			if cfg.Scores[g[0]]-cfg.Scores[g[1]] >= 0.2 {
				for _, i := range g[1:] {
					keep[i] = false
				}
			}
		}
	}
	cell := func(m map[string]*Cell, k string) *Cell {
		if m[k] == nil {
			m[k] = &Cell{}
		}
		return m[k]
	}
	for i, it := range items {
		tc := res.ByType[it.Cand.Type]
		if tc == nil {
			tc = &Cell{}
			res.ByType[it.Cand.Type] = tc
		}
		sc := cell(res.ByShape, it.Doc.Shape)
		var cc *Cell
		if !it.TP {
			cc = cell(res.ByCat, it.Category)
		}
		if it.TP {
			res.Total.TPBase++
			tc.TPBase++
			sc.TPBase++
			if keep[i] {
				res.Total.TP++
				tc.TP++
				sc.TP++
				k := tkey{it.Doc.ID, it.TruthIdx}
				if !covered[k] {
					covered[k] = true
					res.Covered[it.Cand.Type]++
				}
			}
		} else {
			res.Total.FPBase++
			tc.FPBase++
			sc.FPBase++
			cc.FPBase++
			if keep[i] {
				res.Total.FP++
				tc.FP++
				sc.FP++
				cc.FP++
				if len(res.Examples) < maxExamples {
					res.Examples = append(res.Examples, exampleLine(&items[i], cfg.Scores[i]))
				}
			}
		}
		if keep[i] {
			res.Kept++
		}
	}
	for t, n := range truth {
		if n > 0 {
			res.Recall[t] = float64(res.Covered[t]) / float64(n)
		}
	}
	return res
}

func exampleLine(it *Item, score float64) string {
	ctx := strings.ReplaceAll(it.A.Left, "\n", "⏎")
	rctx := strings.ReplaceAll(it.A.Right, "\n", "⏎")
	if len(ctx) > 40 {
		cut := len(ctx) - 40
		for cut < len(ctx) && !utf8.RuneStart(ctx[cut]) {
			cut++
		}
		ctx = ctx[cut:]
	}
	if len(rctx) > 25 {
		cut := 25
		for cut > 0 && !utf8.RuneStart(rctx[cut]) {
			cut--
		}
		rctx = rctx[:cut]
	}
	return fmt.Sprintf("%s [%s truth=%s %.2f] …%s«%s»%s…", it.Doc.Shape, it.Cand.Type, it.Category, score, ctx, it.Cand.Value, rctx)
}

// Markdown renders a comparison table of results.
func Markdown(results []*Result, title string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %s\n\n", title)
	fmt.Fprintf(&sb, "| configuration | detections kept | TP | FP | precision | TP retention (detections) | span retention | FP reduction |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	baseCovered := 0
	for _, n := range results[0].Covered {
		baseCovered += n
	}
	for _, r := range results {
		covered := 0
		for _, n := range r.Covered {
			covered += n
		}
		span := 1.0
		if baseCovered > 0 {
			span = float64(covered) / float64(baseCovered)
		}
		fmt.Fprintf(&sb, "| %s | %d | %d | %d | %.1f%% | %.2f%% | %.2f%% | %.1f%% |\n", r.Name, r.Kept, r.Total.TP, r.Total.FP, 100*r.Total.Precision(), 100*r.Total.Retention(), 100*span, 100*r.Total.FPReduction())
	}
	sb.WriteString("\n_TP retention counts detections (several may cover one labelled span); span retention counts labelled spans that keep at least one detection._\n")
	// per type
	types := []pii.EntityType{}
	for t := range results[0].ByType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	fmt.Fprintf(&sb, "\n#### Per entity type (baseline FP → FP after each configuration; TP retention in brackets)\n\n| type | truth spans | baseline TP | baseline recall | baseline FP | baseline precision |")
	for _, r := range results[1:] {
		fmt.Fprintf(&sb, " %s |", r.Name)
	}
	sb.WriteString("\n|---|---:|---:|---:|---:|---:|")
	for range results[1:] {
		sb.WriteString("---:|")
	}
	sb.WriteString("\n")
	for _, t := range types {
		b := results[0].ByType[t]
		fmt.Fprintf(&sb, "| %s | %d | %d | %.1f%% | %d | %.1f%% |", t, results[0].Truth[t], b.TPBase, 100*results[0].Recall[t], b.FPBase, 100*b.BasePrecision())
		for _, r := range results[1:] {
			c := r.ByType[t]
			if c == nil {
				sb.WriteString(" – |")
				continue
			}
			fmt.Fprintf(&sb, " %d (%.1f%%) |", c.FP, 100*c.Retention())
		}
		sb.WriteString("\n")
	}
	// per shape
	shapes := []string{}
	for s := range results[0].ByShape {
		shapes = append(shapes, s)
	}
	sort.Strings(shapes)
	fmt.Fprintf(&sb, "\n#### Per data shape (FP after each configuration; TP retention in brackets)\n\n| shape | baseline TP | baseline FP |")
	for _, r := range results[1:] {
		fmt.Fprintf(&sb, " %s |", r.Name)
	}
	sb.WriteString("\n|---|---:|---:|")
	for range results[1:] {
		sb.WriteString("---:|")
	}
	sb.WriteString("\n")
	for _, s := range shapes {
		b := results[0].ByShape[s]
		fmt.Fprintf(&sb, "| %s | %d | %d |", s, b.TPBase, b.FPBase)
		for _, r := range results[1:] {
			c := r.ByShape[s]
			fmt.Fprintf(&sb, " %d (%.1f%%) |", c.FP, 100*c.Retention())
		}
		sb.WriteString("\n")
	}
	// FP categories
	cats := []string{}
	for c := range results[0].ByCat {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool { return results[0].ByCat[cats[i]].FPBase > results[0].ByCat[cats[j]].FPBase })
	fmt.Fprintf(&sb, "\n#### False positives by what the span really was (baseline FP → FP after each configuration)\n\n| span really is | baseline FP |")
	for _, r := range results[1:] {
		fmt.Fprintf(&sb, " %s |", r.Name)
	}
	sb.WriteString("\n|---|---:|")
	for range results[1:] {
		sb.WriteString("---:|")
	}
	sb.WriteString("\n")
	for i, c := range cats {
		if i >= 25 {
			break
		}
		fmt.Fprintf(&sb, "| %s | %d |", c, results[0].ByCat[c].FPBase)
		for _, r := range results[1:] {
			fmt.Fprintf(&sb, " %d |", r.ByCat[c].FP)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Sweep reports FP reduction at several retention targets for one scorer
// (thresholds chosen on devItems, applied to testItems).
func Sweep(devItems, testItems []Item, devScores, testScores []float64, targets []float64, truth map[pii.EntityType]int, name string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "| %s: TP retention target | achieved retention (test) | FP reduction (test) | precision |\n|---|---:|---:|---:|\n", name)
	for _, tg := range targets {
		th, def := ChooseThresholds(devItems, devScores, tg, 100)
		r := Evaluate(testItems, Config{Name: name, Scores: testScores, Thresholds: th, Default: def}, truth, 0)
		fmt.Fprintf(&sb, "| %.1f%% | %.2f%% | %.1f%% | %.1f%% |\n", 100*tg, 100*r.Total.Retention(), 100*r.Total.FPReduction(), 100*r.Total.Precision())
	}
	return sb.String()
}
