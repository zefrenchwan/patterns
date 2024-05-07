package nodes

import (
	"errors"
	"slices"

	"github.com/google/uuid"
)

// Entity represents real world objects, not relations
type Entity struct {
	// id of the entity, should be unique
	id string
	// traits are case insensitive information about the entity semantic
	traits []string
	// content is the activity of the entity and its attributes
	content ActiveTimeValues
}

// Id returns the id of the entity
func (e *Entity) Id() string {
	return e.id
}

// NewEntity builds an entity with given traits
func NewEntity(traits []string) Entity {
	res, _ := NewEntityDuring(traits, NewFullPeriod())
	return res
}

// NewEntityDuring builds an entity with given traits.
// Entity is active only during given period.
// If period is empty, it returns an error
func NewEntityDuring(traits []string, period Period) (Entity, error) {
	return NewEntityWithId(uuid.NewString(), traits, period)
}

// NewEntityWithId builds a new entity with a given id and traits, active for given period
func NewEntityWithId(id string, traits []string, period Period) (Entity, error) {
	var result Entity
	if period.IsEmptyPeriod() {
		return result, errors.New("empty period for Entity")
	}

	values := NewActiveTimeValues()
	values.SetActivity(period)
	result.content = values
	result.id = id
	result.traits = append(result.traits, traits...)

	return result, nil
}

// AddTrait appends a trait to the set of traits (no duplicate)
func (t *Entity) AddTrait(trait string) error {
	if t == nil {
		return errors.New("nil entity")
	}

	if len(t.traits) == 0 {
		t.traits = append(t.traits, trait)
	} else if !slices.Contains(t.traits, trait) {
		t.traits = append(t.traits, trait)
	}

	return nil
}

// Traits returns the traits an entity implements, as a sorted slice and nil for nil
// Sorting allows to compare traits using the order.
func (t *Entity) Traits() []string {
	if t == nil {
		return nil
	} else if len(t.traits) == 0 {
		return []string{}
	}

	values := make([]string, len(t.traits))
	copy(values, t.traits)
	slices.Sort(values)
	return values
}

// RemoveTrait removes, if any, the trait
func (t *Entity) RemoveTrait(key string) {
	if t == nil {
		return
	} else if len(t.traits) == 0 {
		return
	}

	deleteFn := func(element string) bool { return element == key }
	t.traits = slices.DeleteFunc(t.traits, deleteFn)
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

// SetPeriodForValue sets the value and the period for that attribute.
func (e *Entity) SetPeriodForValue(attribute string, value string, period Period) error {
	if e == nil {
		return errors.New("nil instance")
	}

	return e.content.SetPeriodForValue(attribute, value, period)
}

// RemovePeriodForAttribute just removes period, no matter the value, for that attribute
func (e *Entity) RemovePeriodForAttribute(attribute string, period Period) error {
	if e == nil {
		return errors.New("nil instance")
	}

	return e.content.RemovePeriodForAttribute(attribute, period)
}

// ValuesForAttribute returns the values for an attribute as a sorted slice during the activity of the entity.
// For instance, if activity is [now, +oo[ and values are set for ] -oo, now - 1 day] , then it returns nil
func (e *Entity) ValuesForAttribute(attribute string) ([]string, error) {
	if e == nil {
		return nil, errors.New("nil instance")
	}

	return e.content.ValuesForAttribute(attribute)
}

// PeriodValuesForAttribute returns, for each value of the attribute, the matching period
func (e *Entity) PeriodValuesForAttribute(attribute string) (map[string]Period, error) {
	if e == nil {
		return nil, errors.New("nil entity")
	}

	return e.content.PeriodsForAttribute(attribute)
}

// ActivePeriod returns the period the entity was active during
func (e *Entity) ActivePeriod() Period {
	result := NewEmptyPeriod()
	if e == nil {
		return result
	}

	return NewPeriodCopy(e.content.periodOfActivity)
}

// AddActivePeriod flags given period as active
func (e *Entity) AddActivePeriod(p Period) error {
	if e == nil {
		return errors.New("nil entity")
	}

	return e.content.AddActivity(p)
}

// RemoveActivePeriod flags p as inactive
func (e *Entity) RemoveActivePeriod(p Period) {
	if e != nil {
		e.content.RemoveActivity(p)
	}
}

// SetActivePeriod sets the period of activity no matter previous value
func (e *Entity) SetActivePeriod(p Period) error {
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
