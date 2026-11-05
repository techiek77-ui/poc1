package attrib

import (
	"testing"
	"unicode/utf8"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

func analyze(t *testing.T, text string, format pii.Format, typ pii.EntityType, value string) *Analysis {
	t.Helper()
	doc := &pii.Doc{ID: "t", Format: format, Text: text}
	v := detect.NewDocView(doc)
	for _, c := range detect.New(detect.Options{Window: 60}).Detect(v) {
		if c.Type == typ && c.Value == value {
			return Analyze(v, &c, 60)
		}
	}
	t.Fatalf("no %s candidate %q in %q", typ, value, text)
	return nil
}

func gov(a *Analysis) (labels.Kind, Side) {
	if a.Governing == nil {
		return "", ""
	}
	return a.Governing.Hit.Kind, a.Governing.Side
}

func TestGoverningNearestLeftWithoutInterveningValue(t *testing.T) {
	a := analyze(t, "aadhar 738192045614, bank acc 509347182063", pii.FormatText, pii.Aadhaar, "509347182063")
	if k, s := gov(a); k != labels.KBankAccount || s != Left {
		t.Fatalf("second number should be governed by 'bank acc', got %v %v", k, s)
	}
	a = analyze(t, "aadhar 738192045614, bank acc 509347182063", pii.FormatText, pii.Aadhaar, "738192045614")
	if k, _ := gov(a); k != labels.KAadhaar {
		t.Fatalf("first number should be governed by 'aadhar', got %v", k)
	}
}

func TestRightLabelNeedsTightSeparator(t *testing.T) {
	a := analyze(t, "Please note 738192045614 (Aadhaar) for the record", pii.FormatText, pii.Aadhaar, "738192045614")
	if k, s := gov(a); k != labels.KAadhaar || s != Right {
		t.Fatalf("parenthesised right label should govern, got %v %v", k, s)
	}
	a = analyze(t, "Customer called from 919830472151 regarding Aadhaar update", pii.FormatText, pii.Aadhaar, "919830472151")
	if a.Governing != nil {
		t.Fatalf("'regarding Aadhaar' must not govern, got %+v", a.Governing.Hit)
	}
}

func TestWeakLabelSkippedForSpecificOne(t *testing.T) {
	a := analyze(t, "Permanent Account Number Card\nBNZPM2501F", pii.FormatText, pii.PAN, "BNZPM2501F")
	if k, _ := gov(a); k != labels.KPAN {
		t.Fatalf("'card' is weak; 'permanent account number' should govern, got %v", k)
	}
}

func TestStructuredKeys(t *testing.T) {
	a := analyze(t, "name,aadhaar_no,account_no\nRamesh Kumar,738192045614,509347182063\n", pii.FormatCSV, pii.Aadhaar, "509347182063")
	if k, s := gov(a); k != labels.KBankAccount || s != Header {
		t.Fatalf("CSV header should govern, got %v %v", k, s)
	}
	sql := "INSERT INTO kyc (id, full_name, aadhaar_no, pan_no, acct_no) VALUES (1, 'Harish Bhat', '840162937518', 'AEXPB7712M', '293847561021');"
	a = analyze(t, sql, pii.FormatText, pii.Aadhaar, "840162937518")
	if k, s := gov(a); k != labels.KAadhaar || s != Key || a.Header != "aadhaar_no" {
		t.Fatalf("SQL column should govern the Aadhaar, got %v %v header=%q", k, s, a.Header)
	}
	a = analyze(t, sql, pii.FormatText, pii.Aadhaar, "293847561021")
	if k, s := gov(a); k != labels.KBankAccount || s != Key {
		t.Fatalf("SQL column should govern the account, got %v %v", k, s)
	}
}

func TestTextClaimsIgnoreNumericLabels(t *testing.T) {
	a := analyze(t, "(carry Aadhaar for open-box delivery). Deliver to: Pooja Nair, Flat 302", pii.FormatText, pii.Name, "Pooja Nair")
	if a.Governing != nil && a.Governing.Hit.Kind == labels.KAadhaar {
		t.Fatalf("a numeric-ID label must not govern a name")
	}
}

func TestShape(t *testing.T) {
	s := shapeOf("2345 6789 0124", "234567890124")
	if s.SeqRun < 9 || !s.Dummy() {
		t.Fatalf("long ascending run should be dummy: %+v", s)
	}
	s = shapeOf("1234 1234 1234", "123412341234")
	if !s.RepeatGroup || !s.Dummy() {
		t.Fatalf("repeated group should be dummy: %+v", s)
	}
	s = shapeOf("919830472151", "919830472151")
	if !s.Mobile91 || s.Dummy() {
		t.Fatalf("91+mobile shape, not dummy: %+v", s)
	}
	if s.Groups != "12" {
		t.Fatalf("groups = %q", s.Groups)
	}
	if g := shapeOf("7381 9204 5614", "738192045614").Groups; g != "4-4-4" {
		t.Fatalf("groups = %q", g)
	}
}

func TestWindowNeverSplitsUTF8(t *testing.T) {
	// 61 bytes of Devanagari before the value: a byte window would cut a rune.
	text := "मेरा आधार मेरी पहचान मेरा आधार मेरी पहचान मेरा Aadhaar: 738192045614 ₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹₹"
	a := analyze(t, text, pii.FormatText, pii.Aadhaar, "738192045614")
	if !utf8.ValidString(a.Left) || !utf8.ValidString(a.Right) {
		t.Fatalf("window boundaries split a character: left=%q right=%q", a.Left, a.Right)
	}
}
