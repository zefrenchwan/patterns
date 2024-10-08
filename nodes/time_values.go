package nodes

import (
	"errors"
	"slices"
	"time"
)

// TimeValues represents attributes with time dependent values.
// Keys are attributes name, values are the values per period.
type TimeValues map[string]map[string]*Period

// ActiveTimeValues represents an object with time dependent values and an activity.
// The object may be active during a given period (for instance, a person during its lifetime).
// Then, when asked for values or attributes, it reacts like a TimeValues but gets the
// intersection of activity and values periods.
// For instance, an active time values since now and with a first name set to "x"
// will have a first name period equals to [now, +oo[.
// Another example would be: "x" has been a student between year -3 and year.
// So, the period of activity should be [year -3, year] and values are naturally bounded with this period.
// A different way to understand it is:
// all values are set during periods, and once asked, the periods are the intersection
// with the period of activity
type ActiveTimeValues struct {
	// periodOfActivity is the period during all values make sense
	periodOfActivity Period
	// values are time dependent values, implicitely bounded by the period of activity
	values TimeValues
}

// NewTimeValues returns a new empty TimeValues
func NewTimeValues() TimeValues {
	return make(map[string]map[string]*Period)
}

// NewActiveTimeValues returns a new empty ActiveTimeValues, active forever.
// This default activity makes values acting like an instance of Timevalues
func NewActiveTimeValues() ActiveTimeValues {
	return ActiveTimeValues{
		periodOfActivity: NewFullPeriod(),
		values:           NewTimeValues(),
	}
}

// ContainsAttribute returns true if receiver is not nil and it contains a non nil entry with that key
func (t TimeValues) ContainsAttribute(attr string) bool {
	switch value, found := t[attr]; found {
	case true:
		return value != nil
	default:
		return false
	}
}

// ContainsAttribute returns true if receiver is not nil and it contains a non nil entry with that key
func (a *ActiveTimeValues) ContainsAttribute(attr string) bool {
	if a == nil {
		return false
	} else if a.values == nil {
		return false
	}

	return a.values.ContainsAttribute(attr)
}

// Attributes returns the sorted slice of all attributes
func (t TimeValues) Attributes() []string {
	if t == nil {
		return nil
	}

	result := make([]string, len(t))
	index := 0
	for k := range t {
		result[index] = k
		index++
	}

	slices.Sort(result)
	return result
}

// Attributes returns the sorted slice of all attributes.
// Nil receiver returns nil
func (a *ActiveTimeValues) Attributes() []string {
	if a == nil {
		return nil
	} else if a.values == nil {
		return nil
	}

	return a.values.Attributes()
}

// SetValue sets a value for an attribute, for the full period.
func (t TimeValues) SetValue(attribute string, value string) error {
	var matchingAttributeMap map[string]*Period

	// find matching map for this attribute, if any
	if t == nil {
		return errors.New("nil values")
	} else if value, found := t[attribute]; !found {
		// not found, allocate
		t[attribute] = make(map[string]*Period)
		matchingAttributeMap = t[attribute]
	} else {
		matchingAttributeMap = value
	}

	// clean the map
	for k := range matchingAttributeMap {
		delete(matchingAttributeMap, k)
	}

	// add value -> full
	period := NewFullPeriod()
	matchingAttributeMap[value] = &period

	// no error
	return nil
}

// SetValue sets a value for an attribute, for the full period.
func (a *ActiveTimeValues) SetValue(attribute string, value string) error {
	if a == nil {
		return errors.New("nil active value")
	} else if a.values == nil {
		a.values = NewTimeValues()
	}

	return a.values.SetValue(attribute, value)
}

// AddValue sets the value of an attribute during a given period.
// It updates the periods of the other values (for the same attribute) accordingly.
func (t TimeValues) AddValue(attribute string, value string, validity Period) error {
	// nil should return an error, empty period should change nothing
	if t == nil {
		return errors.New("nil values")
	} else if validity.IsEmptyPeriod() {
		return nil
	}

	// find matching attribute map if any
	var matchingAttributeMap map[string]*Period
	if value, found := t[attribute]; !found {
		// not found, allocate
		t[attribute] = make(map[string]*Period)
		matchingAttributeMap = t[attribute]
	} else {
		matchingAttributeMap = value
	}

	// for each attribute value different than parameter, get the intersection with the validity
	for valueForAttribute, matchingPeriod := range matchingAttributeMap {
		// will change value later
		if valueForAttribute == value {
			continue
		}

		// remove the period for the other attribute.
		// And if it is empty, value should be removed
		matchingPeriod.Remove(validity)
		if matchingPeriod.IsEmptyPeriod() {
			delete(matchingAttributeMap, value)
		}
	}

	// and set the value
	if matchingPeriod, found := matchingAttributeMap[value]; found {
		matchingPeriod.Add(validity)
	} else {
		copyOfPeriod := NewPeriodCopy(validity)
		matchingAttributeMap[value] = &copyOfPeriod
	}

	return nil
}

// AddValue sets the value of an attribute during a given period.
// It updates the periods of the other values (for the same attribute) accordingly.
// It returns an error if receiver is nil
func (a *ActiveTimeValues) AddValue(attribute string, value string, validity Period) error {
	if a == nil {
		return errors.New("nil active value")
	} else if a.values == nil {
		a.values = NewTimeValues()
	}

	return a.values.AddValue(attribute, value, validity)
}

// SetPeriodForValue sets the value and the period for that attribute.
func (t TimeValues) SetPeriodForValue(attribute string, value string, period Period) error {
	if t == nil {
		return errors.New("nil values")
	} else if !period.IsEmptyPeriod() {
		copyPeriod := NewPeriodCopy(period)

		// no other value, just insert
		if t[attribute] == nil {
			t[attribute] = make(map[string]*Period)
			t[attribute][value] = &copyPeriod
			return nil
		} else if len(t[attribute]) == 0 {
			t[attribute][value] = &copyPeriod
			return nil
		}

		// Possible other values, clean others and then insert.
		// remove period to other values and clean accordingly
		valuesToClean := make([]string, 0)
		for otherValue, otherPeriod := range t[attribute] {
			otherPeriod.Remove(period)
			if otherPeriod.IsEmptyPeriod() {
				valuesToClean = append(valuesToClean, otherValue)
			}
		}

		for _, valueToClean := range valuesToClean {
			delete(t[attribute], valueToClean)
		}

		// and finally, insert it
		t[attribute][value] = &copyPeriod
	} else if _, found := t[attribute]; found {
		// period is empty, delete the attribute
		delete(t[attribute], value)
	}

	return nil
}

// SetPeriodForValue sets the value and the period for that attribute.
func (a *ActiveTimeValues) SetPeriodForValue(attribute string, value string, period Period) error {
	if a == nil {
		return errors.New("nil active value")
	} else if a.values == nil {
		return nil
	}

	return a.SetPeriodForValue(attribute, value, period)
}

// RemovePeriodForAttribute just removes period, no matter the value, for that attribute
func (t TimeValues) RemovePeriodForAttribute(attribute string, period Period) error {
	if t == nil {
		return errors.New("nil values")
	} else if period.IsEmptyPeriod() {
		return nil
	}

	if t[attribute] == nil {
		return nil
	}

	elementsToDelete := make([]string, 0)
	for value, valuePeriod := range t[attribute] {
		valuePeriod.Remove(period)
		if valuePeriod.IsEmptyPeriod() {
			elementsToDelete = append(elementsToDelete, value)
		}
	}

	for _, elementToDelete := range elementsToDelete {
		delete(t[attribute], elementToDelete)
	}

	return nil
}

// RemovePeriodForAttribute just removes period, no matter the value, for that attribute
func (a *ActiveTimeValues) RemovePeriodForAttribute(attribute string, period Period) error {
	if a == nil {
		return errors.New("nil active value")
	} else if a.values == nil {
		return nil
	}

	return a.RemovePeriodForAttribute(attribute, period)
}

// ValuesForAttribute returns the values for an attribute as a sorted slice
func (t TimeValues) ValuesForAttribute(attribute string) ([]string, error) {
	if t == nil {
		return nil, errors.New("nil values")
	} else if attributeValues, found := t[attribute]; !found {
		return nil, nil
	} else if len(attributeValues) == 0 {
		return nil, nil
	} else {
		result := make([]string, len(attributeValues))

		index := 0
		for k := range attributeValues {
			result[index] = k
			index++
		}

		slices.Sort(result)
		return result, nil
	}
}

// ValuesForAttribute returns the values for an attribute as a sorted slice during the activity of the object.
// For instance, if activity is [now, +oo[ and values are set for ] -oo, now - 1 day] , then it returns nil
func (a *ActiveTimeValues) ValuesForAttribute(attribute string) ([]string, error) {
	if a == nil {
		return nil, errors.New("nil active value")
	}

	allValues, errValues := a.PeriodsForAttribute(attribute)
	if errValues != nil {
		return nil, errValues
	}

	if len(allValues) == 0 {
		return nil, nil
	}

	result := make([]string, len(allValues))
	index := 0
	for name := range allValues {
		result[index] = name
		index++
	}

	return result, nil
}

// TimeValuesForAttribute returns, for each value of the attribute, the matching time intervals
func (t TimeValues) TimeValuesForAttribute(attribute string) (map[string][]Interval[time.Time], error) {
	if t == nil {
		return nil, errors.New("nil instance")
	} else if attributeValues, found := t[attribute]; !found {
		return nil, nil
	} else if len(attributeValues) == 0 {
		return nil, nil
	} else {
		result := make(map[string][]Interval[time.Time])

		for value, period := range attributeValues {
			// should not happen
			if period == nil {
				continue
			}

			result[value] = period.AsIntervals()
		}

		return result, nil
	}
}

// TimeValuesForAttribute returns, for each value of the attribute, the matching time intervals
func (a *ActiveTimeValues) TimeValuesForAttribute(attribute string) (map[string][]Interval[time.Time], error) {
	if a == nil {
		return nil, errors.New("nil active value")
	} else if a.values == nil {
		return nil, errors.New("nil values")
	}

	valuePeriodMap, errPeriods := a.PeriodsForAttribute(attribute)
	if errPeriods != nil {
		return nil, errPeriods
	}

	result := make(map[string][]Interval[time.Time])
	for attrValue, period := range valuePeriodMap {
		intervals := period.AsIntervals()
		if len(intervals) == 0 {
			continue
		} else if intervals[0].IsEmpty() {
			continue
		}

		result[attrValue] = intervals
	}

	return result, nil
}

// PeriodsForAttribute returns, for an attribute, each value and its active period.
func (a TimeValues) PeriodsForAttribute(attribute string) (map[string]Period, error) {
	if a == nil {
		return nil, errors.New("nil values")
	} else if attributeValues, found := a[attribute]; !found {
		return nil, nil
	} else if len(attributeValues) == 0 {
		return nil, nil
	} else {
		result := make(map[string]Period)

		for value, period := range attributeValues {
			// should not happen
			if period == nil {
				continue
			}

			copyPeriod := NewPeriodCopy(*period)
			result[value] = copyPeriod
		}

		return result, nil
	}
}

// PeriodsForAttribute returns, for an attribute, each value and its active period.
// Its active period is the intersection of the matching period AND the current activity
func (a *ActiveTimeValues) PeriodsForAttribute(attribute string) (map[string]Period, error) {
	if a == nil {
		return nil, errors.New("nil active value")
	} else if a.values == nil {
		return nil, errors.New("nil values")
	} else if attributeValues, found := a.values[attribute]; !found {
		return nil, nil
	} else if len(attributeValues) == 0 {
		return nil, nil
	} else {
		result := make(map[string]Period)

		for value, period := range attributeValues {
			// should not happen
			if period == nil {
				continue
			}

			// remember that period does not return copy but does its change in the receiver
			copyPeriod := NewPeriodCopy(*period)
			copyPeriod.Intersection(a.periodOfActivity)
			if copyPeriod.IsEmptyPeriod() {
				continue
			}

			result[value] = copyPeriod
		}

		return result, nil
	}
}

// AddActivity sets p as active
func (a *ActiveTimeValues) AddActivity(p Period) error {
	if a == nil {
		return errors.New("nil active value")
	}

	a.periodOfActivity.Add(p)
	return nil
}

// RemoveActivity flags p as inactive
func (a *ActiveTimeValues) RemoveActivity(p Period) error {
	if a == nil {
		return errors.New("nil active value")
	}

	a.periodOfActivity.Remove(p)
	return nil
}

// SetActivity sets the period of activity no matter previous value
func (a *ActiveTimeValues) SetActivity(p Period) error {
	if a == nil {
		return errors.New("nil active value")
	}

	a.periodOfActivity = NewPeriodCopy(p)
	return nil
}

// IsEmpty returns true if the receiver is never active
func (a *ActiveTimeValues) IsEmpty() bool {
	return a == nil || a.periodOfActivity.IsEmptyPeriod()
}

// IsActiveDuring returns true if p and the active period have at least a common point
func (a *ActiveTimeValues) IsActiveDuring(p Period) bool {
	if a == nil {
		return false
	}

	copyPeriod := NewPeriodCopy(p)
	copyPeriod.Intersection(a.periodOfActivity)
	return !copyPeriod.IsEmptyPeriod()
}
