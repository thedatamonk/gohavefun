// scoring/scorer.go
package scoring

import (
	"math"
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
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func Predict(customerID string, featureGroups map[string]store.FeatureVector) PredictionResult {
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
	}
}
