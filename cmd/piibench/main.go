//go:build onnx

// piibench measures the per-candidate cost of an ONNX transformer verifier
// from Go (tokenize + ONNX Runtime), single goroutine, on candidate windows
// exported by piiexport. It answers "can this run on an endpoint?" without
// scoring the whole corpus.
//
//	PII_ORT_LIB=/path/libonnxruntime.so go run -tags onnx ./cmd/piibench -onnx models/tier_b -n 300
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii/onnx"
)

func main() {
	dir := flag.String("onnx", "models/tier_b", "model directory")
	lib := flag.String("ort", os.Getenv("PII_ORT_LIB"), "ONNX Runtime shared library")
	data := flag.String("data", "data/tierb/test_gated.jsonl", "exported windows")
	n := flag.Int("n", 300, "candidates to score")
	flag.Parse()
	if err := onnx.Init(*lib); err != nil {
		fail(err)
	}
	v, err := onnx.New(*dir)
	if err != nil {
		fail(err)
	}
	defer v.Close()
	f, err := os.Open(*data)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	var texts []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() && len(texts) < *n {
		var r struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			texts = append(texts, r.Text)
		}
	}
	for i := 0; i < 10 && i < len(texts); i++ { // warm-up
		v.ScoreText(texts[i])
	}
	if len(texts) == 0 {
		fail(fmt.Errorf("no candidate windows in %s (run piiexport first)", *data))
	}
	durs := make([]time.Duration, 0, len(texts))
	tokens := 0
	for _, t := range texts {
		ids, _, _, err := v.Encode(t)
		if err != nil {
			fail(err)
		}
		tokens += len(ids)
		t0 := time.Now()
		if _, err := v.ScoreText(t); err != nil {
			fail(err)
		}
		durs = append(durs, time.Since(t0))
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	p := func(q float64) time.Duration { return durs[int(q*float64(len(durs)-1))] }
	var total time.Duration
	for _, d := range durs {
		total += d
	}
	fmt.Printf("model %s (%s): %d candidates, mean %d tokens\n", *dir, v.M.Base, len(durs), tokens/len(durs))
	fmt.Printf("latency per candidate: mean %.2f ms, p50 %.2f ms, p95 %.2f ms, max %.2f ms\n",
		float64(total.Microseconds())/float64(len(durs))/1000, float64(p(0.5).Microseconds())/1000, float64(p(0.95).Microseconds())/1000, float64(durs[len(durs)-1].Microseconds())/1000)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
