// Package corpus reads labelled data in the three forms it arrives in:
//
//   - documents (JSONL of pii.Doc) as written by piigen;
//
//   - hand-marked text ([[TYPE:value]] / [[-:kind:value]] markup, see pkg/pii/golden);
//
//   - decision records (JSONL) exported from the product's review queue, one
//     analyst decision per line:
//
//     {"text": "...snippet or line...", "format": "text", "start": 7, "end": 19,
//     "type": "AADHAAR", "decision": "fp", "split": "train", "id": "optional"}
//
//     "value" may be given instead of start/end (first occurrence is used).
//     decision is "tp" or "fp". Records become partially labelled documents:
//     only the judged span is used, everything else in the snippet is ignored.
package corpus

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/golden"
)

// Load reads a JSONL file of documents.
func Load(path string) ([]pii.Doc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var docs []pii.Doc
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var d pii.Doc
		if err := json.Unmarshal(sc.Bytes(), &d); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, sc.Err()
}

// Decision is one analyst judgement on a detection.
type Decision struct {
	ID       string     `json:"id"`
	Text     string     `json:"text"`
	Format   pii.Format `json:"format"`
	Start    int        `json:"start"`
	End      int        `json:"end"`
	Value    string     `json:"value"`
	Type     string     `json:"type"`
	Decision string     `json:"decision"`
	Split    string     `json:"split"`
}

// LoadDecisions reads decision records into partially labelled documents.
func LoadDecisions(path string) ([]pii.Doc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var docs []pii.Doc
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	n := 0
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		n++
		var r Decision
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, n, err)
		}
		if r.Value != "" && r.End == 0 {
			i := strings.Index(r.Text, r.Value)
			if i < 0 {
				return nil, fmt.Errorf("%s line %d: value %q not found in text", path, n, r.Value)
			}
			r.Start, r.End = i, i+len(r.Value)
		}
		if r.Start < 0 || r.End > len(r.Text) || r.End <= r.Start {
			return nil, fmt.Errorf("%s line %d: bad span [%d,%d)", path, n, r.Start, r.End)
		}
		if r.Value != "" && r.Text[r.Start:r.End] != r.Value {
			return nil, fmt.Errorf("%s line %d: span [%d,%d) is %q, not value %q", path, n, r.Start, r.End, r.Text[r.Start:r.End], r.Value)
		}
		typ := pii.EntityType(strings.ToUpper(strings.TrimSpace(r.Type)))
		if !KnownType(typ) {
			return nil, fmt.Errorf("%s line %d: unknown entity type %q (known: %v)", path, n, r.Type, pii.AllTypes)
		}
		r.Format = pii.Format(strings.ToLower(string(r.Format)))
		switch r.Format {
		case "", pii.FormatText, pii.FormatCSV, pii.FormatJSON, pii.FormatLog:
		default:
			return nil, fmt.Errorf("%s line %d: unknown format %q", path, n, r.Format)
		}
		r.Split = strings.ToLower(r.Split)
		switch r.Split {
		case "", "train", "dev", "test":
		default:
			return nil, fmt.Errorf("%s line %d: unknown split %q", path, n, r.Split)
		}
		span := pii.TruthSpan{Start: r.Start, End: r.End, Claim: typ}
		switch strings.ToLower(r.Decision) {
		case "tp", "true", "confirmed", "yes":
			span.Type, span.Kind = typ, "analyst_tp"
		case "fp", "false", "rejected", "no":
			span.Kind = "analyst_fp"
		default:
			return nil, fmt.Errorf("%s line %d: decision must be tp or fp, got %q", path, n, r.Decision)
		}
		if r.Format == "" {
			r.Format = pii.FormatText
		}
		id := r.ID
		if id == "" {
			id = fmt.Sprintf("%s#%d", filepath.Base(path), n)
		}
		docs = append(docs, pii.Doc{ID: id, Split: r.Split, Format: r.Format, Shape: "real", Text: r.Text, Truth: []pii.TruthSpan{span}, Partial: true, Source: "real"})
	}
	return docs, sc.Err()
}

// LoadAny picks the reader from the file: .txt/.md are markup, .jsonl is
// either decision records (first object has a "decision" key) or documents.
func LoadAny(path string) ([]pii.Doc, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md":
		docs, err := golden.LoadFile(path)
		for i := range docs {
			docs[i].Source = "real"
			docs[i].Split = ""
			docs[i].ID = filepath.Base(path) + ":" + docs[i].ID
		}
		return docs, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	first := ""
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			first = sc.Text()
			break
		}
	}
	f.Close()
	var probe map[string]json.RawMessage
	if json.Unmarshal([]byte(first), &probe) == nil {
		if _, ok := probe["decision"]; ok {
			return LoadDecisions(path)
		}
	}
	return Load(path)
}

// LoadAll reads several files of any supported kind.
func LoadAll(paths []string) ([]pii.Doc, error) {
	var all []pii.Doc
	for _, p := range paths {
		docs, err := LoadAny(p)
		if err != nil {
			return nil, err
		}
		all = append(all, docs...)
	}
	return all, nil
}

// KnownType reports whether t is one of the detector's entity types.
func KnownType(t pii.EntityType) bool {
	for _, k := range pii.AllTypes {
		if k == t {
			return true
		}
	}
	return false
}

// AssignSplits gives a split to documents that have none (real data). mode is
// "train", "dev", "test" (everything goes there) or "hash" (80/10/10). The hash
// is over the snippet text for partially labelled documents, so several
// decisions about one snippet always land in the same split.
func AssignSplits(docs []pii.Doc, mode string) error {
	switch mode {
	case "train", "dev", "test", "hash":
	default:
		return fmt.Errorf("unknown split mode %q (train|dev|test|hash)", mode)
	}
	for i := range docs {
		if docs[i].Split != "" && docs[i].Split != "golden" {
			continue
		}
		if mode != "hash" {
			docs[i].Split = mode
			continue
		}
		h := fnv.New32a()
		if docs[i].Partial {
			h.Write([]byte(docs[i].Text))
		} else {
			h.Write([]byte(docs[i].ID))
		}
		switch h.Sum32() % 10 {
		case 8:
			docs[i].Split = "dev"
		case 9:
			docs[i].Split = "test"
		default:
			docs[i].Split = "train"
		}
	}
	return nil
}

// Split returns the documents of one split.
func Split(docs []pii.Doc, split string) []pii.Doc {
	var out []pii.Doc
	for _, d := range docs {
		if d.Split == split {
			out = append(out, d)
		}
	}
	return out
}

// Real returns the documents brought in from outside the generator.
func Real(docs []pii.Doc) []pii.Doc {
	var out []pii.Doc
	for _, d := range docs {
		if d.Source == "real" {
			out = append(out, d)
		}
	}
	return out
}

// Paths is a repeatable flag value.
type Paths []string

func (p *Paths) String() string     { return strings.Join(*p, ",") }
func (p *Paths) Set(s string) error { *p = append(*p, strings.Split(s, ",")...); return nil }
