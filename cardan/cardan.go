package cardan

import (
	"fmt"
	"io"

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
