package verify

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/attrib"
	"github.com/techiek77-ui/poc1/pkg/pii/detect"
)

// Model is a logistic-regression verifier over sparse string features. It is
// a data artifact: the binary loads it at start-up (or on a reload signal),
// so a retrained model is shipped as a file, not as a release. The manifest
// fields let the loader refuse incompatible artifacts and let operators see
// what they are running.
type Model struct {
	Version          string                     `json:"version"`
	SchemaVersion    int                        `json:"schema_version"`
	TrainedAt        string                     `json:"trained_at"`
	TrainingData     string                     `json:"training_data"`
	Examples         int                        `json:"examples"`
	Bias             float32                    `json:"bias"`
	Weights          map[string]float32         `json:"weights"`
	Thresholds       map[pii.EntityType]float64 `json:"thresholds"`
	DefaultThreshold float64                    `json:"default_threshold"`
	Notes            map[string]any             `json:"notes,omitempty"`
}

// Score returns sigmoid(w.x + b).
func (m *Model) Score(feats []string) float64 {
	z := float64(m.Bias)
	for _, f := range feats {
		z += float64(m.Weights[f])
	}
	return 1 / (1 + math.Exp(-z))
}

// Threshold for an entity type.
func (m *Model) Threshold(t pii.EntityType) float64 {
	if v, ok := m.Thresholds[t]; ok {
		return v
	}
	return m.DefaultThreshold
}

// Explain lists the k largest absolute contributions, for the product log.
func (m *Model) Explain(feats []string, k int) string {
	type kv struct {
		f string
		w float32
	}
	var kvs []kv
	for _, f := range feats {
		if w, ok := m.Weights[f]; ok && w != 0 {
			kvs = append(kvs, kv{f, w})
		}
	}
	sort.Slice(kvs, func(i, j int) bool { return math.Abs(float64(kvs[i].w)) > math.Abs(float64(kvs[j].w)) })
	var parts []string
	for i := 0; i < len(kvs) && i < k; i++ {
		parts = append(parts, fmt.Sprintf("%s:%+.2f", kvs[i].f, kvs[i].w))
	}
	return strings.Join(parts, " ")
}

// Save writes the artifact as JSON.
func (m *Model) Save(path string) error {
	b, err := json.MarshalIndent(m, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Load reads an artifact and checks the feature schema.
func Load(path string) (*Model, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Model
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("model %s has feature schema %d, this build expects %d", path, m.SchemaVersion, SchemaVersion)
	}
	if m.DefaultThreshold == 0 {
		m.DefaultThreshold = 0.5
	}
	return &m, nil
}

// Example is one training row.
type Example struct {
	Feats  []string
	Label  bool
	Type   pii.EntityType
	Weight float32
}

// TrainOptions controls the optimiser.
type TrainOptions struct {
	Epochs   int
	LR       float64
	L2       float64
	MinCount int // drop features seen fewer times
	Seed     int64
	PosW     float32 // weight of positive examples
}

// DefaultTrainOptions are a reasonable starting point.
func DefaultTrainOptions() TrainOptions {
	return TrainOptions{Epochs: 8, LR: 0.1, L2: 1e-6, MinCount: 3, Seed: 1, PosW: 1}
}

// Train fits logistic regression with AdaGrad SGD.
func Train(ex []Example, o TrainOptions) *Model {
	counts := map[string]int{}
	for _, e := range ex {
		for _, f := range e.Feats {
			counts[f]++
		}
	}
	idx := map[string]int32{}
	var names []string
	for f, c := range counts {
		if c >= o.MinCount {
			idx[f] = int32(len(names))
			names = append(names, f)
		}
	}
	rows := make([][]int32, len(ex))
	for i, e := range ex {
		seen := map[int32]bool{}
		for _, f := range e.Feats {
			if j, ok := idx[f]; ok && !seen[j] {
				seen[j] = true
				rows[i] = append(rows[i], j)
			}
		}
	}
	w := make([]float64, len(names))
	g2 := make([]float64, len(names))
	bias, biasG2 := 0.0, 0.0
	r := rand.New(rand.NewSource(o.Seed))
	order := r.Perm(len(ex))
	for ep := 0; ep < o.Epochs; ep++ {
		r.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		loss := 0.0
		for _, i := range order {
			z := bias
			for _, j := range rows[i] {
				z += w[j]
			}
			p := 1 / (1 + math.Exp(-z))
			y := 0.0
			wt := 1.0
			if ex[i].Label {
				y = 1
				wt = float64(o.PosW)
			}
			if ex[i].Weight > 0 {
				wt *= float64(ex[i].Weight)
			}
			if ex[i].Label {
				loss -= wt * math.Log(p+1e-12)
			} else {
				loss -= wt * math.Log(1-p+1e-12)
			}
			grad := (p - y) * wt
			for _, j := range rows[i] {
				gj := grad + o.L2*w[j]
				g2[j] += gj * gj
				w[j] -= o.LR * gj / (math.Sqrt(g2[j]) + 1e-8)
			}
			biasG2 += grad * grad
			bias -= o.LR * grad / (math.Sqrt(biasG2) + 1e-8)
		}
		_ = loss
	}
	m := &Model{
		Version:          time.Now().UTC().Format("20060102T150405Z"),
		SchemaVersion:    SchemaVersion,
		TrainedAt:        time.Now().UTC().Format(time.RFC3339),
		Examples:         len(ex),
		Bias:             float32(bias),
		Weights:          make(map[string]float32, len(names)),
		Thresholds:       map[pii.EntityType]float64{},
		DefaultThreshold: 0.5,
	}
	for j, n := range names {
		if w[j] != 0 {
			m.Weights[n] = float32(w[j])
		}
	}
	return m
}

// Linear is the Verifier wrapping a Model.
type Linear struct {
	M    *Model
	Tag  string
	Topk int
}

func (l *Linear) Name() string {
	if l.Tag != "" {
		return l.Tag
	}
	return "linear"
}

func (l *Linear) Score(v *detect.DocView, c *pii.Candidate, a *attrib.Analysis) (float64, string) {
	feats := Features(c, a)
	s := l.M.Score(feats)
	k := l.Topk
	if k == 0 {
		k = 4
	}
	return s, l.M.Explain(feats, k)
}
