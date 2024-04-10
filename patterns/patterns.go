package patterns

import "errors"

// Pattern defines its name, all elements to link together, its hierarchy (parents and childs)
type Pattern struct {
	// name of the pattern, assumed to be unique
	name string
	// definitions of relations and traits
	definitions Dictionary
	// parents of the pattern, to define elements if not found in this pattern
	parents map[string]*Pattern
}

// NewPattern returns a new empty pattern
func NewPattern(name string) Pattern {
	return Pattern{
		name:        name,
		parents:     make(map[string]*Pattern),
		definitions: NewDictionary(),
	}
}

// AddParentName adds a parent, but not its content
func (p *Pattern) AddParentName(name string) error {
	if p == nil {
		return errors.New("nil pattern")
	} else if p.parents == nil {
		p.parents = make(map[string]*Pattern)
	}

	if _, found := p.parents[name]; !found {
		p.parents[name] = nil
	}

	return nil
}

// AddParent adds a parent pattern
func (p *Pattern) AddParent(name string, parent *Pattern) error {
	if p == nil {
		return errors.New("nil pattern")
	} else if p.parents == nil {
		p.parents = make(map[string]*Pattern)
	}

	p.parents[name] = parent

	return nil
}

// AddTrait adds a trait to the patterns
func (p *Pattern) AddTrait(trait string) error {
	if p == nil {
		return errors.New("nil pattern")
	}

	p.definitions.AddTrait(trait)

	return nil
}

// AddTraitParent adds an inheritance link
func (p *Pattern) AddTraitParent(trait, parentTrait string) error {
	if p == nil {
		return errors.New("nil pattern")
	}

	return p.definitions.AddTraitsLink(trait, parentTrait)
}

// AddRelation adds a full relation to the pattern
func (p *Pattern) AddRelation(trait string, roleTraitsMap map[string][]string) error {
	if p == nil {
		return errors.New("nil pattern")
	}

	return p.definitions.AddRelation(trait, roleTraitsMap)
}
