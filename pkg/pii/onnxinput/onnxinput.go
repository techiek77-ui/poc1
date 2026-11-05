// Package onnxinput builds the text a transformer verifier reads for one
// candidate. It is shared by the exporter (training data) and the runtime
// (inference) so the two can never drift apart.
package onnxinput

import (
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
)

// Build renders: "<type> | <key> | <left> « value » <right>" with whitespace
// collapsed. The claimed type and the structured key (CSV header / SQL
// column) are prepended so a single model serves every entity type and sees
// what a human reviewer would see.
func Build(c *pii.Candidate, a *attrib.Analysis) string {
	var sb strings.Builder
	sb.WriteString(strings.ToLower(string(c.Type)))
	sb.WriteString(" | ")
	if a.Header != "" {
		sb.WriteString(strings.ToLower(a.Header))
	} else {
		sb.WriteString("-")
	}
	sb.WriteString(" | ")
	sb.WriteString(collapse(a.Left))
	sb.WriteString(" « ")
	sb.WriteString(collapse(c.Value))
	sb.WriteString(" » ")
	sb.WriteString(collapse(a.Right))
	return sb.String()
}

// Clean is the sanitation every model input goes through (idempotent): valid
// UTF-8, no square brackets (a literal "[SEP]" must not become a special
// token), line breaks as ⏎, single spaces. The Go runtime applies it again in
// Encode and the Python reference generation applies the same function, so
// raw strings in reference.json are compared on equal terms.
func Clean(s string) string { return collapse(s) }

func collapse(s string) string {
	s = strings.ToValidUTF8(s, "")                         // a tokenizer must never see a half character
	s = strings.NewReplacer("[", " ", "]", " ").Replace(s) // "[SEP]" in text must not become a special token
	s = strings.ReplaceAll(s, "\n", " ⏎ ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.Join(strings.Fields(s), " ")
}
