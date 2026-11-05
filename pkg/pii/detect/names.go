package detect

import (
	"regexp"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/gazetteer"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

// Names and addresses cannot be found with a regex. The baseline uses the
// "other way": a label, a title or a dictionary hit proposes a span of
// capitalised words (NAME), and an "address" label or a PIN code with address
// cue words proposes an ADDRESS span. A token-classification NER model would
// replace this candidate generator; the verifier stage stays the same.

var (
	wordRe  = regexp.MustCompile(`[A-Za-z][A-Za-z'.]*`)
	pinRe   = regexp.MustCompile(`\b[1-9]\d{5}\b`)
	houseRe = regexp.MustCompile(`(?i)\b(?:h\.? ?no\.?|house no\.?|flat no\.?|plot no\.?|door no\.?|#)\s*[0-9A-Z/-]+|\b[0-9]{1,4}[A-Z]?[/-][0-9A-Z]{1,4}\b|\b[A-Z]-?[0-9]{1,4}\b`)
	sepRe   = regexp.MustCompile(`^[\s:=\-–—#.>|]*`)
	stateRe = regexp.MustCompile(`^(?:,?\s*(?:[A-Z][a-z]+(?: [A-Z][a-z]+)?)(?:,\s*India)?)`)
)

type word struct {
	s, e int
	t    string
}

func capitalised(w string) bool { return len(w) >= 2 && w[0] >= 'A' && w[0] <= 'Z' }

func allCaps(w string) bool {
	if len(w) < 2 {
		return false
	}
	for i := 0; i < len(w); i++ {
		if w[i] >= 'a' && w[i] <= 'z' {
			return false
		}
	}
	return true
}

func initial(w string) bool { return len(w) == 2 && w[1] == '.' && w[0] >= 'A' && w[0] <= 'Z' }

func (d *Detector) wantType(t pii.EntityType) bool {
	if d.opts.Types == nil {
		return true
	}
	for _, x := range d.opts.Types {
		if x == t {
			return true
		}
	}
	return false
}

func (d *Detector) namesAndAddresses(v *DocView) []pii.Candidate {
	var out []pii.Candidate
	if d.wantType(pii.Name) {
		out = append(out, d.nameCandidates(v)...)
	}
	if d.wantType(pii.Address) {
		out = append(out, d.addressCandidates(v)...)
	}
	return out
}

// nameCandidates proposes NAME spans.
func (d *Detector) nameCandidates(v *DocView) []pii.Candidate {
	text := v.Doc.Text
	var words []word
	for _, loc := range wordRe.FindAllStringIndex(text, -1) {
		words = append(words, word{loc[0], loc[1], text[loc[0]:loc[1]]})
	}
	seen := map[[2]int]bool{}
	var out []pii.Candidate
	emit := func(s, e int, gated bool) {
		if e-s < 3 || seen[[2]int{s, e}] {
			return
		}
		seen[[2]int{s, e}] = true
		c := pii.Candidate{DocID: v.Doc.ID, Start: s, End: e, Value: text[s:e], Type: pii.Name, ChecksumOK: true, KeywordDist: -1, Column: -1}
		d.gate(v, labels.KName, &c)
		if gated {
			c.KeywordHit = true
			if c.KeywordDist < 0 {
				c.KeywordDist = 0
			}
		}
		if d.opts.RequireKeyword && !c.KeywordHit {
			return
		}
		out = append(out, c)
	}
	// span returns up to n consecutive name-like words from index i. Words
	// must share one casing style (Title Case or ALL CAPS; initials fit
	// either), be separated by a single space, and not be stop words.
	span := func(i, n int) (int, int, int) {
		cnt := 0
		s, e := -1, -1
		caps := false
		for j := i; j < len(words) && cnt < n; j++ {
			w := words[j]
			if cnt > 0 {
				gap := text[words[j-1].e:w.s]
				if gap != " " && gap != ". " {
					break
				}
			}
			if gazetteer.IsNameStop(w.t) {
				break
			}
			isCaps, isTitle := allCaps(w.t), capitalised(w.t) && !allCaps(w.t)
			if !isCaps && !isTitle && !initial(w.t) {
				break
			}
			if cnt == 0 {
				caps = isCaps
				s = w.s
			} else if !initial(w.t) && isCaps != caps {
				break
			}
			e = w.e
			cnt++
		}
		return s, e, cnt
	}
	// 1) label-driven
	for _, h := range v.Labels {
		if h.Kind != labels.KName {
			continue
		}
		after := h.End + len(sepRe.FindString(text[h.End:]))
		for i, w := range words {
			if w.s == after {
				if s, e, n := span(i, 4); n >= 1 {
					emit(s, e, true)
				}
				break
			}
		}
	}
	// 2) title-driven and 3) dictionary-driven
	coveredEnd := -1 // a dictionary span already covers words up to here
	for i, w := range words {
		if w.s < coveredEnd {
			continue
		}
		if gazetteer.IsTitle(w.t) {
			if i+1 < len(words) {
				if s, e, n := span(i+1, 3); n >= 1 {
					emit(s, e, true)
				}
			}
			continue
		}
		if !(capitalised(w.t) || allCaps(w.t)) || gazetteer.IsNameStop(w.t) {
			continue
		}
		if s, e, n := span(i, 3); n >= 2 {
			first, last := w.t, ""
			for j := i; j < len(words) && words[j].e <= e; j++ {
				last = words[j].t
			}
			dict := 0
			if gazetteer.IsFirstName(first) {
				dict++
			}
			if gazetteer.IsSurname(last) || gazetteer.IsFirstName(last) {
				dict++
			}
			if dict >= 1 {
				emit(s, e, dict >= 2)
				coveredEnd = e
			}
		}
	}
	return out
}

// addressCandidates proposes ADDRESS spans.
func (d *Detector) addressCandidates(v *DocView) []pii.Candidate {
	text := v.Doc.Text
	var out []pii.Candidate
	seen := map[[2]int]bool{}
	emit := func(s, e int, gated bool) {
		s, e = trimSpan(text, s, e)
		if e-s < 8 || seen[[2]int{s, e}] {
			return
		}
		seen[[2]int{s, e}] = true
		c := pii.Candidate{DocID: v.Doc.ID, Start: s, End: e, Value: text[s:e], Type: pii.Address, ChecksumOK: true, KeywordDist: -1, Column: -1}
		d.gate(v, labels.KAddress, &c)
		if gated {
			c.KeywordHit = true
			if c.KeywordDist < 0 {
				c.KeywordDist = 0
			}
		}
		if d.opts.RequireKeyword && !c.KeywordHit {
			return
		}
		out = append(out, c)
	}
	// 1) label-driven: an "address"-type label, capture until newline / next label of another kind, max 220 chars
	for _, h := range v.Labels {
		if h.Kind != labels.KAddress || !strings.Contains(h.Phrase, "add") && !strings.Contains(h.Phrase, "resid") && !strings.Contains(h.Phrase, "located") {
			continue
		}
		s := h.End + len(sepRe.FindString(text[h.End:]))
		e := s
		limit := min(len(text), s+220)
		for e < limit {
			if text[e] == '\n' || text[e] == '\r' {
				break
			}
			e++
		}
		for _, o := range v.LabelsIn(s, e) {
			if o.Kind != labels.KAddress && o.Kind != labels.KPincode && o.Kind != labels.KGeneric && o.Start > s+4 && o.Start < e {
				e = o.Start
			}
		}
		if countWords(text[s:e]) >= 2 {
			emit(s, e, true)
		}
	}
	// 2) PIN-anchored: walk back over comma-separated segments that look like
	// address parts (cue word, digit, locality), stop at anything else.
	for _, loc := range pinRe.FindAllStringIndex(text, -1) {
		ps, pe := loc[0], loc[1]
		from := max(0, ps-160)
		if nl := strings.LastIndexAny(text[from:ps], "\n\r"); nl >= 0 {
			from = from + nl + 1
		}
		pre := text[from:ps]
		// segments right-to-left
		start := ps
		cues := 0
		segs := strings.Split(pre, ",")
		for k := len(segs) - 1; k >= 0; k-- {
			seg := segs[k]
			segStart := start - len(seg)
			if k < len(segs)-1 {
				segStart-- // the comma
			}
			trimmed := strings.TrimSpace(seg)
			if trimmed == "" && k == len(segs)-1 {
				start = segStart
				continue
			}
			// cut at a label / sentence boundary inside the segment ("Deliver to: Flat 302")
			if idx := strings.LastIndexAny(trimmed, ":;|"); idx >= 0 {
				trimmed = strings.TrimSpace(trimmed[idx+1:])
				segStart = start - len(trimmed)
				if c := segCues(trimmed); c > 0 || len(trimmed) > 0 && k == len(segs)-1 {
					cues += c
					start = segStart
				}
				break
			}
			c := segCues(trimmed)
			if c == 0 || len(trimmed) > 48 {
				break
			}
			cues += c
			start = segStart
		}
		if start > ps-3 { // nothing before the PIN
			continue
		}
		if cues < 2 {
			continue
		}
		e := pe
		if m := stateRe.FindString(text[pe:]); m != "" {
			ws := strings.Fields(strings.Trim(m, ", "))
			ok := false
			for _, w := range ws {
				if gazetteer.IsState(w) || gazetteer.IsState(strings.Join(ws, " ")) || w == "India" {
					ok = true
				}
			}
			if ok {
				e = pe + len(m)
			}
		}
		emit(start, e, true)
	}
	return out
}

// segCues counts address evidence in one comma-separated segment.
func segCues(seg string) int {
	n := 0
	for _, tk := range labels.Tokenize(seg) {
		if gazetteer.AddressCues[tk.Text] {
			n++
		}
	}
	if houseRe.MatchString(seg) {
		n++
	}
	if strings.ContainsAny(seg, "0123456789") && n == 0 {
		n++ // house / plot numbers
	}
	return n
}

func countWords(s string) int { return len(strings.Fields(s)) }

func trimSpan(text string, s, e int) (int, int) {
	for s < e && strings.ContainsRune(" \t:,;-|.", rune(text[s])) {
		s++
	}
	for e > s && strings.ContainsRune(" \t,;|", rune(text[e-1])) {
		e--
	}
	return s, e
}
