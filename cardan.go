package cardan

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Node wraps a YAML node with ID and metadata for reference tracking
type Node struct {
	RefID    string
	AST      *yaml.Node
	AnchorID string
	Line     int
	Column   int
}

// Doc represents a fully parsed and indexed YAML document
type Doc struct {
	NodesByID map[string]*Node
	RawTree   *yaml.Node
}

type LoadOptions struct {
	IncludeTag string // default: \"!include\"
	BasePath   string // must be provided if IncludeTag is set
}

// ParseYAML reads YAML from an io.Reader and indexes its anchors
func ParseYAML(r io.Reader) (*Doc, error) {
	decoder := yaml.NewDecoder(r)

	var root yaml.Node
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	doc := &Doc{
		NodesByID: make(map[string]*Node),
		RawTree:   &root,
	}

	if err := indexNodes(doc, root.Content); err != nil {
		return nil, fmt.Errorf("index error: %w", err)
	}

	return doc, nil
}

func indexNodes(doc *Doc, nodes []*yaml.Node) error {
	for _, n := range nodes {
		if n.Anchor != "" {
			id := n.Anchor
			doc.NodesByID[id] = &Node{
				RefID:    id,
				AST:      n,
				AnchorID: n.Anchor,
				Line:     n.Line,
				Column:   n.Column,
			}
		}
		if n.Kind == yaml.SequenceNode || n.Kind == yaml.MappingNode {
			if err := indexNodes(doc, n.Content); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Doc) ResolveAlias(alias *yaml.Node) (*Node, error) {
	if alias.Kind != yaml.AliasNode {
		return nil, fmt.Errorf("node is not alias: %v", alias)
	}
	target, ok := d.NodesByID[alias.Value]
	if !ok {
		return nil, fmt.Errorf("unresolved alias: *%s", alias.Value)
	}
	return target, nil
}

func LoadWithOptions(r io.Reader, opts LoadOptions) (*Doc, error) {
	doc, err := ParseYAML(r)
	if err != nil {
		return nil, err
	}

	if opts.IncludeTag != "" {
		if opts.BasePath == "" {
			return nil, fmt.Errorf("BasePath must be set when IncludeTag is used")
		}
		visited := make(map[string]bool)
		if err := resolveIncludes(doc.RawTree, opts.BasePath, opts.IncludeTag, visited); err != nil {
			return nil, err
		}
	}

	return doc, nil
}

func resolveIncludes(node *yaml.Node, currentDir string, includeTag string, visited map[string]bool) error {
	if node.Tag == includeTag {
		includePath := filepath.Join(currentDir, node.Value)
		cleanPath := filepath.Clean(includePath)

		if strings.Contains(node.Value, "..") {
			return fmt.Errorf("parent traversal forbidden in include: %s", node.Value)
		}

		// if !strings.HasPrefix(cleanPath, currentDir) {
		// 	return fmt.Errorf("included file escapes base directory: %s", cleanPath)
		// }

		absBase, err := filepath.Abs(currentDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute base path: %w", err)
		}

		absTarget, err := filepath.Abs(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute target path: %w", err)
		}

		if !strings.HasPrefix(absTarget, absBase) {
			return fmt.Errorf("included file escapes base directory: %s", node.Value)
		}

		if visited[cleanPath] {
			return fmt.Errorf("recursive inclusion detected: %s", cleanPath)
		}
		visited[cleanPath] = true

		content, err := os.ReadFile(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to read included file %s: %w", cleanPath, err)
		}

		var includedNode yaml.Node
		if err := yaml.Unmarshal(content, &includedNode); err != nil {
			return fmt.Errorf("failed to parse included file %s: %w", cleanPath, err)
		}

		// Recursively process includes inside included content
		if err := resolveIncludes(&includedNode, filepath.Dir(cleanPath), includeTag, visited); err != nil {
			return err
		}

		// Replace this !include node
		*node = *includedNode.Content[0]
	}

	for _, child := range node.Content {
		if err := resolveIncludes(child, currentDir, includeTag, visited); err != nil {
			return err
		}
	}

	return nil
}
