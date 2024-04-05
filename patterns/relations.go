package patterns

import (
	"errors"
	"slices"

	"github.com/google/uuid"
)

const (
	// SUBJECT_ROLE defines the subject of the relation.
	// For instance, Loves(x, z), then x is the subject
	SUBJECT_ROLE = "subject"
	// OBJECT_ROLE is used to set a direct link with someone / something.
	// For instance: loves(x, y), x is the subject, and y is the object (even if it is indeed a person)
	OBJECT_ROLE = "object"
	// LOCATION_ROLE is used to set a location.
	// For instance, meeting(person, person, restaurant)
	LOCATION_ROLE = "location"
	// MEANS_ROLE is used to define how the action took place.
	// For instance, travel(person, plane), then plane is the means used
	MEANS_ROLE = "means"
	// CAUSE_ROLE is used to explain cause for this link.
	// For instance, give(person, person, money, love), love is the cause of the action
	CAUSE_ROLE = "cause"
	// GOAL_ROLE defines a goal for a link.
	// For instance, marry(person, person, money), money is the goal
	// (no comment on the example, their purpose is to offer clear situations to understand)
	GOAL_ROLE = "goal"
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
	// links maps a role to a group of
	links map[string][]string
}

// NewUnlinkedRelation creates a relation with given traits
func NewUnlinkedRelation(traits []string) Relation {
	return NewUnlinkedRelationWithId(uuid.NewString(), traits)
}

// NewUnlinkedRelationWithId creates a relation with provided id and given traits
func NewUnlinkedRelationWithId(id string, traits []string) Relation {
	var relation Relation
	relation.id = id
	relation.activity = NewFullPeriod()
	relation.traits = append(relation.traits, traits...)
	slices.Sort(relation.traits)
	relation.traits = slices.Compact(relation.traits)
	return relation
}

// NewRelation creates a relation with given traits for a given subject.
func NewRelation(subject string, traits []string) Relation {
	return NewRelationWithId(uuid.NewString(), subject, traits)
}

// NewTimeDependentRelation returns a relation true for a given period
func NewTimeDependentRelation(subject string, traits []string, period Period) Relation {
	result := NewRelation(subject, traits)
	result.activity = NewPeriodCopy(period)
	return result
}

// NewRelationWithId creates a relation with a given id
func NewRelationWithId(id string, subject string, traits []string) Relation {
	return NewMultiRelationWithId(id, []string{subject}, traits)
}

// NewMultiRelation creates a relation with given traits and multiple subjects.
// It allows subject to be a group of persons, for instance
func NewMultiRelation(subjects []string, traits []string) Relation {
	return NewMultiRelationWithId(uuid.NewString(), subjects, traits)
}

// NewTimeDependentMultiRelation returns a new relation with given traits, subjects, and true during period only
func NewTimeDependentMultiRelation(subjects []string, traits []string, period Period) Relation {
	result := NewMultiRelationWithId(uuid.NewString(), subjects, traits)
	result.activity = NewPeriodCopy(period)
	return result
}

// NewMultiRelationWithId creates a new multi relation with a specific id
func NewMultiRelationWithId(id string, subjects []string, traits []string) Relation {
	var result Relation
	result.id = id
	result.activity = NewFullPeriod()

	if len(subjects) != 0 {
		result.links = make(map[string][]string)
		copySubject := make([]string, len(subjects))
		copy(copySubject, subjects)
		result.links[SUBJECT_ROLE] = copySubject
	}

	if len(traits) == 0 {
		return result
	}

	singleTraits := make([]string, len(traits))
	copy(singleTraits, traits)
	slices.Sort(singleTraits)
	result.traits = slices.Compact(singleTraits)
	return result
}

// Id returns the id of the relation
func (r *Relation) Id() string {
	var result string
	if r != nil {
		result = r.id
	}

	return result
}

// GetSubjets returns the subjects of the relation, nil for nil, empty for no subject
func (r *Relation) GetSubjects() []string {
	if r == nil {
		return nil
	} else if len(r.links) == 0 {
		return []string{}
	}

	values := r.links[SUBJECT_ROLE]
	if len(values) == 0 {
		return []string{}
	}
	copySubjects := make([]string, len(values))
	copy(copySubjects, values)
	return copySubjects
}

// GetValuesPerRole returns the values per role, including the subject, for that relation
func (r *Relation) GetValuesPerRole() map[string][]string {
	if r == nil {
		return nil
	}

	result := make(map[string][]string)
	for role, values := range r.links {
		if len(values) == 0 {
			continue
		}

		copyValues := make([]string, len(values))
		copy(copyValues, values)
		result[role] = copyValues
	}

	return result
}

// SetValueForRole sets the value for the role, no matter previous value
func (r *Relation) SetValueForRole(role string, linkedId string) error {
	return r.SetValuesForRole(role, []string{linkedId})
}

// SetValuesForRole sets the values for a given role
func (r *Relation) SetValuesForRole(role string, linkedIds []string) error {
	if r == nil {
		return errors.New("nil relation")
	}

	if r.links == nil {
		r.links = make(map[string][]string)
	}

	r.links[role] = make([]string, len(linkedIds))
	copy(r.links[role], linkedIds)
	return nil
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

	r.activity.Remove(period)
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
