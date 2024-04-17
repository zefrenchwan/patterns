package storage

// ActiveEntity is a DTO to deal with entities
type ActiveEntity struct {
	Id       string              `json:"id"`
	Traits   []string            `json:"traits"`
	Values   []ActiveEntityValue `json:"values"`
	Activity []string            `json:"activity,omitempty"`
}

// ActiveEntityValue is a DTO for entity values
type ActiveEntityValue struct {
	AttributeName  string   `json:"attribute"`
	AttributeValue string   `json:"value"`
	Periods        []string `json:"vaiidity,omitempty"`
}

// ActiveRelation is a DTO for relations
type ActiveRelation struct {
	Id       string              `json:"id"`
	Traits   []string            `json:"traits"`
	Activity []string            `json:"activity,omitempty"`
	Roles    map[string][]string `json:"roles,omitempty"`
}
