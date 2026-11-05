package detect

import (
	"regexp"
	"sort"
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
		caps, capsSet := false, false
		lastInitial := false
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
			isInit := initial(w.t)
			isCaps, isTitle := allCaps(w.t) && !isInit, capitalised(w.t) && !allCaps(w.t)
			if !isCaps && !isTitle && !isInit {
				break
			}
			if !isInit { // the casing style comes from the first real word
				if !capsSet {
					caps, capsSet = isCaps, true
				} else if isCaps != caps {
					break
				}
			}
			if cnt == 0 {
				s = w.s
			}
			e = w.e
			lastInitial = isInit
			cnt++
		}
		if cnt > 0 && !lastInitial { // "Sharma." / "Nair's" at a sentence end
			for e > s && (text[e-1] == '.' || text[e-1] == '\'') {
				e--
			}
			if e-s >= 2 && text[e-1] == 's' && text[e-2] == '\'' {
				e -= 2
			}
		}
		return s, e, cnt
	}
	// 1) label-driven
	for _, h := range v.Labels {
		if h.Kind != labels.KName {
			continue
		}
		after := h.End + len(sepRe.FindString(text[h.End:]))
		i := sort.Search(len(words), func(i int) bool { return words[i].s >= after })
		if i < len(words) && words[i].s == after {
			if s, e, n := span(i, 4); n >= 1 {
				emit(s, e, true)
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
		sep := sepRe.FindString(text[h.End:])
		if strings.ContainsAny(sep, "\n\r") {
			// "Address" in a header row does not introduce the next line: that
			// line holds every column's value, not an address
			ls := strings.LastIndexAny(text[:h.Start], "\n\r") + 1
			le := h.End + strings.IndexAny(text[h.End:]+"\n", "\n\r")
			others := 0
			for _, o := range v.LabelsIn(ls, le) {
				if o.Start != h.Start {
					others++
				}
			}
			if others >= 2 {
				continue
			}
		}
		s := h.End + len(sep)
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
		// an address ends at its PIN code (plus an optional state / country)
		if loc := pinRe.FindStringIndex(text[s:e]); loc != nil {
			pe := s + loc[1]
			e = pe
			if m := stateRe.FindString(text[pe:]); m != "" && stateSuffixOK(m) {
				e = pe + len(m)
			}
		}
		if countWords(text[s:e]) >= 2 && hasAddressWord(text[s:e]) {
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
			segBase := start - len(seg) // where this segment's text begins
			if k < len(segs)-1 {
				segBase-- // the comma that followed it
			}
			segStart := segBase
			trimmed := strings.TrimSpace(seg)
			if trimmed == "" && k == len(segs)-1 {
				start = segStart
				continue
			}
			// cut at a label / sentence boundary inside the segment ("Deliver to: Flat 302")
			if idx := strings.LastIndexAny(trimmed, ":;|"); idx >= 0 {
				trimmed = strings.TrimSpace(trimmed[idx+1:])
				segStart = segBase + strings.LastIndex(seg, trimmed)
				if c := segCues(trimmed); c > 0 || len(trimmed) > 0 && k == len(segs)-1 {
					cues += c
					start = segStart
				}
				break
			}
			segStart = segBase + strings.Index(seg, trimmed)
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
		if cues < 2 || !hasAddressWord(text[start:ps]) {
			continue
		}
		e := pe
		if m := stateRe.FindString(text[pe:]); m != "" && stateSuffixOK(m) {
			e = pe + len(m)
		}
		emit(start, e, true)
	}
	return out
}

// segCues counts address evidence in one comma-separated segment: cue words
// and house-number patterns. A segment that is itself a value (e-mail, UPI
// handle, an identifier of 9+ digits) is not part of an address.
func segCues(seg string) int {
	if strings.Contains(seg, "@") || longestDigitRun(seg) >= 9 {
		return 0
	}
	n := 0
	for _, tk := range labels.Tokenize(seg) {
		if len(tk.Text) >= 3 && gazetteer.AddressCues[tk.Text] {
			n++
		}
	}
	if houseRe.MatchString(seg) {
		n++
	}
	return n
}

func longestDigitRun(s string) int {
	best, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

// hasAddressWord requires at least one alphabetic address cue (street,
// locality, city, state, "flat", "near"...) so that a list of numbers never
// passes as an address.
func hasAddressWord(s string) bool {
	for _, tk := range labels.Tokenize(s) {
		if len(tk.Text) >= 3 && gazetteer.AddressCues[tk.Text] {
			return true
		}
	}
	return false
}

func stateSuffixOK(m string) bool {
	ws := strings.Fields(strings.Trim(m, ", "))
	for _, w := range ws {
		if gazetteer.IsState(w) || gazetteer.IsState(strings.Join(ws, " ")) || w == "India" {
			return true
		}
	}
	return false
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
