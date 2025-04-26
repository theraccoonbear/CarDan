package cardan

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// CarDan holds both a raw YAML AST and metadata
type CarDan struct {
	RootNode *yaml.Node
	Anchors  map[string]*yaml.Node
	Aliases  map[string]string // alias value -> anchor ID
}

// Load parses a YAML file into a CarDan document (raw AST only)
func Load(r io.Reader) (*CarDan, error) {
	var root yaml.Node
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	if len(root.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}

	first := root.Content[0]

	anchors := make(map[string]*yaml.Node)
	aliases := make(map[string]string)

	if err := walkAndIndex(first, anchors, aliases); err != nil {
		return nil, fmt.Errorf("failed to index anchors/aliases: %w", err)
	}

	return &CarDan{
		RootNode: first,
		Anchors:  anchors,
		Aliases:  aliases,
	}, nil
}

// Unmarshal decodes the (already resolved) Node into a Go struct
func (c *CarDan) Unmarshal(out any) error {
	return c.RootNode.Decode(out)
}

// walkAndIndex builds the anchor and alias maps
func walkAndIndex(n *yaml.Node, anchors map[string]*yaml.Node, aliases map[string]string) error {
	if n.Anchor != "" {
		anchors[n.Anchor] = n
	}
	if n.Kind == yaml.AliasNode {
		aliases[n.Value] = n.Value
	}
	for _, child := range n.Content {
		if err := walkAndIndex(child, anchors, aliases); err != nil {
			return err
		}
	}
	return nil
}

// GetRawAnchor returns the raw YAML node for a given anchor
func (c *CarDan) GetRawAnchor(id string) *yaml.Node {
	return c.Anchors[id]
}

// ResolveRefs rewrites a field (by key name) in all mapping nodes to use anchor IDs
func (c *CarDan) ResolveRefs(field string) error {
	return resolveFieldRefs(c.RootNode, field, c.Aliases)
}

func resolveFieldRefs(n *yaml.Node, field string, aliases map[string]string) error {
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content)-1; i += 2 {
			k := n.Content[i]
			v := n.Content[i+1]
			if k.Value == field {
				// field found, now handle list of aliases
				if v.Kind == yaml.SequenceNode {
					for _, item := range v.Content {
						if item.Kind == yaml.AliasNode {
							anchor, ok := aliases[item.Value]
							if !ok {
								return fmt.Errorf("unresolved alias: *%s", item.Value)
							}
							item.Kind = yaml.ScalarNode
							item.Tag = "!!str"
							item.Value = anchor
						}
					}
				} else if v.Kind == yaml.AliasNode {
					// fallback if not a list
					anchor, ok := aliases[v.Value]
					if !ok {
						return fmt.Errorf("unresolved alias: *%s", v.Value)
					}
					v.Kind = yaml.ScalarNode
					v.Tag = "!!str"
					v.Value = anchor
				}
			}
		}
	}

	// Always keep walking recursively
	for _, child := range n.Content {
		if err := resolveFieldRefs(child, field, aliases); err != nil {
			return err
		}
	}
	return nil
}
