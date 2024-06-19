package nodes

import (
	"errors"
	"slices"

	"github.com/google/uuid"
)

const (
	// RELATION_ROLE_SUBJECT defines the subject of the relation.
	// For instance, Loves(x, z), then x is the subject
	RELATION_ROLE_SUBJECT = "subject"
	// RELATION_ROLE_OBJECT is used to set a direct link with someone / something.
	// For instance: loves(x, y), x is the subject, and y is the object (even if it is indeed a person)
	RELATION_ROLE_OBJECT = "object"
	// RELATION_ROLE_LOCATION is used to set a location.
	// For instance, meeting(person, person, restaurant)
	RELATION_ROLE_LOCATION = "location"
)

// Relation defines a relation between elements.
// Think of a relation as a verb: it links a subject and other elements with roles.
// For instance, John meets Jack at the restaurant Z is a relation with:
// subject = John
// traits = "meeting"
// object = "Jack" (it is a person, but grammar calls this role "object")
// location = "the restaurant Z"
type Relation struct {
	// id of the instance of the relation
	id string
	// activtiy definew when the relation is true: t in period is equivalent to relation is true at t
	activity Period
	// traits for the relation (is, extends, ...)
	traits []string
	// links maps a role to a group of operands with their validity.
	// Implementing a period -> []string may be better in some occasions.
	// But wost case is 2^n if size of operands is n. So worst case would crash app.
	links map[string]map[string]Period
}

// NewRelation creates a relation with given traits
func NewRelation(traits []string) Relation {
	return NewRelationWithId(uuid.NewString(), traits)
}

// NewRelationWithId creates a relation with provided id and given traits
func NewRelationWithId(id string, traits []string) Relation {
	var relation Relation
	relation.id = id
	relation.activity = NewFullPeriod()
	relation.traits = append(relation.traits, traits...)
	slices.Sort(relation.traits)
	relation.traits = slices.Compact(relation.traits)
	return relation
}

// Id returns the id of the relation
func (r *Relation) Id() string {
	var result string
	if r != nil {
		result = r.id
	}

	return result
}

// ValuesPerRole returns the values per role, including the subject, for that relation
func (r *Relation) ValuesPerRole() map[string][]string {
	if r == nil {
		return nil
	}

	result := make(map[string][]string)
	for role, values := range r.links {
		if len(values) == 0 {
			continue
		}

		index := 0
		copyValues := make([]string, len(values))
		for value := range values {
			copyValues[index] = value
			index++
		}

		result[role] = copyValues
	}

	return result
}

// PeriodValuesPerRole returns, for a role, linked values and matching validity
func (r *Relation) PeriodValuesPerRole() map[string]map[string]Period {
	if r == nil {
		return nil
	}

	result := make(map[string]map[string]Period)
	for role, values := range r.links {
		result[role] = make(map[string]Period)
		for operand, period := range values {
			result[role][operand] = NewPeriodCopy(period)
		}
	}

	return result
}

// ClearRoleValues deletes all values for a given role
func (r *Relation) ClearRoleValues(role string) {
	if r == nil {
		return
	} else if len(r.links) == 0 {
		return
	}

	delete(r.links, role)
}

// AddPeriodValueForRole adds period for this element in that role
func (r *Relation) AddPeriodValueForRole(role string, linkedId string, period Period) error {
	if r == nil {
		return errors.New("nil relation")
	}

	if period.IsEmptyPeriod() {
		return nil
	}

	if r.links == nil {
		r.links = make(map[string]map[string]Period)
	}

	if r.links[role] == nil {
		r.links[role] = make(map[string]Period)
	}

	switch previousPeriod, found := r.links[role][linkedId]; found {
	case true:
		previousPeriod.Add(period)
		r.links[role][linkedId] = previousPeriod
	case false:
		r.links[role][linkedId] = NewPeriodCopy(period)
	}

	return nil
}

// RemovePeriodValueForRole removes period for this element in that role
func (r *Relation) RemovePeriodValueForRole(role string, linkedId string, period Period) error {
	if r == nil {
		return errors.New("nil relation")
	}

	if period.IsEmptyPeriod() || len(r.links) == 0 || len(r.links[role]) == 0 {
		return nil
	}

	if previous, found := r.links[role][linkedId]; found {
		previous.Remove(period)
		if previous.IsEmptyPeriod() {
			delete(r.links[role], linkedId)
		}
	}

	if len(r.links[role]) == 0 {
		delete(r.links, role)
	}

	return nil
}

// SetValuesForRole sets the values for a given role
func (r *Relation) SetValuesForRole(role string, linkedIds []string) error {
	if r == nil {
		return errors.New("nil relation")
	}

	r.ClearRoleValues(role)
	var globalErr error
	for _, linkedId := range linkedIds {
		currentErr := r.AddPeriodValueForRole(role, linkedId, NewFullPeriod())
		globalErr = errors.Join(globalErr, currentErr)
	}

	return globalErr
}

// AddTrait appends a trait to the set of traits (no duplicate)
func (r *Relation) AddTrait(trait string) error {
	if r == nil {
		return errors.New("nil relation")
	}

	if len(r.traits) == 0 {
		r.traits = append(r.traits, trait)
	} else if !slices.Contains(r.traits, trait) {
		r.traits = append(r.traits, trait)
	}

	return nil
}

// Traits returns the traits an entity implements, as a sorted slice and nil for nil
// Sorting allows to compare traits using the order.
func (r *Relation) Traits() []string {
	if r == nil {
		return nil
	} else if len(r.traits) == 0 {
		return []string{}
	}

	values := make([]string, len(r.traits))
	copy(values, r.traits)
	slices.Sort(values)
	return values
}

// RemoveTrait removes, if any, the trait
func (r *Relation) RemoveTrait(key string) {
	if r == nil {
		return
	} else if len(r.traits) == 0 {
		return
	}

	deleteFn := func(element string) bool { return element == key }
	r.traits = slices.DeleteFunc(r.traits, deleteFn)
}

// ActivePeriod returns the period the relation was active during
func (r *Relation) ActivePeriod() Period {
	if r == nil {
		return NewEmptyPeriod()
	}

	return NewPeriodCopy(r.activity)
}

// AddActivePeriod flags given period as active
func (r *Relation) AddActivePeriod(period Period) error {
	if r == nil {
		return errors.New("nil relation")
	}

	r.activity.Add(period)
	return nil
}

// RemoveActivePeriod flags given period as inactive
func (r *Relation) RemoveActivePeriod(period Period) {
	if r == nil {
		return
	}

	r.activity.Remove(period)
}

// SetActivePeriod forces the activity for that relation
func (r *Relation) SetActivePeriod(period Period) error {
	if r == nil {
		return errors.New("nil relation")
	}

	r.activity = NewPeriodCopy(period)
	return nil
}

// IsActiveDuring returns true if the element is active at least one moment during the period
func (r *Relation) IsActiveDuring(period Period) bool {
	if r == nil {
		return false
	}

	copyPeriod := NewPeriodCopy(r.activity)
	copyPeriod.Intersection(period)
	return !copyPeriod.IsEmptyPeriod()
}
