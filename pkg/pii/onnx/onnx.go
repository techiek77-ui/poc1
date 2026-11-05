//go:build onnx

// Package onnx runs a transformer verifier exported to ONNX from Go through
// ONNX Runtime (shipped as libonnxruntime.so / onnxruntime.dll) and a pure-Go
// HuggingFace-compatible tokenizer. Build with `-tags onnx`.
//
// Files expected in the model directory (written by ml/train_tier_b.py):
//
//	model.onnx or model.int8.onnx   the classifier (inputs: input_ids, attention_mask[, token_type_ids]; output: logits [1,2])
//	tokenizer.json                  HuggingFace tokenizer
//	manifest.json                   max_len, thresholds, training provenance
package onnx

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
	"github.com/techiek77-ui/poc1/pkg/pii/onnxinput"
)

// Manifest is written next to the model by the training script.
type Manifest struct {
	Version          string                     `json:"version"`
	Base             string                     `json:"base"`
	MaxLen           int                        `json:"max_len"`
	Inputs           []string                   `json:"inputs"`
	Thresholds       map[pii.EntityType]float64 `json:"thresholds"`
	DefaultThreshold float64                    `json:"default_threshold"`
	ModelFile        string                     `json:"model_file"`
}

// Verifier scores candidates with the ONNX model.
type Verifier struct {
	M       Manifest
	tk      *tokenizer.Tokenizer
	sess    *ort.DynamicAdvancedSession
	mu      sync.Mutex
	inputs  []string
	hasType bool
}

var initOnce sync.Once
var initErr error

// Init points ONNX Runtime at the shared library and initialises it once.
func Init(ortLib string) error {
	initOnce.Do(func() {
		if ortLib != "" {
			ort.SetSharedLibraryPath(ortLib)
		}
		initErr = ort.InitializeEnvironment()
	})
	return initErr
}

// New loads a model directory.
func New(dir string) (*Verifier, error) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.MaxLen == 0 {
		m.MaxLen = 128
	}
	if m.DefaultThreshold == 0 {
		m.DefaultThreshold = 0.5
	}
	if m.ModelFile == "" {
		m.ModelFile = "model.onnx"
	}
	tk, err := pretrained.FromFile(filepath.Join(dir, "tokenizer.json"))
	if err != nil {
		return nil, fmt.Errorf("tokenizer: %w", err)
	}
	inputs := m.Inputs
	if len(inputs) == 0 {
		inputs = []string{"input_ids", "attention_mask"}
	}
	sess, err := ort.NewDynamicAdvancedSession(filepath.Join(dir, m.ModelFile), inputs, []string{"logits"}, nil)
	if err != nil {
		return nil, fmt.Errorf("onnx session: %w", err)
	}
	v := &Verifier{M: m, tk: tk, sess: sess, inputs: inputs}
	for _, n := range inputs {
		if n == "token_type_ids" {
			v.hasType = true
		}
	}
	return v, nil
}

// Close releases the session.
func (v *Verifier) Close() {
	if v.sess != nil {
		v.sess.Destroy()
	}
}

func (v *Verifier) Name() string { return "B onnx" }

// Threshold for an entity type.
func (v *Verifier) Threshold(t pii.EntityType) float64 {
	if x, ok := v.M.Thresholds[t]; ok {
		return x
	}
	return v.M.DefaultThreshold
}

// Encode tokenizes the model input for a candidate (exposed for tests).
func (v *Verifier) Encode(text string) ([]int64, []int64, []int64, error) {
	en, err := v.tk.EncodeSingle(text, true)
	if err != nil {
		return nil, nil, nil, err
	}
	n := len(en.Ids)
	if n > v.M.MaxLen {
		n = v.M.MaxLen
	}
	ids := make([]int64, n)
	mask := make([]int64, n)
	types := make([]int64, n)
	for i := 0; i < n; i++ {
		ids[i] = int64(en.Ids[i])
		mask[i] = 1
		if i < len(en.TypeIds) {
			types[i] = int64(en.TypeIds[i])
		}
	}
	return ids, mask, types, nil
}

// Score runs the model. Returns P(candidate is the claimed type).
func (v *Verifier) Score(_ *detect.DocView, c *pii.Candidate, a *attrib.Analysis) (float64, string) {
	text := onnxinput.Build(c, a)
	p, err := v.ScoreText(text)
	if err != nil {
		return 0.5, "onnx error: " + err.Error()
	}
	return p, fmt.Sprintf("onnx p=%.3f", p)
}

// ScoreText scores a prepared input string.
func (v *Verifier) ScoreText(text string) (float64, error) {
	ids, mask, types, err := v.Encode(text)
	if err != nil {
		return 0, err
	}
	shape := ort.NewShape(1, int64(len(ids)))
	tIds, err := ort.NewTensor(shape, ids)
	if err != nil {
		return 0, err
	}
	defer tIds.Destroy()
	tMask, err := ort.NewTensor(shape, mask)
	if err != nil {
		return 0, err
	}
	defer tMask.Destroy()
	in := []ort.Value{tIds, tMask}
	if v.hasType {
		tTypes, err := ort.NewTensor(shape, types)
		if err != nil {
			return 0, err
		}
		defer tTypes.Destroy()
		in = append(in, tTypes)
	}
	out := []ort.Value{nil}
	v.mu.Lock()
	err = v.sess.Run(in, out)
	v.mu.Unlock()
	if err != nil {
		return 0, err
	}
	defer out[0].Destroy()
	logits, ok := out[0].(*ort.Tensor[float32])
	if !ok {
		return 0, fmt.Errorf("unexpected output type %T", out[0])
	}
	d := logits.GetData()
	switch len(d) {
	case 1:
		return 1 / (1 + math.Exp(-float64(d[0]))), nil
	case 2:
		z := float64(d[1] - d[0])
		return 1 / (1 + math.Exp(-z)), nil
	}
	return 0, fmt.Errorf("unexpected logits length %d", len(d))
}
