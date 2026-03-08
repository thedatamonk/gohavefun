package registry

import (
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	r, err := NewSQLiteRegistry(dbPath)
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func sampleView() FeatureViewDef {
	return FeatureViewDef{
		Name:        "customer_profile",
		EntityType:  "customer",
		Description: "Customer profile features",
		Owner:       "growth-team",
		Features: []FeatureDef{
			{Name: "tenure_months", Dtype: "float64", Description: "Months since signup"},
			{Name: "plan_tier", Dtype: "float64", Description: "1=basic, 2=pro, 3=enterprise"},
			{Name: "monthly_charge", Dtype: "float64", Description: "Monthly subscription price"},
		},
		Tags: map[string]string{"release": "v1"},
	}
}

func TestCreateAndGet(t *testing.T) {
	r := newTestRegistry(t)
	view := sampleView()

	if err := r.Create(view); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, ok := r.Get("customer_profile")
	if !ok {
		t.Fatal("expected view to exist")
	}
	if got.Name != "customer_profile" {
		t.Fatalf("expected name=customer_profile, got %s", got.Name)
	}
	if got.Owner != "growth-team" {
		t.Fatalf("expected owner=growth-team, got %s", got.Owner)
	}
	if len(got.Features) != 3 {
		t.Fatalf("expected 3 features, got %d", len(got.Features))
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("expected created_at to be set")
	}
}

func TestCreateDuplicate(t *testing.T) {
	r := newTestRegistry(t)
	view := sampleView()

	if err := r.Create(view); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(view); err == nil {
		t.Fatal("expected error on duplicate create")
	}
}

func TestGetMissing(t *testing.T) {
	r := newTestRegistry(t)
	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("expected view to be missing")
	}
}

func TestList(t *testing.T) {
	r := newTestRegistry(t)
	r.Create(sampleView())
	r.Create(FeatureViewDef{
		Name:       "usage_metrics",
		EntityType: "customer",
		Features:   []FeatureDef{{Name: "logins_last_30d", Dtype: "float64"}},
	})

	views := r.List()
	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}
}

func TestUpdate(t *testing.T) {
	r := newTestRegistry(t)
	r.Create(sampleView())

	updated := sampleView()
	updated.Owner = "platform-team"
	updated.Description = "Updated description"

	if err := r.Update("customer_profile", updated); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got, _ := r.Get("customer_profile")
	if got.Owner != "platform-team" {
		t.Fatalf("expected owner=platform-team, got %s", got.Owner)
	}
}

func TestUpdateMissing(t *testing.T) {
	r := newTestRegistry(t)
	if err := r.Update("nonexistent", sampleView()); err == nil {
		t.Fatal("expected error on update of missing view")
	}
}

func TestDelete(t *testing.T) {
	r := newTestRegistry(t)
	r.Create(sampleView())

	if err := r.Delete("customer_profile"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, ok := r.Get("customer_profile")
	if ok {
		t.Fatal("expected view to be deleted")
	}
}

func TestDeleteMissing(t *testing.T) {
	r := newTestRegistry(t)
	if err := r.Delete("nonexistent"); err == nil {
		t.Fatal("expected error on delete of missing view")
	}
}

func TestValidateFeatures(t *testing.T) {
	r := newTestRegistry(t)
	r.Create(sampleView())

	if err := r.ValidateFeatures("customer_profile", []string{"tenure_months", "plan_tier"}); err != nil {
		t.Fatalf("expected valid: %v", err)
	}

	if err := r.ValidateFeatures("customer_profile", []string{"tenure_months", "unknown_field"}); err == nil {
		t.Fatal("expected error for unknown feature")
	}

	if err := r.ValidateFeatures("nonexistent", []string{"tenure_months"}); err == nil {
		t.Fatal("expected error for unknown view")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	r1, _ := NewSQLiteRegistry(dbPath)
	r1.Create(sampleView())
	r1.Close()

	r2, _ := NewSQLiteRegistry(dbPath)
	defer r2.Close()

	got, ok := r2.Get("customer_profile")
	if !ok {
		t.Fatal("expected data to persist after reopen")
	}
	if got.Owner != "growth-team" {
		t.Fatalf("expected owner=growth-team, got %s", got.Owner)
	}
}
