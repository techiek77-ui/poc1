package detect

import (
	"strings"
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
)

func aadhaar(prefix string) string { return prefix + string(checksum.VerhoeffCheckDigit(prefix)) }

func run(t *testing.T, text string, format pii.Format) []pii.Candidate {
	t.Helper()
	doc := &pii.Doc{ID: "t", Format: format, Text: text}
	return New(DefaultOptions()).Detect(NewDocView(doc))
}

func types(cs []pii.Candidate) map[pii.EntityType]int {
	m := map[pii.EntityType]int{}
	for _, c := range cs {
		m[c.Type]++
	}
	return m
}

func TestCanonicalFalsePositive(t *testing.T) {
	a := aadhaar("90000000000")
	b := aadhaar("90000000001")
	text := "aadhar " + a + ", bank acc " + b
	cs := run(t, text, pii.FormatText)
	got := types(cs)
	// The baseline (regex + Verhoeff + keyword in 60 chars) flags BOTH numbers as
	// Aadhaar — this is the false positive the verifiers must remove — and also
	// claims both as bank accounts.
	if got[pii.Aadhaar] != 2 {
		t.Fatalf("baseline should flag 2 Aadhaar candidates, got %v (%+v)", got, cs)
	}
	if got[pii.BankAccount] != 2 {
		t.Fatalf("baseline should flag 2 bank-account candidates, got %v", got)
	}
}

func TestKeywordGate(t *testing.T) {
	a := aadhaar("90000000000")
	if cs := run(t, "ref "+a+" processed", pii.FormatText); len(cs) != 0 {
		t.Fatalf("no keyword -> no candidates, got %+v", cs)
	}
	if cs := run(t, "Aadhaar No: "+a, pii.FormatText); types(cs)[pii.Aadhaar] != 1 {
		t.Fatalf("keyword present -> Aadhaar candidate, got %+v", cs)
	}
	d := New(Options{Window: 60, RequireKeyword: false})
	cs := d.Detect(NewDocView(&pii.Doc{ID: "x", Format: pii.FormatText, Text: "ref " + a + " processed"}))
	if types(cs)[pii.Aadhaar] != 1 || cs[0].KeywordHit {
		t.Fatalf("ungated detection should return the candidate with KeywordHit=false, got %+v", cs)
	}
}

func TestFirstDigitAndVerhoeff(t *testing.T) {
	if cs := run(t, "aadhaar "+aadhaar("10000000000"), pii.FormatText); types(cs)[pii.Aadhaar] != 0 {
		t.Fatal("Aadhaar starting with 1 must be rejected")
	}
	if cs := run(t, "aadhaar 900000000003", pii.FormatText); types(cs)[pii.Aadhaar] != 0 && checksum.Verhoeff("900000000003") == false {
		t.Fatal("Verhoeff-invalid number must be rejected")
	}
	if cs := run(t, "aadhaar 9000 0000 0002 1234", pii.FormatText); types(cs)[pii.Aadhaar] != 0 {
		t.Fatal("prefix of a longer grouped number must be rejected")
	}
}

func TestCSVHeader(t *testing.T) {
	a := aadhaar("90000000000")
	b := aadhaar("90000000001")
	text := "name,aadhaar_no,account_no\nRamesh Kumar," + a + "," + b + "\n"
	cs := run(t, text, pii.FormatCSV)
	var aad, acc []pii.Candidate
	for _, c := range cs {
		switch c.Type {
		case pii.Aadhaar:
			aad = append(aad, c)
		case pii.BankAccount:
			acc = append(acc, c)
		}
	}
	// header "aadhaar_no" gates column 1; the window from column 2 also sees
	// "aadhaar_no"?? No — the header is on the previous line, 60 chars away.
	if len(aad) < 1 {
		t.Fatalf("expected Aadhaar via header, got %+v", cs)
	}
	for _, c := range aad {
		if c.Column == 1 && !c.HeaderHit {
			t.Errorf("column 1 should be header-gated: %+v", c)
		}
	}
	if len(acc) < 1 {
		t.Fatalf("expected bank account via header, got %+v", cs)
	}
}

func TestOtherEntities(t *testing.T) {
	cases := map[string]pii.EntityType{
		"PAN: ABCPE1234F":              pii.PAN,
		"GSTIN 27AAPFU0939F1ZV":        pii.GSTIN,
		"card no 4111 1111 1111 1111":  pii.CreditCard,
		"Voter ID ABC1234567":          pii.VoterID,
		"Passport No: A1234567":        pii.Passport,
		"IFSC: SBIN0001234":            pii.IFSC,
		"mobile +91 98765 43210":       pii.Mobile,
		"email ramesh.kumar@gmail.com": pii.Email,
		"UPI: ramesh@okaxis":           pii.UPI,
		"UAN 100123456789":             pii.UAN,
		"Pincode: 560038":              pii.Pincode,
		"DOB: 12/03/1990":              pii.DOB,
		"DL No: MH12 20110012345":      pii.DrivingLicence,
		"Name: Ramesh Kumar":           pii.Name,
		"Mr. Suresh Patil has applied": pii.Name,
		"Address: 12, MG Road, Indiranagar, Bengaluru - 560038, Karnataka": pii.Address,
	}
	for text, want := range cases {
		if got := types(run(t, text, pii.FormatText)); got[want] == 0 {
			t.Errorf("%q: want %s, got %v", text, want, got)
		}
	}
}

func TestNameNegatives(t *testing.T) {
	for _, text := range []string{"Name: Not Available", "Thank You Team", "Dear Sir Madam"} {
		for _, c := range run(t, text, pii.FormatText) {
			if c.Type == pii.Name && (c.Value == "Not Available" || c.Value == "Thank You Team") {
				t.Errorf("%q: stop words must not form a name: %+v", text, c)
			}
		}
	}
}

func TestCSVQuotedNewline(t *testing.T) {
	a := aadhaar("73819204561")
	text := "code,name,aadhaar_no,notes\nE1,Zoya Khan," + a + ",\"line one\nold acc 509347182063\"\n"
	l := ParseCSVLayout(text)
	if len(l.Rows) != 2 {
		t.Fatalf("rows = %d, want 2 (quoted newline must not split a row)", len(l.Rows))
	}
	col, hdr, ok := l.ColumnAt(strings.Index(text, "509347182063"))
	if !ok || col != 3 || hdr != "notes" {
		t.Fatalf("account in the quoted notes cell mapped to col %d (%q)", col, hdr)
	}
}

func TestNoOverlappingDictionaryNames(t *testing.T) {
	cs := run(t, "Guests: Ramesh Kumar Singh and Priya Sharma attended.", pii.FormatText)
	var names []string
	for _, c := range cs {
		if c.Type == pii.Name {
			names = append(names, c.Value)
		}
	}
	for i := range names {
		for j := range names {
			if i != j && strings.Contains(names[i], names[j]) {
				t.Fatalf("overlapping name candidates: %v", names)
			}
		}
	}
}

func TestCSVStrayQuoteAndDelimiter(t *testing.T) {
	text := "name,mobile,notes\nA,9876543210,5'11\" tall\nB,9123456789,ok\nC,9988776655,fine\n"
	l := ParseCSVLayout(text)
	if len(l.Rows) != 4 {
		t.Fatalf("rows = %d, want 4 (a stray quote must not swallow the file)", len(l.Rows))
	}
	if col, hdr, _ := l.ColumnAt(strings.Index(text, "9988776655")); col != 1 || hdr != "mobile" {
		t.Fatalf("last mobile mapped to col %d %q", col, hdr)
	}
	if d := detectDelim("\"Name, Last\";Mobile\nx;y\n"); d != ';' {
		t.Fatalf("delimiter inside quotes counted: got %q", d)
	}
	cs := run(t, text, pii.FormatCSV)
	if types(cs)[pii.Mobile] != 3 {
		t.Fatalf("all three mobiles should be header-gated, got %v", types(cs))
	}
}

func TestNamesWithInitialsAndTrailingPunctuation(t *testing.T) {
	cases := map[string]string{
		"Name: A. Kumar":                "A. Kumar",
		"Name: R. K. Sharma, Mobile: 9": "R. K. Sharma",
		"Dear Rahul Sharma.":            "Rahul Sharma",
		"Regards, Priya Nair's team":    "Priya Nair",
		"Applicant: S. Ramanathan":      "S. Ramanathan",
	}
	for text, want := range cases {
		found := false
		for _, c := range run(t, text, pii.FormatText) {
			if c.Type == pii.Name && c.Value == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want NAME %q, got %+v", text, want, run(t, text, pii.FormatText))
		}
	}
}

func TestAddressWalkBackKeepsFirstCharacter(t *testing.T) {
	for _, text := range []string{"Deliver to: Flat 302, MG Road, Bangalore 560001", "Ship to: 12, MG Road, Indiranagar, Bengaluru 560038"} {
		for _, c := range run(t, text, pii.FormatText) {
			if c.Type == pii.Address && (strings.HasPrefix(c.Value, "lat ") || strings.HasPrefix(c.Value, "2, ")) {
				t.Errorf("%q: address span lost its first character: %q", text, c.Value)
			}
		}
	}
}

func TestEmailIsNotUPI(t *testing.T) {
	if got := types(run(t, "UPI helpdesk: care@axisbank.com", pii.FormatText)); got[pii.UPI] != 0 {
		t.Fatalf("e-mail reported as UPI: %v", got)
	}
	if got := types(run(t, "UPI: ramesh@okaxis", pii.FormatText)); got[pii.UPI] != 1 {
		t.Fatalf("real UPI handle lost: %v", got)
	}
}

func TestLoneDigitBeforeMobileIsNotAGroup(t *testing.T) {
	d := New(Options{Window: 60, RequireKeyword: false})
	for _, text := range []string{"Sl No 10 9876543210", "Qty 2 +91 98765 43210"} {
		cs := d.Detect(NewDocView(&pii.Doc{ID: "x", Format: pii.FormatText, Text: text}))
		if types(cs)[pii.Mobile] == 0 {
			t.Errorf("%q: mobile lost to the digit-run check", text)
		}
	}
}
