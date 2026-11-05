// Package corpus reads the JSONL corpus written by piigen.
package corpus

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/techiek77-ui/poc1/pkg/pii"
)

// Load reads all documents.
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
		var d pii.Doc
		if err := json.Unmarshal(sc.Bytes(), &d); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, sc.Err()
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
