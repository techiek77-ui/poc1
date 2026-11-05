//go:build onnx

package onnx

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestParityWithPython loads the smoke model produced by ml/make_test_onnx.py
// and checks that Go tokenization and scoring match the Python reference.
// Needs PII_ORT_LIB=/path/to/libonnxruntime.so and the model directory
// (PII_ONNX_DIR, default ../../../models/tier_b_smoke).
func TestParityWithPython(t *testing.T) {
	lib := os.Getenv("PII_ORT_LIB")
	if lib == "" {
		t.Skip("PII_ORT_LIB not set")
	}
	dir := os.Getenv("PII_ONNX_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", "models", "tier_b_smoke")
	}
	if _, err := os.Stat(filepath.Join(dir, "reference.json")); err != nil {
		t.Skipf("no smoke model at %s: %v", dir, err)
	}
	if err := Init(lib); err != nil {
		t.Fatal(err)
	}
	v, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	var refs []struct {
		Text string  `json:"text"`
		IDs  []int64 `json:"ids"`
		P    float64 `json:"p"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &refs); err != nil {
		t.Fatal(err)
	}
	for i, r := range refs {
		ids, _, _, err := v.Encode(r.Text)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != len(r.IDs) {
			t.Fatalf("ref %d: token count %d, python %d\n%q\ngo=%v\npy=%v", i, len(ids), len(r.IDs), r.Text, ids, r.IDs)
		}
		for j := range ids {
			if ids[j] != r.IDs[j] {
				t.Fatalf("ref %d: token %d differs: go %d python %d (%q)", i, j, ids[j], r.IDs[j], r.Text)
			}
		}
		p, err := v.ScoreText(r.Text)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(p-r.P) > 1e-4 {
			t.Fatalf("ref %d: p=%.6f python %.6f", i, p, r.P)
		}
	}
	t.Logf("%d references match (tokens and probabilities)", len(refs))
}
