// Package golden loads hand-written evaluation documents. They are written
// in styles the synthetic generator does not produce, so they test whether
// the verifiers generalise beyond their training distribution.
//
// Format: documents are separated by a header line
//
//	=== <format> <shape> <free text title>
//
// and PII values are marked inline as [[TYPE:value]]; non-PII look-alikes as
// [[-:kind:value]]. Markers are removed when loading and become truth spans.
package golden

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
)

var markerRe = regexp.MustCompile(`\[\[([A-Z_]+|-):(?:([a-z_0-9]+):)?(.*?)\]\]`)

// Parse converts the marked-up text into documents.
func Parse(src string) ([]pii.Doc, error) {
	var docs []pii.Doc
	var cur *pii.Doc
	var body strings.Builder
	flush := func() error {
		if cur == nil {
			return nil
		}
		text, truth, err := unmark(strings.TrimRight(body.String(), "\n") + "\n")
		if err != nil {
			return fmt.Errorf("%s: %w", cur.ID, err)
		}
		cur.Text, cur.Truth = text, truth
		docs = append(docs, *cur)
		body.Reset()
		cur = nil
		return nil
	}
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	n := 0
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "=== ") {
			if err := flush(); err != nil {
				return nil, err
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return nil, fmt.Errorf("bad header %q", line)
			}
			n++
			cur = &pii.Doc{ID: fmt.Sprintf("g%03d", n), Split: "golden", Format: pii.Format(parts[1]), Shape: parts[2]}
			continue
		}
		if cur != nil {
			body.WriteString(line + "\n")
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return docs, nil
}

func unmark(s string) (string, []pii.TruthSpan, error) {
	var out strings.Builder
	var truth []pii.TruthSpan
	last := 0
	for _, m := range markerRe.FindAllStringSubmatchIndex(s, -1) {
		out.WriteString(s[last:m[0]])
		typ := s[m[2]:m[3]]
		kind := ""
		if m[4] >= 0 {
			kind = s[m[4]:m[5]]
		}
		val := s[m[6]:m[7]]
		st := out.Len()
		out.WriteString(val)
		t := pii.TruthSpan{Start: st, End: out.Len(), Kind: kind}
		if typ != "-" {
			t.Type = pii.EntityType(typ)
			if kind == "" {
				t.Kind = strings.ToLower(typ)
			}
		} else if kind == "" {
			return "", nil, fmt.Errorf("non-PII marker needs a kind: %q", s[m[0]:m[1]])
		}
		truth = append(truth, t)
		last = m[1]
	}
	out.WriteString(s[last:])
	return out.String(), truth, nil
}

// LoadFile reads and parses a golden file.
func LoadFile(path string) ([]pii.Doc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(b))
}
