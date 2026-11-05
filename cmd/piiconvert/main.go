// piiconvert turns a span-labelled dataset in the usual NER layout
// ({"text": ..., "spans": [{"start": .., "end": .., "label": ..}]}) into the
// document corpus piitrain/piieval read, mapping the dataset's label names to
// this detector's entity types.
//
//	go run ./cmd/piiconvert -in theirs.jsonl -out data/theirs.jsonl \
//	    -spans-field privacy_mask -map 'AADHAAR_NUMBER=AADHAAR,PAN_NUMBER=PAN,PERSON=NAME,STREET_ADDRESS=ADDRESS'
//
// Labels missing from the map are an error unless -unmapped says what to do:
// "drop" (default when given) removes them and marks the document partially
// labelled, so nothing is assumed about the dropped spans; "nonpii" keeps them
// as non-PII look-alikes (only right when the dataset labels no PII outside
// the map). Offsets are interpreted as rune indexes (Python convention) or
// bytes, decided once per file; when a span object carries its own text the
// conversion is verified against it.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/corpus"
)

type rawSpan struct {
	start, end int
	label      string
	value      string // the span's own text, if the dataset provides one
}

type rawDoc struct {
	text  string
	spans []rawSpan
}

func main() {
	in := flag.String("in", "", "input JSONL (one object per line)")
	out := flag.String("out", "", "output JSONL corpus")
	textField := flag.String("text-field", "text", "field holding the document text")
	spansField := flag.String("spans-field", "", "field holding the span list (default: first of spans, entities, privacy_mask, labels)")
	startField := flag.String("start-field", "start", "span start field")
	endField := flag.String("end-field", "end", "span end field")
	labelField := flag.String("label-field", "label", "span label field")
	offsets := flag.String("offsets", "auto", "byte|rune|auto: how the dataset counts offsets (decided once per file)")
	mapping := flag.String("map", "", "label map THEIRS=OURS,...; OURS is an entity type (AADHAAR, PAN, NAME, ...) or '-' for non-PII")
	unmapped := flag.String("unmapped", "", "drop|nonpii: what to do with labels not in the map (required when any exist)")
	format := flag.String("format", "text", "file format to record (text|csv|json|log)")
	split := flag.String("split", "hash", "split to assign: train|dev|test|hash")
	flag.Parse()
	if *in == "" || *out == "" {
		fail(fmt.Errorf("-in and -out are required"))
	}
	*format = strings.ToLower(*format)
	switch pii.Format(*format) {
	case pii.FormatText, pii.FormatCSV, pii.FormatJSON, pii.FormatLog:
	default:
		fail(fmt.Errorf("-format must be text|csv|json|log, got %q", *format))
	}
	*split = strings.ToLower(*split)
	switch *split {
	case "train", "dev", "test", "hash":
	default:
		fail(fmt.Errorf("-split must be train|dev|test|hash, got %q", *split))
	}
	*unmapped = strings.ToLower(*unmapped)
	switch *unmapped {
	case "", "drop", "nonpii":
	default:
		fail(fmt.Errorf("-unmapped must be drop|nonpii, got %q", *unmapped))
	}
	switch *offsets {
	case "auto", "byte", "rune":
	default:
		fail(fmt.Errorf("-offsets must be byte|rune|auto, got %q", *offsets))
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
		if v != "-" && !corpus.KnownType(pii.EntityType(v)) {
			fail(fmt.Errorf("unknown entity type %q in -map (known: %v)", v, pii.AllTypes))
		}
		labelMap[strings.ToUpper(strings.TrimSpace(k))] = pii.EntityType(v)
	}

	// pass 1: read everything
	raws, err := readAll(*in, *textField, *spansField, *startField, *endField, *labelField)
	if err != nil {
		fail(err)
	}
	// decide byte vs rune once per file
	useRunes := *offsets == "rune"
	if *offsets == "auto" {
		useRunes = decideRunes(raws)
		if useRunes {
			fmt.Fprintln(os.Stderr, "note: offsets interpreted as rune indexes (text has multi-byte characters and every span stays valid that way); pass -offsets byte to override")
		}
	}
	// pass 2: convert
	unknownLabels := map[string]int{}
	var docs []pii.Doc
	spans, dropped := 0, 0
	for n, rd := range raws {
		doc := pii.Doc{ID: fmt.Sprintf("%s#%d", filepath.Base(*in), n+1), Format: pii.Format(*format), Shape: "real", Text: rd.text, Source: "real"}
		for _, sp := range rd.spans {
			st, en := sp.start, sp.end
			if useRunes {
				st, en = runeToByte(rd.text, st), runeToByte(rd.text, en)
			}
			if st < 0 || en > len(rd.text) || en <= st {
				fail(fmt.Errorf("line %d: span [%d,%d) outside text (offsets=%s)", n+1, sp.start, sp.end, *offsets))
			}
			if sp.value != "" && rd.text[st:en] != sp.value {
				fail(fmt.Errorf("line %d: span [%d,%d) is %q but the dataset says %q — wrong -offsets?", n+1, sp.start, sp.end, rd.text[st:en], sp.value))
			}
			typ, ok := labelMap[strings.ToUpper(sp.label)]
			if !ok {
				unknownLabels[sp.label]++
				if *unmapped != "nonpii" {
					dropped++
					doc.Partial = true
					continue
				}
				typ = "-"
			}
			ts := pii.TruthSpan{Start: st, End: en, Kind: strings.ToLower(sp.label)}
			if typ != "-" {
				ts.Type = typ
				ts.Kind = strings.ToLower(string(typ))
			}
			doc.Truth = append(doc.Truth, ts)
			spans++
		}
		docs = append(docs, doc)
	}
	if len(unknownLabels) > 0 && *unmapped == "" {
		var names []string
		for l := range unknownLabels {
			names = append(names, l)
		}
		sort.Strings(names)
		fail(fmt.Errorf("labels not in -map: %v — add them to -map, or pass -unmapped drop (ignore them, documents become partially labelled) or -unmapped nonpii (treat them as non-PII look-alikes)", names))
	}
	if err := corpus.AssignSplits(docs, *split); err != nil {
		fail(err)
	}
	if err := writeDocs(*out, docs); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %d documents, %d spans (%d dropped) to %s\n", len(docs), spans, dropped, *out)
	if len(unknownLabels) > 0 {
		fmt.Printf("labels not in -map (%s): %v\n", *unmapped, unknownLabels)
	}
}

func readAll(path, textField, spansField, startField, endField, labelField string) ([]rawDoc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	var out []rawDoc
	n := 0
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		n++
		var rec map[string]json.RawMessage
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		var text string
		if err := json.Unmarshal(rec[textField], &text); err != nil {
			return nil, fmt.Errorf("line %d: field %q is not a string", n, textField)
		}
		sf := spansField
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
				return nil, fmt.Errorf("line %d: field %q is not a list of objects", n, sf)
			}
		}
		rd := rawDoc{text: text}
		for _, sp := range raw {
			var rs rawSpan
			if json.Unmarshal(sp[startField], &rs.start) != nil || json.Unmarshal(sp[endField], &rs.end) != nil || json.Unmarshal(sp[labelField], &rs.label) != nil {
				return nil, fmt.Errorf("line %d: span %v lacks %s/%s/%s", n, sp, startField, endField, labelField)
			}
			for _, vf := range []string{"value", "text", "span_text", "entity", "word"} {
				var v string
				if raw, ok := sp[vf]; ok && json.Unmarshal(raw, &v) == nil && v != "" {
					rs.value = v
					break
				}
			}
			rd.spans = append(rd.spans, rs)
		}
		out = append(out, rd)
	}
	return out, sc.Err()
}

// decideRunes returns true when the file contains multi-byte text and every
// span is valid under the rune interpretation (and, where span texts are
// given, matches them), i.e. when the dataset evidently counts runes.
func decideRunes(raws []rawDoc) bool {
	multi, runeOK, byteOK := false, true, true
	for _, rd := range raws {
		if len(rd.text) != utf8.RuneCountInString(rd.text) {
			multi = true
		}
		for _, sp := range rd.spans {
			bs, be := sp.start, sp.end
			if bs < 0 || be > len(rd.text) || be <= bs || !utf8.RuneStart(rd.text[bs]) || (be < len(rd.text) && !utf8.RuneStart(rd.text[be])) || (sp.value != "" && rd.text[bs:be] != sp.value) {
				byteOK = false
			}
			rs, re := runeToByte(rd.text, sp.start), runeToByte(rd.text, sp.end)
			if rs < 0 || re > len(rd.text) || re <= rs || (sp.value != "" && rd.text[rs:re] != sp.value) {
				runeOK = false
			}
		}
	}
	if !multi {
		return false // identical either way
	}
	return runeOK && !byteOK || (runeOK && byteOK) // prefer runes (Python tooling) when both fit
}

func writeDocs(path string, docs []pii.Doc) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	for i := range docs {
		if err := enc.Encode(&docs[i]); err != nil {
			f.Close()
			return err
		}
	}
	return f.Close()
}

// runeToByte converts a rune index to a byte offset (invalid bytes count as one rune).
func runeToByte(s string, r int) int {
	i := 0
	for n := 0; i < len(s) && n < r; n++ {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return i
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
