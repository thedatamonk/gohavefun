// scoring/scorer.go
package scoring

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"github.com/rohil/gofun/feature"
	"github.com/rohil/gofun/store"
)

type RiskFactor struct {
	Feature      string  `json:"feature"`
	Value        float64 `json:"value"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

type PredictionResult struct {
	CustomerID       string       `json:"customer_id"`
	ChurnProbability float64      `json:"churn_probability"`
	RiskLevel        string       `json:"risk_level"`
	TopRiskFactors   []RiskFactor `json:"top_risk_factors"`
	ModelType        string       `json:"model_type"`
}

type Scorer struct {
	model *XGBoostModel
}

// NewScorer tries to load an XGBoost model from modelDir/model.json.
// Falls back to logistic regression if loading fails.
func NewScorer(modelDir string) *Scorer {
	modelPath := filepath.Join(modelDir, "model.json")
	model, err := LoadXGBoostModel(modelPath)
	if err != nil {
		fmt.Printf("No XGBoost model at %s, using logistic regression fallback\n", modelPath)
		return &Scorer{}
	}
	fmt.Printf("Loaded XGBoost model: %d trees, %d features\n", len(model.Trees), model.NumFeatures)
	return &Scorer{model: model}
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (s *Scorer) Predict(customerID string, featureGroups map[string]store.FeatureVector) PredictionResult {
	if s.model != nil {
		return s.predictXGBoost(customerID, featureGroups)
	}
	return s.predictFallback(customerID, featureGroups)
}

func (s *Scorer) predictXGBoost(customerID string, featureGroups map[string]store.FeatureVector) PredictionResult {
	allFeatures := make(map[string]float64)
	for _, fv := range featureGroups {
		for k, v := range fv {
			allFeatures[k] = v
		}
	}

	prob := s.model.PredictProba(allFeatures)
	prob = math.Round(prob*1000) / 1000

	riskLevel := "low"
	if prob >= 0.7 {
		riskLevel = "high"
	} else if prob >= 0.3 {
		riskLevel = "medium"
	}

	// For XGBoost, report features with highest absolute values as risk factors.
	// NOTE: risk factors currently use logistic regression weights, not XGBoost
	// feature importances. This is a known limitation; a future improvement would
	// derive contributions from the XGBoost model (e.g., SHAP values).
	var factors []RiskFactor
	for _, name := range feature.AllFeatureNames {
		val := allFeatures[name]
		w := feature.Weights[name]
		factors = append(factors, RiskFactor{
			Feature:      name,
			Value:        val,
			Weight:       w,
			Contribution: w * val,
		})
	}
	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Contribution > factors[j].Contribution
	})
	var topFactors []RiskFactor
	for _, f := range factors {
		if f.Contribution > 0 && len(topFactors) < 3 {
			topFactors = append(topFactors, f)
		}
	}

	return PredictionResult{
		CustomerID:       customerID,
		ChurnProbability: prob,
		RiskLevel:        riskLevel,
		TopRiskFactors:   topFactors,
		ModelType:        "xgboost",
	}
}

func (s *Scorer) predictFallback(customerID string, featureGroups map[string]store.FeatureVector) PredictionResult {
	allFeatures := make(map[string]float64)
	for _, fv := range featureGroups {
		for k, v := range fv {
			allFeatures[k] = v
		}
	}

	sum := feature.Bias
	var factors []RiskFactor
	for _, name := range feature.AllFeatureNames {
		val := allFeatures[name]
		w := feature.Weights[name]
		contribution := w * val
		sum += contribution
		factors = append(factors, RiskFactor{
			Feature:      name,
			Value:        val,
			Weight:       w,
			Contribution: contribution,
		})
	}

	prob := sigmoid(sum)

	riskLevel := "low"
	if prob >= 0.7 {
		riskLevel = "high"
	} else if prob >= 0.3 {
		riskLevel = "medium"
	}

	sort.Slice(factors, func(i, j int) bool {
		return factors[i].Contribution > factors[j].Contribution
	})
	var topFactors []RiskFactor
	for _, f := range factors {
		if f.Contribution > 0 && len(topFactors) < 3 {
			topFactors = append(topFactors, f)
		}
	}

	return PredictionResult{
		CustomerID:       customerID,
		ChurnProbability: math.Round(prob*1000) / 1000,
		RiskLevel:        riskLevel,
		TopRiskFactors:   topFactors,
		ModelType:        "logistic_regression",
	}
}

// Predict is the package-level function for backward compatibility.
func Predict(customerID string, featureGroups map[string]store.FeatureVector) PredictionResult {
	fallback := &Scorer{}
	return fallback.Predict(customerID, featureGroups)
}
