package golden_test

import (
	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
)

func analyzeFor(v *detect.DocView, c *pii.Candidate) *attrib.Analysis {
	return attrib.Analyze(v, c, 60)
}
