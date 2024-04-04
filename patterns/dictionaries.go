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

// Dictionary contains all the meta data about relations and traits
type Dictionary struct {
	// traitsDictionary links a name to the trait information
	traitsDictionary map[string]*traitsInformation
}

// NewDictionary returns an empty dictionary
func NewDictionary() Dictionary {
	return Dictionary{
		traitsDictionary: make(map[string]*traitsInformation),
	}
}

// AddTraitsLink links a trait to a super trait.
// Example of call: d.AddTraitsLink("cat", "animal")
// If traits did not already exist, they are
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

// DirectSubTraits returns the sorted slice of all subtraits of parameter
// If d is nil, or has no value for that trait, it returns nil.
// It trait exists in the dictionary, but with no link, it returns empty
// Otherwise, it returns the traits that extends parameter
func (d *Dictionary) DirectSubTraits(trait string) []string {
	if d == nil || d.traitsDictionary == nil {
		return nil
	}

	information := d.traitsDictionary[trait]
	if information == nil {
		return nil
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

	information := d.traitsDictionary[trait]
	if information == nil {
		return nil
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
