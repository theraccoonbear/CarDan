package cardan

import (
	"strings"
	"testing"
)

type Job struct {
	Name      string   `yaml:"name"`
	DependsOn []string `yaml:"depends_on"`
}

func TestCarDanLoadAndResolve(t *testing.T) {
	content := loadTestYAML(t, "load_and_resolve.yml")
	r := strings.NewReader(content)

	cd, err := Load(r)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cd.GetRawAnchor("jobA") == nil {
		t.Error("expected anchor 'jobA' to be present")
	}

	if err := cd.ResolveRefs("depends_on"); err != nil {
		t.Errorf("ResolveRefs failed: %v", err)
	}

	var m map[string]Job
	if err := cd.Unmarshal(&m); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if m["jobB"].DependsOn[0] != "jobA" {
		t.Errorf("expected alias to resolve to 'jobA', got: %v", m["jobB"].DependsOn)
	}
}
