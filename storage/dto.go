package storage

// ActiveEntity is a snapshot of an entity at a given time
type ActiveEntity struct {
	Id     string              `json:"id"`
	Traits []string            `json:"traits"`
	Values []ActiveEntityValue `json:"values"`
}

// ActiveEntityValue is an entity value at a given time
type ActiveEntityValue struct {
	AttributeName  string `json:"attribute"`
	AttributeValue string `json:"value"`
}
