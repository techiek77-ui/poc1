package verify

import (
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
)

func aadhaar(prefix string) string { return prefix + string(checksum.VerhoeffCheckDigit(prefix)) }

func scoreAll(t *testing.T, text string, format pii.Format) map[string]float64 {
	t.Helper()
	doc := &pii.Doc{ID: "t", Format: format, Text: text}
	v := detect.NewDocView(doc)
	cs := detect.New(detect.DefaultOptions()).Detect(v)
	out := map[string]float64{}
	for i := range cs {
		a := attrib.Analyze(v, &cs[i], 60)
		s, reason := Rules{}.Score(v, &cs[i], a)
		out[string(cs[i].Type)+":"+cs[i].Value] = s
		t.Logf("%s %q -> %.2f (%s)", cs[i].Type, cs[i].Value, s, reason)
	}
	return out
}

func TestRulesCanonical(t *testing.T) {
	// the user's exact example (digits deliberately repetitive) ...
	checkCanonical(t, aadhaar("90000000000"), aadhaar("90000000001"))
	// ... and realistic-looking numbers
	checkCanonical(t, aadhaar("73819204561"), aadhaar("50934718206"))
}

func checkCanonical(t *testing.T, a, b string) {
	t.Helper()
	s := scoreAll(t, "aadhar "+a+", bank acc "+b, pii.FormatText)
	if s["AADHAAR:"+a] < 0.5 {
		t.Errorf("true Aadhaar should be kept, got %.2f", s["AADHAAR:"+a])
	}
	if s["AADHAAR:"+b] >= 0.5 {
		t.Errorf("bank account claimed as Aadhaar should be suppressed, got %.2f", s["AADHAAR:"+b])
	}
	if s["BANK_ACCOUNT:"+b] < 0.5 {
		t.Errorf("true bank account should be kept, got %.2f", s["BANK_ACCOUNT:"+b])
	}
	if s["BANK_ACCOUNT:"+a] >= 0.5 {
		t.Errorf("Aadhaar claimed as bank account should be suppressed, got %.2f", s["BANK_ACCOUNT:"+a])
	}
}

func TestRulesLabelAfterValueAndCSV(t *testing.T) {
	a := aadhaar("90000000000")
	s := scoreAll(t, "Please note "+a+" (Aadhaar) for the record", pii.FormatText)
	if s["AADHAAR:"+a] < 0.5 {
		t.Errorf("right-adjacent label should attribute, got %.2f", s["AADHAAR:"+a])
	}
	a, b := aadhaar("73819204561"), aadhaar("50934718206")
	s = scoreAll(t, "name,aadhaar_no,account_no\nRamesh Kumar,"+a+","+b+"\n", pii.FormatCSV)
	if s["AADHAAR:"+a] < 0.9 || s["BANK_ACCOUNT:"+b] < 0.9 {
		t.Errorf("header-governed values should score high: %v", s)
	}
	if s["AADHAAR:"+b] >= 0.5 || s["BANK_ACCOUNT:"+a] >= 0.5 {
		t.Errorf("cross-column claims should be suppressed: %v", s)
	}
}

func TestRulesDummyAndMobile(t *testing.T) {
	d := aadhaar("22222222222")
	s := scoreAll(t, "test aadhaar "+d, pii.FormatText)
	if s["AADHAAR:"+d] >= 0.5 {
		t.Errorf("dummy pattern should be suppressed, got %.2f", s["AADHAAR:"+d])
	}
	m := aadhaar("91987654321") // 91 + mobile shape, Verhoeff valid
	s = scoreAll(t, "uid card copy pending; contact "+m, pii.FormatText)
	if s["AADHAAR:"+m] >= 0.5 {
		t.Errorf("91+mobile governed by 'contact' should be suppressed, got %.2f", s["AADHAAR:"+m])
	}
}
