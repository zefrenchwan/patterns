package patterns

import (
	"errors"
	"slices"
)

// traitsInformation contains any information for a trait.
// So far, those are sub and super traits.
type traitsInformation struct {
	// subTraits are traits that extends current trait.
	// For instance, dog is a sub trait of animal
	subTraits []string
	// superTraits are traits that are extensions of current trait.
	// For instance, animal is a super trait of dog
	superTraits []string
}

// appendTraitsInformation adds all the traits informations to make one single traitInformation
// If result is empty, it returns nil
func appendTraitsInformation(values ...*traitsInformation) *traitsInformation {
	var result *traitsInformation
	for _, value := range values {
		if value == nil {
			continue
		}

		if result == nil {
			result = new(traitsInformation)
		}

		result.subTraits = append(result.subTraits, value.subTraits...)
		result.superTraits = append(result.superTraits, value.superTraits...)

		if len(result.subTraits) != 0 {
			slices.Sort(result.subTraits)
			result.subTraits = slices.Compact(result.subTraits)
		}

		if len(result.superTraits) != 0 {
			slices.Sort(result.superTraits)
			result.superTraits = slices.Compact(result.superTraits)
		}
	}

	return result
}

// relationsInformation defines meta data for a relation: accepted types of parameters, sub and super relations
type relationsInformation struct {
	// subRelations of the relation.
	// For instance, friends is a subrelation of knows
	subRelations []string
	// superRelations of the relation.
	// For instance, couple is a super relation of married
	superRelations []string
	// traitsRoles links role with possible traits (superclasses only).
	// For instance, person knows would accept object roles to be animal, places, etc
	traitsRoles map[string][]string
}

// appendRelationsInformation append relations information
func appendRelationsInformation(values ...*relationsInformation) *relationsInformation {
	var result *relationsInformation

	for _, value := range values {
		if value == nil {
			continue
		}

		if result == nil {
			result = new(relationsInformation)
			result.traitsRoles = make(map[string][]string)
		}

		result.subRelations = append(result.subRelations, value.subRelations...)
		result.superRelations = append(result.superRelations, value.superRelations...)

		for k, v := range value.traitsRoles {
			if len(v) == 0 {
				continue
			} else if current, found := result.traitsRoles[k]; !found || current == nil {
				result.traitsRoles[k] = slices.Clone(v)
			} else {
				result.traitsRoles[k] = append(result.traitsRoles[k], v...)
			}

			slices.Sort(result.traitsRoles[k])
			result.traitsRoles[k] = slices.Compact(result.traitsRoles[k])
		}
	}

	return result
}

// Dictionary contains all the meta data about relations and traits.
// There is no name as an attribute of the struct.
// Reason is that dictionaries are context based, then will be grouped in themes.
type Dictionary struct {
	// traitsDictionary links a name to the trait information
	traitsDictionary map[string]*traitsInformation
	// relationsDictionary links a trait to all its info (parameters traits, subrelation, superrelation)
	relationsDictionary map[string]*relationsInformation
}

// NewDictionary returns an empty dictionary
func NewDictionary() Dictionary {
	return Dictionary{
		traitsDictionary:    make(map[string]*traitsInformation),
		relationsDictionary: make(map[string]*relationsInformation),
	}
}

// HasRelationTrait returns true if the dictionary has a relation trait named value
func (d *Dictionary) HasRelationTrait(value string) bool {
	if d == nil || d.traitsDictionary == nil {
		return false
	}

	_, found := d.relationsDictionary[value]
	return found
}

// HasEntityTrait returns true if the dictionary has a trait named value
func (d *Dictionary) HasEntityTrait(value string) bool {
	if d == nil || d.traitsDictionary == nil {
		return false
	}

	_, found := d.traitsDictionary[value]
	return found
}

// AddTrait appends a trait
func (d *Dictionary) AddTrait(trait string) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if _, found := d.traitsDictionary[trait]; !found {
		d.traitsDictionary[trait] = nil
	}

	return nil
}

// AddTraitsLink links a trait to a super trait.
// Example of call: d.AddTraitsLink("cat", "animal")
// If traits did not already exist, append in dictionary
func (d *Dictionary) AddTraitsLink(currentTrait string, superTrait string) error {
	if d == nil {
		return errors.New("nil dictionary")
	}

	if d.traitsDictionary == nil {
		d.traitsDictionary = make(map[string]*traitsInformation)
	}

	if d.traitsDictionary[currentTrait] == nil {
		d.traitsDictionary[currentTrait] = &traitsInformation{
			superTraits: []string{superTrait},
		}

	} else if !slices.Contains(d.traitsDictionary[currentTrait].superTraits, superTrait) {
		d.traitsDictionary[currentTrait].superTraits = append(d.traitsDictionary[currentTrait].superTraits, superTrait)
	}

	if d.traitsDictionary[superTrait] == nil {
		d.traitsDictionary[superTrait] = &traitsInformation{
			subTraits: []string{currentTrait},
		}

	} else if !slices.Contains(d.traitsDictionary[superTrait].subTraits, currentTrait) {
		d.traitsDictionary[superTrait].subTraits = append(d.traitsDictionary[superTrait].subTraits, currentTrait)
	}

	return nil
}

// AddRelation adds the relation with specific roles.
// Existing relation is overridden.
func (d *Dictionary) AddRelation(trait string, roleTraitsMap map[string][]string) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if len(roleTraitsMap) == 0 {
		return errors.New("no role for relation, it is not a relation")
	} else if d.relationsDictionary == nil {
		d.relationsDictionary = make(map[string]*relationsInformation)
	}

	// test if roles are valid before insertion
	allTraits := make([]string, 0)
	for _, v := range roleTraitsMap {
		if len(v) != 0 {
			allTraits = append(allTraits, v...)
		}
	}

	if len(allTraits) == 0 {
		return errors.New("empty roles, invalid relation")
	}

	// insert traits if not already there
	if d.traitsDictionary == nil {
		d.traitsDictionary = make(map[string]*traitsInformation)
	}

	slices.Sort(allTraits)
	allTraits = slices.Compact(allTraits)

	for _, currentTrait := range allTraits {
		if _, found := d.traitsDictionary[currentTrait]; !found {
			d.traitsDictionary[currentTrait] = &traitsInformation{
				subTraits:   make([]string, 0),
				superTraits: make([]string, 0),
			}
		}
	}

	// insertion is ok, then
	if d.relationsDictionary[trait] == nil {
		value := relationsInformation{
			traitsRoles: make(map[string][]string),
		}

		d.relationsDictionary[trait] = &value
	} else if d.relationsDictionary[trait].traitsRoles == nil {
		d.relationsDictionary[trait].traitsRoles = make(map[string][]string)
	} else {
		delete(d.relationsDictionary, trait)
	}

	for k, v := range roleTraitsMap {
		if len(v) == 0 {
			continue
		}

		d.relationsDictionary[trait].traitsRoles[k] = slices.Clone(v)
	}

	return nil
}

// AddRelationWithSubject adds a relation with given trait, and specific possible subject traits.
// If relation did exist before, it is overriden
func (d *Dictionary) AddRelationWithSubject(trait string, subjectTraits []string) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if len(subjectTraits) == 0 {
		return errors.New("no subject")
	}

	subjectMap := map[string][]string{SUBJECT_ROLE: subjectTraits}
	return d.AddRelation(trait, subjectMap)
}

// AddRelationWithObject adds a relation with specific subject and object.
// If relation existed before, it is overridden
func (d *Dictionary) AddRelationWithObject(trait string, subjectTraits []string, objectTraits []string) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if len(subjectTraits) == 0 {
		return errors.New("no subject")
	} else if len(objectTraits) == 0 {
		return errors.New("no object")
	}

	rolesMap := map[string][]string{SUBJECT_ROLE: subjectTraits, OBJECT_ROLE: objectTraits}
	return d.AddRelation(trait, rolesMap)
}

// SetRelationRole adds the relation if not already there, and sets the role for the values
func (d *Dictionary) SetRelationRole(trait string, role string, values []string) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if len(values) == 0 {
		return errors.New("no value for role")
	}

	rolesMap := map[string][]string{role: values}
	return d.AddRelation(trait, rolesMap)
}

// AddRelationLink lins a subrelation to a relation.
// For instance, couple(x,y) is a subrelation of knows(x,y)
// Both relations should exist before insertion (to be sure both are valid)
func (d *Dictionary) AddRelationLink(subRelation, superRelation string) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if subRelation == superRelation {
		return errors.New("invalid same values link")
	} else if d.relationsDictionary == nil {
		return errors.New("nil relations")
	} else if _, foundSub := d.relationsDictionary[subRelation]; !foundSub {
		return errors.New("missing relation")
	} else if _, foundSup := d.relationsDictionary[superRelation]; !foundSup {
		return errors.New("missing relation")
	}

	if d.relationsDictionary[subRelation] == nil {
		d.relationsDictionary[subRelation] = &relationsInformation{
			superRelations: make([]string, 0),
		}
	}

	if d.relationsDictionary[superRelation] == nil {
		d.relationsDictionary[superRelation] = &relationsInformation{
			subRelations: make([]string, 0),
		}
	}

	if !slices.Contains(d.relationsDictionary[superRelation].subRelations, subRelation) {
		d.relationsDictionary[superRelation].subRelations = append(d.relationsDictionary[superRelation].subRelations, subRelation)
	}

	if !slices.Contains(d.relationsDictionary[subRelation].superRelations, superRelation) {
		d.relationsDictionary[subRelation].superRelations = append(d.relationsDictionary[subRelation].superRelations, superRelation)
	}

	return nil
}

// DirectSubTraits returns the sorted slice of all subtraits of parameter
// If d is nil, or has no value for that trait, it returns nil.
// It trait exists in the dictionary, but with no link, it returns empty
// Otherwise, it returns the traits that extends parameter
func (d *Dictionary) DirectSubTraits(trait string) []string {
	if d == nil || d.traitsDictionary == nil {
		return nil
	}

	information, found := d.traitsDictionary[trait]
	if !found {
		return nil
	} else if information == nil {
		return []string{}
	}

	size := len(information.subTraits)
	if len(information.subTraits) == 0 {
		return []string{}
	}

	result := make([]string, size)
	copy(result, information.subTraits)
	slices.Sort(result)

	return result
}

// DirectSuperTraits returns the sorted slice of all supertraits of parameter
// If d is nil, or has no value for that trait, it returns nil.
// It trait exists in the dictionary, but with no link, it returns empty
// Otherwise, it returns the traits that are extensions of parameter
func (d *Dictionary) DirectSuperTraits(trait string) []string {
	if d == nil || d.traitsDictionary == nil {
		return nil
	}

	information, found := d.traitsDictionary[trait]
	if !found {
		return nil
	} else if information == nil {
		return []string{}
	}

	size := len(information.superTraits)
	if len(information.superTraits) == 0 {
		return []string{}
	}

	result := make([]string, size)
	copy(result, information.superTraits)
	slices.Sort(result)

	return result
}

// GetDirectSubRelations returns the subrelations, if any, for a given relation.
// It returns nil for nil or no relation, empty if the relation exists with no subclass,
// the sorted values otherwise
func (d *Dictionary) GetDirectSubRelations(relation string) []string {
	if d == nil {
		return nil
	} else if d.relationsDictionary == nil {
		return nil
	}

	if value, found := d.relationsDictionary[relation]; !found {
		return nil
	} else if value == nil || len(value.subRelations) == 0 {
		return []string{}
	} else {
		result := slices.Clone(value.subRelations)
		slices.Sort(result)
		return result
	}
}

// GetDirectSuperRelations returns the super relations of a relation.
// If returns empty if the relation exists but with no super relation, the values if any.
// Otherwise, it returns nil
func (d *Dictionary) GetDirectSuperRelations(relation string) []string {
	if d == nil {
		return nil
	} else if d.relationsDictionary == nil {
		return nil
	}

	if value, found := d.relationsDictionary[relation]; !found {
		return nil
	} else if value == nil || len(value.superRelations) == 0 {
		return []string{}
	} else {
		result := slices.Clone(value.superRelations)
		slices.Sort(result)
		return result
	}
}

// GetRelationRoles returns the roles and accepted traits per name.
// It returns nil for nil dictionary, empty for no match, and sorted values otherwise (per role)
func (d *Dictionary) GetRelationRoles(trait string) map[string][]string {
	if d == nil {
		return nil
	} else if d.relationsDictionary == nil {
		return make(map[string][]string)
	} else if d.relationsDictionary[trait] == nil {
		return make(map[string][]string)
	}

	value := d.relationsDictionary[trait].traitsRoles
	result := make(map[string][]string)

	for k, v := range value {
		traits := slices.Clone(v)
		slices.Sort(traits)
		result[k] = traits
	}

	return result
}

// Merge adds all the values from other to current dictionary
func (d *Dictionary) Merge(other Dictionary) error {
	if d == nil {
		return errors.New("nil dictionary")
	} else if d.relationsDictionary == nil {
		d.relationsDictionary = make(map[string]*relationsInformation)
	}

	for k, v := range other.relationsDictionary {
		d.relationsDictionary[k] = appendRelationsInformation(d.relationsDictionary[k], v)
	}

	if len(d.relationsDictionary) == 0 {
		d.relationsDictionary = nil
	}

	if d.traitsDictionary == nil {
		d.traitsDictionary = make(map[string]*traitsInformation)
	}

	for k, v := range other.traitsDictionary {
		d.traitsDictionary[k] = appendTraitsInformation(d.traitsDictionary[k], v)
	}

	if len(d.traitsDictionary) == 0 {
		d.traitsDictionary = nil
	}

	return nil
}
