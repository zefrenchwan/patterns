package patterns

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Entity is the base object in the patterns system.
// Basically, it is anything that may appear in a system: an object, a person, an event, a relation, anything.
// An entity may not link other entities or become a relation (with operands).
type Entity struct {
	// id of the entity, should be unique
	id string
	// labels are case insensitive information about the entity
	labels []string
	// content is the activity of the entity and its attributes
	content ActiveTimeValues
	// operands are other entities
	operands []string
}

// Id returns the id of the entity
func (e *Entity) Id() string {
	return e.id
}

// NewEntity builds an entity with given labels
func NewEntity(labels []string) Entity {
	res, _ := NewEntityDuring(labels, NewFullPeriod())
	return res
}

// NewEntityDuring builds an entity with given labels.
// Entity is active only during given period.
// If period is empty, it returns an error
func NewEntityDuring(labels []string, period Period) (Entity, error) {
	var result Entity
	if period.IsEmptyPeriod() {
		return result, errors.New("empty period for Entity")
	}

	values := NewActiveTimeValues()
	values.SetActivity(period)
	result.content = values
	result.id = uuid.NewString()
	result.labels = append(result.labels, labels...)

	return result, nil
}

// NewRelationDuring returns a relation on its linked elements, active during a given period
func NewRelationDuring(labels []string, period Period, linkedElements []Entity) (Entity, error) {
	var result Entity
	if res, err := NewEntityDuring(labels, period); err != nil {
		return result, err
	} else {
		result = res
	}

	if err := result.SetRelation(linkedElements); err != nil {
		return result, err
	}
	return result, nil
}

// AddLabel appends a label to the set of labels (no duplicate)
func (t *Entity) AddLabel(label string) {
	if t == nil {
		return
	}

	t.labels = append(t.labels, label)
}

// Labels returns the labels as a sorted slice and nil for nil
// Sorting allows to compare labels using the order.
func (t *Entity) Labels() []string {
	if t == nil {
		return nil
	} else if len(t.labels) == 0 {
		return nil
	}

	values := make([]string, len(t.labels))
	copy(values, t.labels)
	slices.Sort(values)
	return values
}

// RemoveLabel removes, if any, the label key
func (t *Entity) RemoveLabel(key string) {
	if t == nil {
		return
	}

	if !slices.ContainsFunc(t.labels, func(a string) bool { return strings.EqualFold(a, key) }) {
		t.labels = append(t.labels, key)
	}
}

// ClearRelation deletes all operands (but keeps related entities)
func (e *Entity) ClearRelation() {
	if e != nil {
		e.operands = nil
	}
}

// SetRelation adds other elements as operands
func (e *Entity) SetRelation(linkedElements []Entity) error {
	if e == nil {
		return errors.New("nil entity")
	} else if len(linkedElements) == 0 {
		return errors.New("not enough linked elements")
	}

	e.operands = nil
	e.operands = make([]string, len(linkedElements))
	for index, element := range linkedElements {
		e.operands[index] = element.Id()
	}

	return nil
}

// IsRelation returns true if entity is not nil and is linked to others
func (e *Entity) IsRelation() bool {
	return e != nil && len(e.operands) != 0
}

// ContainsAttribute returns true if receiver is not nil and it contains a non nil entry with that key
func (e *Entity) ContainsAttribute(attr string) bool {
	if e == nil {
		return false
	}

	return e.content.ContainsAttribute(attr)
}

// Attributes returns the sorted slice of all attributes.
// Nil receiver returns nil
func (e *Entity) Attributes() []string {
	if e == nil {
		return nil
	}

	return e.content.Attributes()
}

// SetValue sets a value for an attribute, for the full period.
func (e *Entity) SetValue(attribute string, value string) error {
	if e == nil {
		return errors.New("nil instance")
	}

	return e.content.SetValue(attribute, value)
}

// AddValue sets the value of an attribute during a given period.
// It updates the periods of the other values (for the same attribute) accordingly.
// It returns an error if receiver is nil
func (e *Entity) AddValue(attribute string, value string, validity Period) error {
	if e == nil {
		return errors.New("nil instance")
	}

	return e.content.AddValue(attribute, value, validity)
}

// ValuesForAttribute returns the values for an attribute as a sorted slice during the activity of the entity.
// For instance, if activity is [now, +oo[ and values are set for ] -oo, now - 1 day] , then it returns nil
func (e *Entity) ValuesForAttribute(attribute string) ([]string, error) {
	if e == nil {
		return nil, errors.New("nil instance")
	}

	return e.content.ValuesForAttribute(attribute)
}

// TimeValuesForAttribute returns, for each value of the attribute, the matching time intervals
func (e *Entity) TimeValuesForAttribute(attribute string) (map[string][]Interval[time.Time], error) {
	if e == nil {
		return nil, errors.New("nil entity")
	}

	return e.content.TimeValuesForAttribute(attribute)
}

// AddActivity sets p as active
func (e *Entity) AddActivity(p Period) error {
	if e == nil {
		return errors.New("nil entity")
	}

	return e.AddActivity(p)
}

// RemoveActivity flags p as inactive
func (e *Entity) RemoveActivity(p Period) error {
	if e == nil {
		return errors.New("nil entity")
	}

	return e.content.RemoveActivity(p)
}

// SetActivity sets the period of activity no matter previous value
func (e *Entity) SetActivity(p Period) error {
	if e == nil {
		return errors.New("nil entity")
	}

	return e.content.SetActivity(p)
}

// IsInactive returns true if it is is never active
func (e *Entity) IsInactive() bool {
	return e == nil || e.content.IsEmpty()
}

// IsActiveDuring returns true if p and the entity have at least a common point
func (e *Entity) IsActiveDuring(p Period) bool {
	if e == nil {
		return false
	}

	return e.content.IsActiveDuring(p)
}
