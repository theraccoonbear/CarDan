package cardan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseYAMLAndResolve(t *testing.T) {
	doc := parseTestDoc(t, "parse_and_resolve.yml")

	node, ok := doc.NodesByID["default"]
	if !ok {
		t.Fatalf("expected anchor 'default' not found")
	}
	if node.AST.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping node for anchor 'default', got %v", node.AST.Kind)
	}

	job1 := findMapEntry(doc.RawTree, "job1")
	if job1 == nil {
		t.Fatal("expected 'job1' node not found")
	}

	merged := findMapEntry(job1, "<<")
	if merged == nil {
		t.Fatal("expected '<<' merge key not found in job1")
	}

	resolved, err := doc.ResolveAlias(merged)
	if err != nil {
		t.Fatalf("failed to resolve alias: %v", err)
	}
	if resolved.AST.Kind != yaml.MappingNode {
		t.Fatalf("expected resolved alias to be a mapping, got %v", resolved.AST.Kind)
	}
}

func TestLoadWithIncludes_OK(t *testing.T) {
	base := "testdata/include_ok"
	mainPath := filepath.Join(base, "main.yml")
	r, err := os.Open(mainPath)
	if err != nil {
		t.Fatalf("failed to open main.yml: %v", err)
	}
	defer r.Close()

	doc, err := LoadWithOptions(r, LoadOptions{
		IncludeTag: "!include",
		BasePath:   base,
	})
	if err != nil {
		t.Fatalf("unexpected error loading with includes: %v", err)
	}

	tasks := findMapEntry(doc.RawTree, "tasks")
	if tasks == nil {
		t.Fatal("expected 'tasks' key not found in YAML")
	}
	if tasks.Kind != yaml.SequenceNode || len(tasks.Content) == 0 {
		t.Fatal("expected 'tasks' to be a non-empty sequence")
	}
	firstTask := tasks.Content[0]
	task1 := findMapEntry(firstTask, "task1")
	if task1 == nil {
		t.Fatal("expected 'task1' inside first included task node")
	}
}

func TestLoadWithIncludes_DenyTraversal(t *testing.T) {
	base := "testdata/include_traversal"
	mainPath := filepath.Join(base, "main.yml")
	r, err := os.Open(mainPath)
	if err != nil {
		t.Fatalf("failed to open main.yml: %v", err)
	}
	defer r.Close()

	_, err = LoadWithOptions(r, LoadOptions{
		IncludeTag: "!include",
		BasePath:   base,
	})
	if err == nil || !strings.Contains(err.Error(), "parent traversal forbidden") {
		t.Fatalf("expected traversal forbidden error, got: %v", err)
	}
}

func TestLoadWithIncludes_EscapeBaseDir(t *testing.T) {
	base := "testdata/include_escape"
	mainPath := filepath.Join(base, "main.yml")
	r, err := os.Open(mainPath)
	if err != nil {
		t.Fatalf("failed to open main.yml: %v", err)
	}
	defer r.Close()

	_, err = LoadWithOptions(r, LoadOptions{
		IncludeTag: "!include",
		BasePath:   base,
	})
	if err == nil || (!strings.Contains(err.Error(), "escapes base directory") && !strings.Contains(err.Error(), "failed to read included file")) {
		t.Fatalf("expected escape base directory or read error, got: %v", err)
	}
}

func TestLoadWithIncludes_RecursiveIncludes(t *testing.T) {
	base := "testdata/include_recursive"
	mainPath := filepath.Join(base, "main.yml")
	r, err := os.Open(mainPath)
	if err != nil {
		t.Fatalf("failed to open main.yml: %v", err)
	}
	defer r.Close()

	_, err = LoadWithOptions(r, LoadOptions{
		IncludeTag: "!include",
		BasePath:   base,
	})
	if err == nil || !strings.Contains(err.Error(), "recursive inclusion detected") {
		t.Fatalf("expected recursive inclusion error, got: %v", err)
	}
}

func TestLoadWithIncludes_DenySyntacticTraversal(t *testing.T) {
	base := "testdata/include_upward_synthetic"
	mainPath := filepath.Join(base, "main.yml")
	r, err := os.Open(mainPath)
	if err != nil {
		t.Fatalf("failed to open main.yml: %v", err)
	}
	defer r.Close()

	_, err = LoadWithOptions(r, LoadOptions{
		IncludeTag: "!include",
		BasePath:   base,
	})
	if err == nil || !strings.Contains(err.Error(), "parent traversal forbidden") {
		t.Fatalf("expected parent traversal forbidden on syntactic upward, got: %v", err)
	}
}

func TestLoadWithIncludes_AnchorIndexed(t *testing.T) {
	base := "testdata/include_anchor"
	mainPath := filepath.Join(base, "main.yml")
	r, err := os.Open(mainPath)
	if err != nil {
		t.Fatalf("failed to open main.yml: %v", err)
	}
	defer r.Close()

	doc, err := LoadWithOptions(r, LoadOptions{
		IncludeTag: "!include",
		BasePath:   base,
	})
	if err != nil {
		t.Fatalf("unexpected error loading with includes: %v", err)
	}

	if _, ok := doc.NodesByID["inc_anchor"]; !ok {
		t.Fatalf("expected anchor from included file to be indexed")
	}
}

// === HELPERS ===

func parseTestDoc(t *testing.T, filename string) *Doc {
	t.Helper()
	content := loadTestYAML(t, filename)
	doc, err := ParseYAML(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return doc
}

func findMapEntry(root *yaml.Node, key string) *yaml.Node {
	if root.Kind != yaml.DocumentNode && root.Kind != yaml.MappingNode {
		return nil
	}
	var nodes []*yaml.Node
	if root.Kind == yaml.DocumentNode {
		nodes = root.Content
	} else {
		nodes = []*yaml.Node{root}
	}
	for _, n := range nodes {
		if n.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(n.Content)-1; i += 2 {
			k := n.Content[i]
			v := n.Content[i+1]
			if k.Value == key {
				return v
			}
		}
	}
	return nil
}
