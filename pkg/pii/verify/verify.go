// Package verify holds the second-stage verifiers that score baseline
// candidates: a transparent rule verifier and a feature-based linear model
// with pure-Go inference. Both consume the same attrib.Analysis.
package verify

import (
	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
)

// Verifier scores a candidate: probability-like score in [0,1] that the
// candidate really is the claimed entity type, plus a reason for the log.
type Verifier interface {
	Name() string
	Score(v *detect.DocView, c *pii.Candidate, a *attrib.Analysis) (float64, string)
}
