// piiscan runs the baseline detector and the verifiers over a file (or
// stdin) and prints every detection with its scores and reasons. It is the
// quickest way to see what the verifiers do on your own text.
//
//	echo 'aadhar 900000000002, bank acc 900000000018' | go run ./cmd/piiscan
//	go run ./cmd/piiscan -format csv customers.csv
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/verify"
)

func main() {
	format := flag.String("format", "text", "text|csv|json|log (what the product knows from the file type)")
	modelPath := flag.String("model", "models/tier_a_lr.json", "Tier A model artifact (optional)")
	window := flag.Int("window", 60, "context window")
	ungated := flag.Bool("ungated", false, "also show candidates without a keyword in the window")
	flag.Parse()

	var data []byte
	var err error
	if flag.NArg() > 0 {
		data, err = os.ReadFile(flag.Arg(0))
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	doc := &pii.Doc{ID: "input", Format: pii.Format(*format), Text: string(data)}
	v := detect.NewDocView(doc)
	det := detect.New(detect.Options{Window: *window, RequireKeyword: !*ungated})
	cands := det.Detect(v)

	var lin *verify.Linear
	if m, err := verify.Load(*modelPath); err == nil {
		if err := m.CheckWindow(*window); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		lin = &verify.Linear{M: m}
	} else {
		fmt.Fprintf(os.Stderr, "(no model loaded: %v)\n", err)
	}
	rules := verify.Rules{}
	fmt.Printf("%-15s %-28s %-8s %-7s %-7s %s\n", "TYPE", "VALUE", "KEYWORD", "RULES", "LINEAR", "WHY")
	for i := range cands {
		c := &cands[i]
		a := attrib.Analyze(v, c, *window)
		rs, rr := rules.Score(v, c, a)
		ls, lr := -1.0, ""
		keep := "-"
		if lin != nil {
			ls, lr = lin.Score(v, c, a)
			if ls >= lin.M.Threshold(c.Type) {
				keep = "KEEP"
			} else {
				keep = "DROP"
			}
		}
		val := c.Value
		if len(val) > 26 {
			val = val[:23] + "..."
		}
		kw := "no"
		if c.KeywordHit {
			kw = fmt.Sprintf("yes@%d", c.KeywordDist)
		}
		lscore := "  n/a"
		if ls >= 0 {
			lscore = fmt.Sprintf("%.3f", ls)
		}
		fmt.Printf("%-15s %-28s %-8s %.2f    %s %s | rules: %s | model: %s\n", c.Type, val, kw, rs, lscore, keep, rr, strings.TrimSpace(lr))
	}
	if len(cands) == 0 {
		fmt.Println("(no candidates)")
	}
}
