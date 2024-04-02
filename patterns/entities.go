package patterns

import (
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
)

type Element interface {
	// Id returns the (unique) id of the element
	Id()

	// ActivePeriod returns the period the entity was active during
	ActivePeriod() Period
	// AddActivePeriod flags given period as active
	AddActivePeriod(Period) error
	// RemoveActivePeriod flags given period as inactive
	RemoveActivePeriod(Period)
	// SetActivePeriod forces the activity for that element
	SetActivePeriod(Period) error
	// IsActiveDuring returns true if the element is active at least one moment during the period
	IsActiveDuring(Period) bool

	// Traits returns all the traits an element implements
	Traits() []string
	// AddTrait adds a trait if not already present
	AddTrait(string) error
	// RemoveTrait removes a trait to an element
	RemoveTrait(string)

	// ContainsAttribute returns true if the element is not nil and contains a value for that attribute
	ContainsAttribute(string) bool
	// Attributes returns all the attributes of the element
	Attributes() []string
	// SetValue sets the value for that attribute during the full period
	SetValue(string, string) error
	// AddValue adds to the attribute and value the period
	AddValue(string, string, Period) error
	// ValuesForAttribute returns the value of the attribute during the period activity
	ValuesForAttribute(attribute string) ([]string, error)
	// PeriodValuesForAttribute returns the values and matching period for a given attribute
	PeriodValuesForAttribute(attribute string) (map[string]Period, error)
}

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

	t.traits = append(t.traits, trait)
	return nil
}

// Traits returns the traits an entity implements, as a sorted slice and nil for nil
// Sorting allows to compare traits using the order.
func (t *Entity) Traits() []string {
	if t == nil {
		return nil
	} else if len(t.traits) == 0 {
		return nil
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
	}

	if !slices.ContainsFunc(t.traits, func(a string) bool { return strings.EqualFold(a, key) }) {
		t.traits = append(t.traits, key)
	}
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

// PeriodValuesForAttribute returns, for each value of the attribute, the matching period
func (e *Entity) PeriodValuesForAttribute(attribute string) (map[string]Period, error) {
	if e == nil {
		return nil, errors.New("nil entity")
	}

	return e.content.periodsForAttribute(attribute)
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
