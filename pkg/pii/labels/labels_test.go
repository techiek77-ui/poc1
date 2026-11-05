package labels

import (
	"testing"

	"github.com/techiek77-ui/poc1/pkg/pii"
)

func kinds(hits []Hit) []Kind {
	var ks []Kind
	for _, h := range hits {
		ks = append(ks, h.Kind)
	}
	return ks
}

func TestTokenizeSplitsIdentifiers(t *testing.T) {
	toks := Tokenize("aadhaarNo=9000; acct_no: 12, Aadhaar_Number")
	var got []string
	for _, tk := range toks {
		got = append(got, tk.Text)
	}
	want := []string{"aadhaar", "no", "9000", "acct", "no", "12", "aadhaar", "number"}
	if len(got) != len(want) {
		t.Fatalf("tokens = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tokens = %v, want %v", got, want)
		}
	}
}

func TestFindCanonicalLine(t *testing.T) {
	hits := Find("aadhar 900000000002, bank acc 900000000018")
	if len(hits) != 2 {
		t.Fatalf("hits = %+v, want 2", hits)
	}
	if hits[0].Kind != KAadhaar || hits[0].Type != pii.Aadhaar {
		t.Errorf("first hit = %+v, want aadhaar", hits[0])
	}
	if hits[1].Kind != KBankAccount || hits[1].Phrase != "bank acc" {
		t.Errorf("second hit = %+v, want bank acc", hits[1])
	}
}

func TestLongestPhraseWins(t *testing.T) {
	hits := Find("Permanent Account Number: ABCPE1234F")
	if len(hits) != 1 || hits[0].Kind != KPAN {
		t.Fatalf("hits = %+v, want one PAN hit", hits)
	}
	hits = Find("Universal Account Number 100123456789")
	if len(hits) != 1 || hits[0].Kind != KUAN {
		t.Fatalf("hits = %+v, want one UAN hit", hits)
	}
}

func TestOCRNormalisation(t *testing.T) {
	hits := Find("Acc0unt N0: 900000000018")
	if len(hits) != 1 || hits[0].Kind != KBankAccount || !hits[0].OCR {
		t.Fatalf("hits = %+v, want OCR-normalised bank account label", hits)
	}
}

func TestIdentifierForms(t *testing.T) {
	for _, s := range []string{`{"aadhaarNumber": "900000000002"}`, "aadhaar_no=900000000002", "AADHAAR NO 900000000002", "e-Aadhaar: 9000 0000 0002"} {
		hits := Find(s)
		if len(hits) == 0 || hits[0].Kind != KAadhaar {
			t.Errorf("%q: hits = %+v, want aadhaar", s, kinds(hits))
		}
	}
}

func TestPhrasesUniqueAcrossKinds(t *testing.T) {
	owner := map[string]Kind{}
	for kind, phrases := range Phrases {
		for _, p := range phrases {
			if o, ok := owner[p]; ok && o != kind {
				t.Errorf("phrase %q listed under both %s and %s", p, o, kind)
			}
			owner[p] = kind
		}
	}
}
