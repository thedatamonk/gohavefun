package scoring

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModelFromJSON(t *testing.T) {
	// Create a minimal XGBoost JSON model with 1 tree, 3 nodes
	// Tree: if feature[0] < 10.0 -> leaf -0.5, else -> leaf 0.3
	modelJSON := `{
		"learner": {
			"gradient_booster": {
				"model": {
					"trees": [
						{
							"id": 0,
							"tree_param": {"num_nodes": "3"},
							"left_children":  [1, -1, -1],
							"right_children": [2, -1, -1],
							"split_indices":  [0, 0, 0],
							"split_conditions": [10.0, 0.0, 0.0],
							"default_left":   [1, 0, 0],
							"base_weights":   [0.0, -0.5, 0.3]
						}
					]
				}
			},
			"feature_names": ["tenure_months"],
			"learner_model_param": {
				"base_score": "5.000000e-01",
				"num_feature": "1"
			}
		}
	}`

	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.json")
	os.WriteFile(modelPath, []byte(modelJSON), 0644)

	model, err := LoadXGBoostModel(modelPath)
	if err != nil {
		t.Fatalf("failed to load model: %v", err)
	}

	if len(model.Trees) != 1 {
		t.Fatalf("expected 1 tree, got %d", len(model.Trees))
	}
	if model.Trees[0].NumNodes != 3 {
		t.Fatalf("expected 3 nodes, got %d", model.Trees[0].NumNodes)
	}
}

func TestTreeTraversal(t *testing.T) {
	// Same tree: if feature[0] < 10.0 -> leaf -0.5, else -> leaf 0.3
	tree := Tree{
		NumNodes:        3,
		LeftChildren:    []int{1, -1, -1},
		RightChildren:   []int{2, -1, -1},
		SplitIndices:    []int{0, 0, 0},
		SplitConditions: []float64{10.0, 0.0, 0.0},
		DefaultLeft:     []int{1, 0, 0},
		BaseWeights:     []float64{0.0, -0.5, 0.3},
	}

	tests := []struct {
		name     string
		features []float64
		want     float64
	}{
		{"below threshold", []float64{5.0}, -0.5},
		{"above threshold", []float64{15.0}, 0.3},
		{"at threshold", []float64{10.0}, 0.3}, // not less than, goes right
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tree.Predict(tt.features)
			if got != tt.want {
				t.Errorf("Predict(%v) = %f, want %f", tt.features, got, tt.want)
			}
		})
	}
}

func TestModelPredict(t *testing.T) {
	// Two trees, both split on feature[0] < 10.0
	// Tree 1: left=-0.3, right=0.2
	// Tree 2: left=-0.1, right=0.4
	model := &XGBoostModel{
		BaseScore:    0.0,
		NumFeatures:  1,
		FeatureNames: []string{"tenure_months"},
		Trees: []Tree{
			{
				NumNodes: 3, LeftChildren: []int{1, -1, -1}, RightChildren: []int{2, -1, -1},
				SplitIndices: []int{0, 0, 0}, SplitConditions: []float64{10.0, 0.0, 0.0},
				DefaultLeft: []int{1, 0, 0}, BaseWeights: []float64{0.0, -0.3, 0.2},
			},
			{
				NumNodes: 3, LeftChildren: []int{1, -1, -1}, RightChildren: []int{2, -1, -1},
				SplitIndices: []int{0, 0, 0}, SplitConditions: []float64{10.0, 0.0, 0.0},
				DefaultLeft: []int{1, 0, 0}, BaseWeights: []float64{0.0, -0.1, 0.4},
			},
		},
	}

	features := map[string]float64{"tenure_months": 5.0}
	prob := model.PredictProba(features)
	// sigmoid(0.0 + (-0.3) + (-0.1)) = sigmoid(-0.4) ~ 0.401
	if prob < 0.39 || prob > 0.41 {
		t.Errorf("PredictProba with tenure=5 = %f, want ~0.401", prob)
	}

	features2 := map[string]float64{"tenure_months": 15.0}
	prob2 := model.PredictProba(features2)
	// sigmoid(0.0 + 0.2 + 0.4) = sigmoid(0.6) ~ 0.646
	if prob2 < 0.64 || prob2 > 0.66 {
		t.Errorf("PredictProba with tenure=15 = %f, want ~0.646", prob2)
	}
}
