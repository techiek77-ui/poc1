package main

import "testing"

func TestRuneToByte(t *testing.T) {
	s := "Txn ₹ 999 to Priya"
	// rune index of "Priya" is 13; byte offset is 15 because ₹ is 3 bytes
	if got := runeToByte(s, 13); got != 15 || s[got:got+5] != "Priya" {
		t.Fatalf("runeToByte = %d (%q)", got, s[got:])
	}
	if runeToByte(s, 0) != 0 || runeToByte(s, 1000) != len(s) {
		t.Fatal("bounds")
	}
}
