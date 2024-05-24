package graphs

import (
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/zefrenchwan/patterns.git/nodes"
)

// Node defines an element visible in the graph
type Node struct {
	// equivalenceClass defines the equivalent elements (key) and their source graph
	equivalenceClass map[string]string
	// conflict is false if all the values in the equivalence class are the same, false otherwise
	conflict bool
	// sourceGraph contains the source graph
	sourceGraph string
	// value is the actual displayed value
	value nodes.Element
}

// Graph defines a graph
type Graph struct {
	// Id is the id of the graph
	Id string
	// Name is the name of the graph to be displayed
	Name string
	// Description of the graph
	Description string
	// values are the nodes to display, key is the id of the element.
	// Those values to display may come from the graph (owned) or from imported graphs
	values map[string]Node
	// dirtyNodes contain the nodes that were changed since load (to ease graph updates)
	dirtyNodes []string
}

// NewGraph constructs a new graph
func NewGraph(name, description string) Graph {
	return Graph{
		Id:          uuid.NewString(),
		Name:        name,
		Description: description,
		values:      make(map[string]Node),
		dirtyNodes:  nil,
	}
}

// CreateNodeFrom adds the value
func (g *Graph) CreateNodeFrom(newValue nodes.Element, source string) error {
	if g == nil {
		return nil
	}

	id := newValue.Id()
	nodeValue, found := g.values[source]
	if !found {
		return errors.New("source node does not exist")
	} else if source != id {
		return errors.New("same id for source and new value")
	}

	nodeValue.conflict = false
	nodeValue.sourceGraph = g.Id
	nodeValue.value = newValue
	nodeValue.equivalenceClass[id] = g.Id
	delete(g.values, source)
	g.values[id] = nodeValue
	g.dirtyNodes = append(g.dirtyNodes, id)
	return nil
}

// UpsertOwnedNode changes the node in the graph if it owns the node
func (g *Graph) UpsertOwnedNode(element nodes.Element) {
	if g == nil {
		return
	}

	id := element.Id()
	valueForNode := g.values[id]

	if slices.Contains(g.dirtyNodes, id) {
		valueForNode.value = element
	} else {
		g.dirtyNodes = append(g.dirtyNodes, id)
		valueForNode = Node{
			equivalenceClass: nil,
			conflict:         false,
			sourceGraph:      g.Id,
			value:            element,
		}
	}

	g.values[id] = valueForNode
}
