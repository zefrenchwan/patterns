package patterns

type Pattern struct {
	name        string
	definitions Dictionary
	parents     map[string]*Pattern
	childs      map[string]*Pattern
}

func NewPattern(name string) Pattern {
	return Pattern{
		name:        name,
		parents:     make(map[string]*Pattern),
		childs:      make(map[string]*Pattern),
		definitions: NewDictionary(),
	}
}
