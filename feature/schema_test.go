// feature/schema_test.go
package feature

import "testing"

func TestFeatureNames(t *testing.T) {
	if len(AllFeatureNames) != 13 {
		t.Fatalf("expected 13 features, got %d", len(AllFeatureNames))
	}
}

func TestWeights(t *testing.T) {
	for _, name := range AllFeatureNames {
		if _, ok := Weights[name]; !ok {
			t.Fatalf("missing weight for feature %q", name)
		}
	}
	if Bias == 0 {
		t.Fatal("bias should not be zero")
	}
}

func TestFeatureGroups(t *testing.T) {
	groups := []string{"customer_profile", "usage_metrics", "billing", "support"}
	for _, g := range groups {
		names, ok := FeatureGroups[g]
		if !ok {
			t.Fatalf("missing feature group %q", g)
		}
		if len(names) == 0 {
			t.Fatalf("feature group %q is empty", g)
		}
	}
}
