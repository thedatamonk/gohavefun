package scoring

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// Tree represents a single XGBoost decision tree using flat arrays.
type Tree struct {
	NumNodes        int
	LeftChildren    []int
	RightChildren   []int
	SplitIndices    []int
	SplitConditions []float64
	DefaultLeft     []int
	BaseWeights     []float64
}

// Predict traverses the tree and returns the leaf value.
func (t *Tree) Predict(features []float64) float64 {
	node := 0
	for {
		if t.LeftChildren[node] == -1 {
			return t.BaseWeights[node]
		}

		splitIdx := t.SplitIndices[node]
		threshold := t.SplitConditions[node]

		var val float64
		if splitIdx < len(features) {
			val = features[splitIdx]
		} else if t.DefaultLeft[node] == 1 {
			node = t.LeftChildren[node]
			continue
		} else {
			node = t.RightChildren[node]
			continue
		}

		if val < threshold {
			node = t.LeftChildren[node]
		} else {
			node = t.RightChildren[node]
		}
	}
}

// XGBoostModel holds the full parsed XGBoost model.
type XGBoostModel struct {
	BaseScore    float64
	NumFeatures  int
	FeatureNames []string
	Trees        []Tree
}

// PredictProba returns churn probability using sigmoid over summed tree outputs.
func (m *XGBoostModel) PredictProba(featureMap map[string]float64) float64 {
	// Build feature vector in order
	features := make([]float64, m.NumFeatures)
	for i, name := range m.FeatureNames {
		features[i] = featureMap[name]
	}

	sum := m.BaseScore
	for i := range m.Trees {
		sum += m.Trees[i].Predict(features)
	}

	return 1.0 / (1.0 + math.Exp(-sum))
}

// xgbJSON mirrors the XGBoost save_model JSON structure (only fields we need).
type xgbJSON struct {
	Learner struct {
		GradientBooster struct {
			Model struct {
				Trees []xgbTreeJSON `json:"trees"`
			} `json:"model"`
		} `json:"gradient_booster"`
		FeatureNames      []string `json:"feature_names"`
		LearnerModelParam struct {
			BaseScore  string `json:"base_score"`
			NumFeature string `json:"num_feature"`
		} `json:"learner_model_param"`
	} `json:"learner"`
}

type xgbTreeJSON struct {
	TreeParam struct {
		NumNodes string `json:"num_nodes"`
	} `json:"tree_param"`
	LeftChildren    []int     `json:"left_children"`
	RightChildren   []int     `json:"right_children"`
	SplitIndices    []int     `json:"split_indices"`
	SplitConditions []float64 `json:"split_conditions"`
	DefaultLeft     []int     `json:"default_left"`
	BaseWeights     []float64 `json:"base_weights"`
}

// LoadXGBoostModel parses an XGBoost JSON model file.
func LoadXGBoostModel(path string) (*XGBoostModel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read model file: %w", err)
	}

	var raw xgbJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse model JSON: %w", err)
	}

	numFeatures, _ := strconv.Atoi(raw.Learner.LearnerModelParam.NumFeature)
	baseScore, _ := strconv.ParseFloat(raw.Learner.LearnerModelParam.BaseScore, 64)

	trees := make([]Tree, len(raw.Learner.GradientBooster.Model.Trees))
	for i, t := range raw.Learner.GradientBooster.Model.Trees {
		numNodes, _ := strconv.Atoi(t.TreeParam.NumNodes)
		trees[i] = Tree{
			NumNodes:        numNodes,
			LeftChildren:    t.LeftChildren,
			RightChildren:   t.RightChildren,
			SplitIndices:    t.SplitIndices,
			SplitConditions: t.SplitConditions,
			DefaultLeft:     t.DefaultLeft,
			BaseWeights:     t.BaseWeights,
		}
	}

	featureNames := raw.Learner.FeatureNames

	// If feature_names not in model JSON, try meta.json in same directory
	if len(featureNames) == 0 {
		metaPath := filepath.Join(filepath.Dir(path), "meta.json")
		if metaData, err := os.ReadFile(metaPath); err == nil {
			var meta struct {
				FeatureNames []string `json:"feature_names"`
			}
			if json.Unmarshal(metaData, &meta) == nil {
				featureNames = meta.FeatureNames
			}
		}
	}

	return &XGBoostModel{
		BaseScore:    baseScore,
		NumFeatures:  numFeatures,
		FeatureNames: featureNames,
		Trees:        trees,
	}, nil
}
