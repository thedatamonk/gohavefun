package registry

import (
	"fmt"
	"time"
)

type FeatureDef struct {
	Name        string `json:"name"`
	Dtype       string `json:"dtype"`
	Description string `json:"description,omitempty"`
}

type FeatureViewDef struct {
	Name        string            `json:"name"`
	EntityType  string            `json:"entity_type"`
	Description string            `json:"description,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Features    []FeatureDef      `json:"features"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type Registry struct {
	store registryStore
}

type registryStore interface {
	create(view FeatureViewDef) error
	get(name string) (FeatureViewDef, bool)
	list() []FeatureViewDef
	update(name string, view FeatureViewDef) error
	delete(name string) error
	close() error
}

func (r *Registry) Create(view FeatureViewDef) error {
	return r.store.create(view)
}

func (r *Registry) Get(name string) (FeatureViewDef, bool) {
	return r.store.get(name)
}

func (r *Registry) List() []FeatureViewDef {
	return r.store.list()
}

func (r *Registry) Update(name string, view FeatureViewDef) error {
	return r.store.update(name, view)
}

func (r *Registry) Delete(name string) error {
	return r.store.delete(name)
}

func (r *Registry) Close() error {
	return r.store.close()
}

func (r *Registry) ValidateFeatures(viewName string, featureNames []string) error {
	view, ok := r.store.get(viewName)
	if !ok {
		return fmt.Errorf("unknown feature view %q", viewName)
	}

	allowed := make(map[string]bool, len(view.Features))
	for _, f := range view.Features {
		allowed[f.Name] = true
	}

	for _, name := range featureNames {
		if !allowed[name] {
			return fmt.Errorf("unknown feature %q in view %q", name, viewName)
		}
	}
	return nil
}
