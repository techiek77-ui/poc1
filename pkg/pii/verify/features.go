package verify

import (
	"fmt"
	"strconv"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

// SchemaVersion identifies the feature extractor. A model artifact records
// the schema it was trained with and the loader refuses a mismatch, so a
// retrained model can be shipped on its own as long as the schema is equal.
const SchemaVersion = 1

func bucket(d int) string {
	switch {
	case d <= 2:
		return "0"
	case d <= 8:
		return "1"
	case d <= 20:
		return "2"
	case d <= 40:
		return "3"
	default:
		return "4"
	}
}

func fbucket(f float64) string {
	switch {
	case f < 0.1:
		return "0"
	case f < 0.3:
		return "1"
	case f < 0.5:
		return "2"
	case f < 0.7:
		return "3"
	default:
		return "4"
	}
}

func labelFeats(prefix string, r *attrib.LabelRef) []string {
	if r == nil {
		return []string{prefix + "=none"}
	}
	out := []string{
		prefix + "=" + string(r.Hit.Kind),
		prefix + "_side=" + string(r.Side),
		prefix + "_dist=" + bucket(r.Dist),
		prefix + "_sep=" + r.Separator,
		prefix + "_ntok=" + strconv.Itoa(r.Hit.NTok),
	}
	if r.Intervening > 0 {
		out = append(out, prefix+"_interv")
	}
	if r.Newline {
		out = append(out, prefix+"_nl")
	}
	if r.Hit.OCR {
		out = append(out, prefix+"_ocr")
	}
	if r.FollowedByValue {
		out = append(out, prefix+"_fbv")
	}
	if r.Hit.Weak {
		out = append(out, prefix+"_weak")
	}
	return out
}

// Features turns a candidate and its analysis into sparse binary features.
// Everything here is available in production (no generator metadata).
func Features(c *pii.Candidate, a *attrib.Analysis) []string {
	t := string(c.Type)
	kind := labels.TypeKind[c.Type]
	f := make([]string, 0, 64)
	add := func(s string) { f = append(f, s) }
	addT := func(s string) { f = append(f, s, t+"&"+s) } // with a per-type conjunction

	add("type=" + t)
	addT("fmt=" + string(a.Format))
	if c.KeywordHit {
		addT("kw_hit")
		addT("kw_dist=" + bucket(c.KeywordDist))
	} else {
		addT("kw_miss")
	}
	if c.HeaderHit {
		addT("hdr_hit")
	}

	// governing label
	g := a.Governing
	for _, s := range labelFeats("gov", g) {
		addT(s)
	}
	switch {
	case g == nil:
		addT("gov_none")
	case g.Hit.Kind == kind:
		addT("gov_same")
	case g.Hit.Kind == labels.KGeneric:
		addT("gov_generic")
	default:
		addT("gov_diff")
		if g.Hit.Kind.IsPII() {
			addT("gov_diff_pii")
		} else {
			addT("gov_diff_nonpii")
		}
	}
	for _, s := range labelFeats("same", a.SameNearest) {
		addT(s)
	}
	for _, s := range labelFeats("comp", a.CompNearest) {
		addT(s)
	}
	if a.SameNearest != nil && a.CompNearest != nil {
		if a.SameNearest.Dist < a.CompNearest.Dist {
			addT("same_closer")
		} else {
			addT("comp_closer")
		}
	}
	// every label kind in the window, by side
	seen := map[string]bool{}
	for i := range a.Labels {
		r := &a.Labels[i]
		k := fmt.Sprintf("win_%s=%s", r.Side, r.Hit.Kind)
		if !seen[k] {
			seen[k] = true
			addT(k)
		}
		if r.Intervening == 0 {
			k2 := fmt.Sprintf("adj_%s=%s", r.Side, r.Hit.Kind)
			if !seen[k2] {
				seen[k2] = true
				addT(k2)
			}
		}
	}
	if a.Format == pii.FormatCSV {
		if len(a.HeaderKinds) == 0 {
			addT("hdr_nolabel")
		}
		for _, k := range a.HeaderKinds {
			addT("hdr=" + string(k))
		}
	}
	if a.TestContext {
		addT("testctx")
	}
	if a.Header != "" && len(a.HeaderKinds) == 0 {
		addT("key_nolabel")
	}
	addT("nlabels=" + bucket(a.LabelCount))
	addT("nvals=" + bucket(a.OtherValues))
	addT("linedig=" + fbucket(a.LineDigitPct))
	addT("linelen=" + bucket(a.LineLen/10))

	// number shape
	if s := a.Shape; s != nil {
		addT("len=" + strconv.Itoa(s.Len))
		addT("groups=" + s.Groups)
		addT("first=" + string(s.First))
		addT("rep=" + fbucket(s.RepeatRatio))
		if s.Sequential {
			addT("seq")
		}
		if s.Palindrome {
			addT("palin")
		}
		if s.LeadingZero {
			addT("lead0")
		}
		if s.Mobile91 {
			addT("m91")
		}
		if s.StartsWith1 {
			addT("starts1")
		}
		if s.LuhnOK {
			addT("luhn")
		}
		if s.VerhoeffOK {
			addT("verhoeff")
		}
	} else {
		addT("nonnumeric")
	}

	// context tokens, nearest first
	for i, tok := range a.LeftTokens {
		switch {
		case i == 0:
			addT("l1=" + tok)
			for _, tri := range trigrams(tok) {
				add("l1c=" + tri)
			}
		case i == 1:
			addT("l2=" + tok)
			for _, tri := range trigrams(tok) {
				add("l2c=" + tri)
			}
		case i == 2:
			addT("l3=" + tok)
		default:
			add("lb=" + tok)
		}
	}
	for i, tok := range a.RightTokens {
		switch {
		case i == 0:
			addT("r1=" + tok)
			for _, tri := range trigrams(tok) {
				add("r1c=" + tri)
			}
		case i == 1:
			addT("r2=" + tok)
		default:
			add("rb=" + tok)
		}
	}
	if len(a.LeftTokens) >= 2 {
		addT("l12=" + a.LeftTokens[1] + "_" + a.LeftTokens[0])
	}
	if g != nil {
		addT("gov_kind_sep=" + string(g.Hit.Kind) + "/" + g.Separator)
	}

	// names / addresses
	if ni := a.NameInfo; ni != nil {
		addT("nwords=" + strconv.Itoa(min(ni.Words, 8)))
		if ni.DictFirst {
			addT("dict_first")
		}
		if ni.DictLast {
			addT("dict_last")
		}
		addT("dict_any=" + strconv.Itoa(min(ni.DictAny, 3)))
		if ni.AllCaps {
			addT("allcaps")
		}
		if ni.Title {
			addT("title")
		}
		addT("stop=" + strconv.Itoa(min(ni.StopWords, 2)))
		addT("cues=" + strconv.Itoa(min(ni.AddrCues, 5)))
		if ni.HasPIN {
			addT("haspin")
		}
		if ni.HasCity {
			addT("hascity")
		}
		if ni.HasState {
			addT("hasstate")
		}
		if ni.HasDigits {
			addT("hasdigits")
		}
		if ni.HasComma {
			addT("hascomma")
		}
		addT("spanlen=" + bucket(ni.SpanLength/8))
		if ni.NextWord != "" {
			addT("next=" + ni.NextWord)
		}
	}
	return f
}

func trigrams(s string) []string {
	if len(s) < 3 {
		return []string{s}
	}
	out := make([]string, 0, len(s)-2)
	for i := 0; i+3 <= len(s) && i < 10; i++ {
		out = append(out, s[i:i+3])
	}
	return out
}
