package verify

import (
	"fmt"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

// Rules is configuration 1: no ML. It attributes the value to its governing
// label (header/key, else nearest left label with no value in between, else a
// tightly attached right label), suppresses when a competing label governs,
// and applies number-shape sanity checks.
//
// Policy: never suppress without positive evidence against the claim. The
// absence of a governing label keeps the baseline's decision; only a
// competing label that governs the value, or a dummy digit pattern, pushes a
// candidate below the 0.5 threshold.
type Rules struct{}

func (Rules) Name() string { return "rules" }

func (Rules) Score(v *detect.DocView, c *pii.Candidate, a *attrib.Analysis) (float64, string) {
	kind := labels.TypeKind[c.Type]
	g := a.Governing
	score := 0.5
	reason := ""
	governedSame := g != nil && g.Hit.Kind == kind
	switch {
	case g == nil:
		if a.SameNearest != nil {
			score = 0.6
			reason = fmt.Sprintf("no governing label; '%s' within window (d=%d)", a.SameNearest.Hit.Phrase, a.SameNearest.Dist)
		} else {
			score = 0.52
			reason = "no label in window"
		}
	case governedSame:
		switch {
		case g.Structured():
			score = 0.97
		case g.Hit.Weak:
			score = 0.8
		default:
			score = 0.95
		}
		reason = fmt.Sprintf("governing label '%s' (%s) matches claim", g.Hit.Phrase, g.Side)
	case g.Hit.Kind == labels.KGeneric:
		score = 0.6
		if a.SameNearest != nil && a.SameNearest.Intervening == 0 {
			score = 0.7
		}
		reason = fmt.Sprintf("generic label '%s' governs", g.Hit.Phrase)
	default:
		score = 0.08
		if g.Hit.Weak {
			score = 0.4
		}
		reason = fmt.Sprintf("competing label '%s' (%s) governs, claim is %s", g.Hit.Phrase, g.Hit.Kind, c.Type)
		// "9000 0000 0002 (Aadhaar)": a same-type label tightly attached on the
		// right outranks a left label.
		if s := a.SameNearest; s != nil && g.Side == attrib.Left && s.Side == attrib.Right && s.Intervening == 0 && !s.FollowedByValue && s.Dist <= 25 && (s.Separator == "paren" || s.Separator == "dash" || s.Separator == "colon" || s.Separator == "verb" || s.Separator == "hash") {
			score = 0.6
			reason += fmt.Sprintf("; but '%s' attached on the right", s.Hit.Phrase)
		}
	}
	if s := a.Shape; s != nil {
		switch {
		case s.Dummy():
			if governedSame {
				score *= 0.7 // labelled as the claimed type: keep, but flag
			} else {
				score *= 0.3
			}
			reason += "; dummy digit pattern"
		case s.RepeatRatio >= 0.7:
			if governedSame {
				score *= 0.85
			} else {
				score *= 0.5
			}
			reason += "; repetitive digits"
		}
		if c.Type == pii.Aadhaar && s.Mobile91 && !governedSame {
			score *= 0.5
			reason += "; looks like 91+mobile"
		}
		if c.Type == pii.Aadhaar && s.Groups == "4-4-4" && g == nil {
			score += 0.1
			reason += "; printed in Aadhaar 4-4-4 groups"
		}
	}
	if a.TestContext && (g == nil || !g.Structured()) {
		if a.Shape != nil && a.Shape.Dummy() {
			score *= 0.5 // dummy digits in a test fixture: not a person
		} else {
			score *= 0.8
		}
		reason += "; test/sample wording nearby"
	}
	if ni := a.NameInfo; ni != nil {
		if c.Type == pii.Name {
			if ni.StopWords > 0 {
				score *= 0.3
				reason += "; contains stop word"
			}
			if ni.Words == 1 && ni.DictAny == 0 && !ni.Title && (!governedSame || g.Hit.Weak) {
				score *= 0.5
				reason += "; single non-dictionary word"
			}
			if gazetteer_isStreetWord(ni.NextWord) {
				score *= 0.5
				reason += "; followed by '" + ni.NextWord + "' (street name)"
			}
			if ni.Title && score < 0.6 {
				score = 0.6
			}
		} else if ni.AddrCues == 0 && !ni.HasPIN && !governedSame {
			score *= 0.5
			reason += "; no address cues"
		}
	}
	if score > 0.99 {
		score = 0.99
	}
	if score < 0.01 {
		score = 0.01
	}
	return score, reason
}

var streetWords = map[string]bool{"road": true, "rd": true, "marg": true, "street": true, "st": true, "nagar": true, "lane": true, "chowk": true, "colony": true, "path": true, "circle": true, "layout": true, "bridge": true, "stadium": true, "hospital": true, "college": true, "school": true, "park": true, "garden": true}

func gazetteer_isStreetWord(w string) bool { return streetWords[w] }
