// Package checksum implements the check-digit algorithms used by Indian
// identifiers: Verhoeff (Aadhaar, VID), Luhn (payment cards, IMEI) and the
// base-36 weighted check character of GSTIN.
package checksum

import "strings"

// Verhoeff multiplication, permutation and inverse tables.
var (
	verhoeffD = [10][10]int{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
		{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
		{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
		{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
		{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
		{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
		{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
		{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
		{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
	}
	verhoeffP = [8][10]int{
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
		{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
		{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
		{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
		{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
		{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
		{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
	}
	verhoeffInv = [10]int{0, 4, 3, 2, 1, 5, 6, 7, 8, 9}
)

// Verhoeff reports whether digits (ASCII digits only, check digit last) pass
// the Verhoeff check.
func Verhoeff(digits string) bool {
	c := 0
	n := len(digits)
	if n == 0 {
		return false
	}
	for i := 0; i < n; i++ {
		ch := digits[n-1-i]
		if ch < '0' || ch > '9' {
			return false
		}
		c = verhoeffD[c][verhoeffP[i%8][int(ch-'0')]]
	}
	return c == 0
}

// VerhoeffCheckDigit returns the check digit to append to digits.
func VerhoeffCheckDigit(digits string) byte {
	c := 0
	n := len(digits)
	for i := 0; i < n; i++ {
		ch := digits[n-1-i]
		c = verhoeffD[c][verhoeffP[(i+1)%8][int(ch-'0')]]
	}
	return byte('0' + verhoeffInv[c])
}

// Luhn reports whether digits pass the Luhn (mod 10) check.
func Luhn(digits string) bool {
	n := len(digits)
	if n < 2 {
		return false
	}
	sum := 0
	double := false
	for i := n - 1; i >= 0; i-- {
		ch := digits[i]
		if ch < '0' || ch > '9' {
			return false
		}
		d := int(ch - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// LuhnCheckDigit returns the check digit to append to digits.
func LuhnCheckDigit(digits string) byte {
	// Append a 0 and compute what would make it valid.
	n := len(digits)
	sum := 0
	double := true
	for i := n - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return byte('0' + (10-sum%10)%10)
}

const base36 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// GSTINCheckChar computes the 15th character of a GSTIN from its first 14.
// Returns 0 if the input contains a character outside [0-9A-Z].
func GSTINCheckChar(first14 string) byte {
	if len(first14) != 14 {
		return 0
	}
	sum := 0
	for i := 0; i < 14; i++ {
		v := strings.IndexByte(base36, first14[i])
		if v < 0 {
			return 0
		}
		f := 1
		if i%2 == 1 {
			f = 2
		}
		p := v * f
		sum += p/36 + p%36
	}
	return base36[(36-sum%36)%36]
}

// GSTIN reports whether a 15-character GSTIN has a valid check character.
func GSTIN(s string) bool {
	if len(s) != 15 {
		return false
	}
	c := GSTINCheckChar(s[:14])
	return c != 0 && c == s[14]
}

// Digits strips everything except ASCII digits.
func Digits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
