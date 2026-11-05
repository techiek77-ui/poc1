// piiconvert turns a span-labelled dataset in the usual NER layout
// ({"text": ..., "spans": [{"start": .., "end": .., "label": ..}]}) into the
// document corpus piitrain/piieval read, mapping the dataset's label names to
// this detector's entity types.
//
//	go run ./cmd/piiconvert -in theirs.jsonl -out data/theirs.jsonl \
//	    -spans-field privacy_mask -map 'AADHAAR_NUMBER=AADHAAR,PAN_NUMBER=PAN,PERSON=NAME,STREET_ADDRESS=ADDRESS'
//
// Labels not in the map are kept as non-PII look-alikes (their name becomes
// the category in the evaluation) with -unmapped nonpii (default), or dropped
// with -unmapped drop, which also marks the document as partially labelled so
// nothing is assumed about the unlabelled rest.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
)

func main() {
	in := flag.String("in", "", "input JSONL (one object per line)")
	out := flag.String("out", "", "output JSONL corpus")
	textField := flag.String("text-field", "text", "field holding the document text")
	spansField := flag.String("spans-field", "", "field holding the span list (default: first of spans, entities, privacy_mask, labels)")
	startField := flag.String("start-field", "start", "span start field (byte or rune offset, see -offsets)")
	endField := flag.String("end-field", "end", "span end field")
	labelField := flag.String("label-field", "label", "span label field")
	offsets := flag.String("offsets", "auto", "byte|rune|auto: how the dataset counts offsets (python datasets usually count runes)")
	mapping := flag.String("map", "", "label map THEIRS=OURS,...; OURS is an entity type (AADHAAR, PAN, NAME, ...) or '-' for non-PII")
	unmapped := flag.String("unmapped", "nonpii", "nonpii|drop: what to do with labels not in the map")
	format := flag.String("format", "text", "file format to record (text|csv|json|log)")
	split := flag.String("split", "hash", "split to assign: train|dev|test|hash")
	flag.Parse()
	if *in == "" || *out == "" {
		fail(fmt.Errorf("-in and -out are required"))
	}
	known := map[pii.EntityType]bool{}
	for _, t := range pii.AllTypes {
		known[t] = true
	}
	labelMap := map[string]pii.EntityType{}
	for _, kv := range strings.Split(*mapping, ",") {
		if kv = strings.TrimSpace(kv); kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			fail(fmt.Errorf("bad -map entry %q", kv))
		}
		v = strings.ToUpper(strings.TrimSpace(v))
		if v != "-" && !known[pii.EntityType(v)] {
			fail(fmt.Errorf("unknown entity type %q in -map (known: %v)", v, pii.AllTypes))
		}
		labelMap[strings.ToUpper(strings.TrimSpace(k))] = pii.EntityType(v)
	}

	f, err := os.Open(*in)
	if err != nil {
		fail(err)
	}
	defer f.Close()
	o, err := os.Create(*out)
	if err != nil {
		fail(err)
	}
	defer o.Close()
	enc := json.NewEncoder(o)
	enc.SetEscapeHTML(false)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	n, written, spans, dropped := 0, 0, 0, 0
	unknownLabels := map[string]int{}
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		n++
		var rec map[string]json.RawMessage
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			fail(fmt.Errorf("line %d: %w", n, err))
		}
		var text string
		if err := json.Unmarshal(rec[*textField], &text); err != nil {
			fail(fmt.Errorf("line %d: field %q is not a string", n, *textField))
		}
		sf := *spansField
		if sf == "" {
			for _, c := range []string{"spans", "entities", "privacy_mask", "labels"} {
				if _, ok := rec[c]; ok {
					sf = c
					break
				}
			}
		}
		var raw []map[string]json.RawMessage
		if sf != "" {
			if err := json.Unmarshal(rec[sf], &raw); err != nil {
				fail(fmt.Errorf("line %d: field %q is not a list of objects", n, sf))
			}
		}
		runeOffsets := *offsets == "rune" || (*offsets == "auto" && len(text) != len([]rune(text)))
		doc := pii.Doc{ID: fmt.Sprintf("%s#%d", filepath.Base(*in), n), Format: pii.Format(*format), Shape: "real", Text: text, Source: "real"}
		for _, sp := range raw {
			var st, en int
			var label string
			if json.Unmarshal(sp[*startField], &st) != nil || json.Unmarshal(sp[*endField], &en) != nil || json.Unmarshal(sp[*labelField], &label) != nil {
				fail(fmt.Errorf("line %d: span %v lacks %s/%s/%s", n, sp, *startField, *endField, *labelField))
			}
			if runeOffsets {
				st, en = runeToByte(text, st), runeToByte(text, en)
			}
			if st < 0 || en > len(text) || en <= st {
				fail(fmt.Errorf("line %d: span [%d,%d) outside text", n, st, en))
			}
			typ, ok := labelMap[strings.ToUpper(label)]
			if !ok {
				unknownLabels[label]++
				if *unmapped == "drop" {
					dropped++
					doc.Partial = true
					continue
				}
				typ = "-"
			}
			ts := pii.TruthSpan{Start: st, End: en, Kind: strings.ToLower(label)}
			if typ != "-" {
				ts.Type = typ
				ts.Kind = strings.ToLower(string(typ))
			}
			doc.Truth = append(doc.Truth, ts)
			spans++
		}
		if err := enc.Encode(&doc); err != nil {
			fail(err)
		}
		written++
	}
	if err := sc.Err(); err != nil {
		fail(err)
	}
	docs, _ := corpus.Load(*out) // re-read to assign splits consistently
	corpus.AssignSplits(docs, *split)
	o.Seek(0, 0)
	o.Truncate(0)
	enc = json.NewEncoder(o)
	enc.SetEscapeHTML(false)
	for i := range docs {
		enc.Encode(&docs[i])
	}
	fmt.Printf("wrote %d documents, %d spans (%d dropped) to %s\n", written, spans, dropped, *out)
	if len(unknownLabels) > 0 {
		fmt.Printf("labels not in -map (kept as non-PII unless -unmapped drop): %v\n", unknownLabels)
	}
}

// runeToByte converts a rune index to a byte offset.
func runeToByte(s string, r int) int {
	i := 0
	for n := 0; i < len(s) && n < r; n++ {
		_, size := decodeRune(s[i:])
		i += size
	}
	return i
}

func decodeRune(s string) (rune, int) {
	for i, r := range s {
		if i == 0 {
			return r, len(string(r))
		}
	}
	return 0, 1
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
