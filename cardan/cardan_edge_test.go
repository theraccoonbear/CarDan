package cardan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func loadTestYAML(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read test YAML %q: %v", filename, err)
	}
	return string(data)
}

type parseYAMLTestCase struct {
	name          string
	yamlInput     string
	expectAnchors []string
	expectError   bool
}

type resolveAliasTestCase struct {
	name        string
	yamlFile    string
	aliasTarget string
	expectError bool
	errorSubstr string
}

func TestParseYAMLAndAnchorIndexing(t *testing.T) {
	tests := []parseYAMLTestCase{
		{
			name: "anchors are indexed",
			yamlInput: `
first: &first
  name: foo
second: &second
  name: bar
`,
			expectAnchors: []string{"first", "second"},
			expectError:   false,
		},
		{
			name: "no anchors",
			yamlInput: `
first:
  name: foo
second:
  name: bar
`,
			expectAnchors: []string{},
			expectError:   false,
		},
		{
			name: "invalid YAML structure",
			yamlInput: `
first
  name: foo
`,
			expectAnchors: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseYAML(strings.NewReader(tt.yamlInput))

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(doc.NodesByID) != len(tt.expectAnchors) {
				t.Errorf("expected %d anchors, got %d", len(tt.expectAnchors), len(doc.NodesByID))
			}

			for _, anchor := range tt.expectAnchors {
				if _, ok := doc.NodesByID[anchor]; !ok {
					t.Errorf("expected anchor %q not found", anchor)
				}
			}
		})
	}
}

func TestResolveAlias(t *testing.T) {
	tests := []resolveAliasTestCase{
		{
			name:        "valid alias resolution",
			yamlFile:    "resolve_valid.yml",
			aliasTarget: "first",
			expectError: false,
		},
		{
			name:        "not an alias node",
			yamlFile:    "resolve_not_alias.yml",
			aliasTarget: "",
			expectError: true,
			errorSubstr: "node is not alias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseYAML(strings.NewReader(loadTestYAML(t, tt.yamlFile)))
			if err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			// Find first interesting node
			var targetNode *yaml.Node
			var findNode func(nodes []*yaml.Node)
			findNode = func(nodes []*yaml.Node) {
				for _, n := range nodes {
					// Search for first alias OR first mapping value, depending on test
					if (tt.expectError && tt.errorSubstr == "node is not alias" && n.Kind == yaml.MappingNode) ||
						(!tt.expectError && n.Kind == yaml.AliasNode) ||
						(tt.expectError && tt.errorSubstr == "unresolved alias" && n.Kind == yaml.AliasNode) {
						targetNode = n
						return
					}
					findNode(n.Content)
				}
			}
			findNode(doc.RawTree.Content)

			if targetNode == nil {
				t.Fatal("could not find target node for test")
			}

			resolved, err := doc.ResolveAlias(targetNode)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorSubstr)
				}
				if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.AnchorID != tt.aliasTarget {
				t.Errorf("expected alias to resolve to anchor %q, got %q", tt.aliasTarget, resolved.AnchorID)
			}
		})
	}
}
