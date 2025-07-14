package cardan

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type DAGNode struct {
	ID        string
	DependsOn []*DAGNode
	AST       *yaml.Node
	Visited   bool
	Visiting  bool
}

type DAG struct {
	Nodes map[string]*DAGNode
}

func BuildDAG(doc *Doc, key string) (*DAG, error) {
	dag := &DAG{Nodes: make(map[string]*DAGNode)}

	for id, node := range doc.NodesByID {
		dagNode := &DAGNode{
			ID:  id,
			AST: node.AST,
		}
		dag.Nodes[id] = dagNode
	}

	for id, dagNode := range dag.Nodes {
		dn := dagNode.AST
		if dn.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(dn.Content); i += 2 {
			k := dn.Content[i]
			v := dn.Content[i+1]
			if k.Value == key {
				if v.Kind == yaml.SequenceNode {
					for _, item := range v.Content {
						resolved, err := doc.ResolveAlias(item)
						if err != nil {
							return nil, fmt.Errorf("%s: bad depends_on ref: %w", id, err)
						}
						depNode, ok := dag.Nodes[resolved.RefID]
						if !ok {
							return nil, fmt.Errorf("%s: unresolved DAG node: %s", id, resolved.RefID)
						}
						dagNode.DependsOn = append(dagNode.DependsOn, depNode)
					}
				}
			}
		}
	}

	return dag, nil
}

func (g *DAG) DetectCycles() error {
	for _, node := range g.Nodes {
		if err := visit(node); err != nil {
			return err
		}
	}
	return nil
}

func visit(n *DAGNode) error {
	if n.Visited {
		return nil
	}
	if n.Visiting {
		return fmt.Errorf("cycle detected at node: %s", n.ID)
	}
	n.Visiting = true
	for _, dep := range n.DependsOn {
		if err := visit(dep); err != nil {
			return err
		}
	}
	n.Visiting = false
	n.Visited = true
	return nil
}
