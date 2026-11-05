package checksum

import (
	"math/rand"
	"testing"
)

func TestVerhoeffKnownVectors(t *testing.T) {
	// Wikipedia example: payload 236 has check digit 3.
	if got := VerhoeffCheckDigit("236"); got != '3' {
		t.Fatalf("check digit of 236 = %c, want 3", got)
	}
	if !Verhoeff("2363") {
		t.Fatal("2363 should be valid")
	}
	if Verhoeff("2364") {
		t.Fatal("2364 should be invalid")
	}
	if Verhoeff("") || Verhoeff("12a4") {
		t.Fatal("empty / non-digit input must be invalid")
	}
}

func TestVerhoeffDetectsSingleErrorsAndTranspositions(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for n := 0; n < 2000; n++ {
		payload := make([]byte, 11)
		for i := range payload {
			payload[i] = byte('0' + r.Intn(10))
		}
		s := string(payload) + string(VerhoeffCheckDigit(string(payload)))
		if !Verhoeff(s) {
			t.Fatalf("generated %s should validate", s)
		}
		// single-digit substitution
		b := []byte(s)
		i := r.Intn(len(b))
		b[i] = byte('0' + (int(b[i]-'0')+1+r.Intn(9))%10)
		if Verhoeff(string(b)) {
			t.Fatalf("single substitution %s -> %s not detected", s, b)
		}
		// adjacent transposition
		b = []byte(s)
		i = r.Intn(len(b) - 1)
		if b[i] != b[i+1] {
			b[i], b[i+1] = b[i+1], b[i]
			if Verhoeff(string(b)) {
				t.Fatalf("adjacent transposition %s -> %s not detected", s, b)
			}
		}
	}
}

func TestLuhn(t *testing.T) {
	for _, ok := range []string{"4111111111111111", "79927398713", "5555555555554444", "378282246310005", "6011111111111117"} {
		if !Luhn(ok) {
			t.Errorf("%s should be Luhn-valid", ok)
		}
	}
	for _, bad := range []string{"4111111111111112", "79927398710", "1", ""} {
		if Luhn(bad) {
			t.Errorf("%s should be Luhn-invalid", bad)
		}
	}
	if LuhnCheckDigit("7992739871") != '3' {
		t.Errorf("Luhn check digit of 7992739871 should be 3")
	}
	if LuhnCheckDigit("411111111111111") != '1' {
		t.Errorf("Luhn check digit of 411111111111111 should be 1")
	}
}

func TestGSTIN(t *testing.T) {
	// Published valid GSTINs; both also validate with python-stdnum.
	for _, ok := range []string{"27AAPFU0939F1ZV", "07AAGFF2194N1Z1"} {
		if !GSTIN(ok) {
			t.Errorf("%s should have a valid check character (computed %c)", ok, GSTINCheckChar(ok[:14]))
		}
	}
	if GSTIN("27AAPFU0939F1ZW") {
		t.Error("altered check character should fail")
	}
	if GSTIN("27AAPFU0939F1Z") {
		t.Error("wrong length should fail")
	}
}

func TestDigits(t *testing.T) {
	if Digits("9000 0000-0002x") != "900000000002" {
		t.Fatal("Digits should strip non-digits")
	}
}
