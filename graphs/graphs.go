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
	EquivalenceClass map[string]string
	// SourceGraph contains the source graph
	SourceGraph string
	// value is the actual displayed value
	Value nodes.Element
	// Editable is true if current user may modify the node
	Editable bool
}

// Graph defines a graph
type Graph struct {
	// Id is the id of the graph
	Id string
	// Name is the name of the graph to be displayed
	Name string
	// Description of the graph
	Description string
	// Metadata for graph contains any key values for that graph.
	// If user needs a list, it is ready.
	// If user needs key - value, convention is that first value in list is value.
	// If key only matters (for a label), it is possible with values = nil
	Metadata map[string][]string
	// values are the nodes to display, key is the id of the element.
	// Those values to display may come from the graph (owned) or from imported graphs
	values map[string]Node
	// dirtyNodes contain the nodes that were changed since load (to ease graph updates)
	dirtyNodes []string
}

// NewEmptyGraph returns a new empty graph
func NewEmptyGraph() Graph {
	return Graph{
		Metadata:   make(map[string][]string),
		values:     make(map[string]Node),
		dirtyNodes: nil,
	}
}

// NewGraph constructs a new graph
func NewGraph(name, description string) Graph {
	return NewGraphWithId(uuid.NewString(), name, description)
}

// NewGraphWithId builds a new graph with a given id, name and description
func NewGraphWithId(id, name, description string) Graph {
	graph := NewEmptyGraph()
	graph.Id = id
	graph.Name = name
	graph.Description = description
	return graph
}

// MarkExistingElementAsDirty flags a node as dirty, to save
func (g *Graph) MarkExistingElementAsDirty(currentElement nodes.Element) error {
	if g == nil || currentElement == nil {
		return errors.New("nil value")
	}

	elementId := currentElement.Id()
	currentNode, found := g.values[elementId]
	if !found {
		return errors.New("node did not exist")
	} else if !currentNode.Editable {
		return errors.New("node is not editable")
	} else if g.dirtyNodes == nil {
		g.dirtyNodes = []string{elementId}
	} else if !slices.Contains(g.dirtyNodes, elementId) {
		g.dirtyNodes = append(g.dirtyNodes, elementId)
	}

	currentNode.Value = currentElement
	g.values[elementId] = currentNode

	return nil
}

// SetElement set values (overwrites a previous one if any) for a given element
func (g *Graph) SetElement(currentElement nodes.Element, sourceGraph string, editable bool, equivalenceClassByGaph map[string]string) {
	if g == nil || currentElement == nil {
		return
	}

	elementId := currentElement.Id()
	_, found := g.values[elementId]
	if found {
		delete(g.values, elementId)
	}

	g.values[elementId] = Node{
		EquivalenceClass: equivalenceClassByGaph,
		SourceGraph:      sourceGraph,
		Value:            currentElement,
		Editable:         editable,
	}
}

// CreateNodeFrom adds the value in this graph from an existing one.
// Typical use case is when a node comes from another graph (layer) but is changed to become a node in the new graph.
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

	nodeValue.SourceGraph = g.Id
	nodeValue.Value = newValue
	nodeValue.EquivalenceClass[id] = g.Id
	delete(g.values, source)
	g.values[id] = nodeValue
	g.dirtyNodes = append(g.dirtyNodes, id)
	return nil
}

// Nodes returns all the nodes in the graph
func (g *Graph) Nodes() []Node {
	if g == nil {
		return nil
	}

	var result []Node
	for _, node := range g.values {
		result = append(result, node)
	}

	return result
}

// DirtyNodes returns the nodes to save, either an empty slice, or the values
func (g *Graph) DirtyNodes() []Node {
	if g == nil || len(g.dirtyNodes) == 0 {
		return []Node{}
	}

	var result []Node
	for _, node := range g.values {
		result = append(result, node)
	}

	return result
}
