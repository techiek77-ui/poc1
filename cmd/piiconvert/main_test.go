package main

import "testing"

func TestRuneToByte(t *testing.T) {
	s := "Txn ₹ 999 to Priya"
	if got := runeToByte(s, 13); got != 15 || s[got:got+5] != "Priya" {
		t.Fatalf("runeToByte = %d (%q)", got, s[got:])
	}
	if runeToByte(s, 0) != 0 || runeToByte(s, 1000) != len(s) {
		t.Fatal("bounds")
	}
	// an invalid byte counts as one rune, not three
	if got := runeToByte("ab\xffcd Priya", 3); got != 3 {
		t.Fatalf("invalid byte advanced %d", got)
	}
}

func TestDecideRunes(t *testing.T) {
	// byte offsets on multi-byte text: span text proves it
	raws := []rawDoc{{text: "Txn ₹ 999 to Priya Nair", spans: []rawSpan{{start: 15, end: 25, label: "PERSON", value: "Priya Nair"}}}}
	if decideRunes(raws) {
		t.Fatal("byte offsets misread as runes")
	}
	raws = []rawDoc{{text: "Txn ₹ 999 to Priya Nair", spans: []rawSpan{{start: 13, end: 23, label: "PERSON", value: "Priya Nair"}}}}
	if !decideRunes(raws) {
		t.Fatal("rune offsets not recognised")
	}
}
