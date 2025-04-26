package cardan

import (
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
