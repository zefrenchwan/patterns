package patterns

import (
	"errors"
	"slices"
	"time"
)

// default comparator for time operations.
// Golang does not allow const value, but it makes no sense to change this value.
// Once we define it, we want to hide details of intervals and periods creation
var periodComparator = NewTimeComparator()

// NewLeftInfiniteTimeInterval returns the interval ]-oo, maxTime)
func NewLeftInfiniteTimeInterval(maxTime time.Time, maxIncluded bool) Interval[time.Time] {
	return periodComparator.NewLeftInfiniteInterval(maxTime, maxIncluded)
}

// NewRightInfiniteTimeInterval returns (minTime, +oo [
func NewRightInfiniteTimeInterval(minTime time.Time, minIncluded bool) Interval[time.Time] {
	return periodComparator.NewRightInfiniteInterval(minTime, minIncluded)
}

// NewFiniteTimeInterval returns the interval (minTime, maxTime).
// If the interval would be empty, it returns an error
func NewFiniteTimeInterval(minTime, maxTime time.Time, minIn, maxIn bool) (Interval[time.Time], error) {
	return periodComparator.NewFiniteInterval(minTime, maxTime, minIn, maxIn)
}

// TimeIntervalsCompare is a shortcut to compare intervals of time without explicit use of comparator.
// Contract is the same comparator.CompareInterval
func TimeIntervalsCompare(a, b Interval[time.Time]) int {
	return periodComparator.CompareInterval(a, b)
}

// Period is a set of moments, a moment being a time interval.
// It is neither a duration, nor a set of duration.
// For instance, a person lived in a country from 1999 to 2021 and since 2023.
// Implementation may be extended to any type of elements.
type Period struct {
	// elements are the set of separated intervals of time.
	// Invariants are:
	// * if empty or just containing empty, the period is empty
	// * if period is not empty, it contains separated intervals of time
	elements []Interval[time.Time]
}

// NewPeriod returns a period that contains base exactly
func NewPeriod(base Interval[time.Time]) Period {
	var period Period
	period.elements = []Interval[time.Time]{base}
	return period
}

// NewPeriodCopy just copies its parameter.
// Intervals are immutable so they are not copied
func NewPeriodCopy(p Period) Period {
	var period Period
	period.elements = make([]Interval[time.Time], len(p.elements))
	copy(period.elements, p.elements)
	return period
}

// NewEmptyPeriod returns an empty period
func NewEmptyPeriod() Period {
	var period Period
	period.elements = make([]Interval[time.Time], 0)
	return period
}

// NewFullPeriod returns a full period
func NewFullPeriod() Period {
	var period Period
	period.elements = []Interval[time.Time]{NewTimeComparator().NewFullInterval()}
	return period
}

// IsEmptyPeriod returns true for an empty period or nil (assumed then to be empty)
func (p *Period) IsEmptyPeriod() bool {
	return p == nil || len(p.elements) == 0 || p.elements[0].IsEmpty()
}

// AsIntervals returns the period as a sorted set of separated intervals
func (p *Period) AsIntervals() []Interval[time.Time] {
	if p == nil {
		return nil
	}

	result := make([]Interval[time.Time], len(p.elements))
	copy(result, p.elements)
	slices.SortFunc(result, periodComparator.CompareInterval)
	return result
}

// AddInterval adds an interval to the period, but ensures invariant that all elements are separated
func (p *Period) AddInterval(i Interval[time.Time]) error {
	if p == nil {
		return errors.New("nil period")
	} else if len(p.elements) == 0 {
		p.elements = []Interval[time.Time]{i}
	} else if i.IsEmpty() || p.elements[0].IsEmpty() {
		return nil
	} else if p.elements[0].IsFull() {
		// already full
		return nil
	}

	// period is not empty, interval to add is not empty
	p.elements = periodComparator.Union(i, p.elements...)

	return nil
}

// Add is the union of periods.
// It returns an error if the receiver is nil
func (p *Period) Add(other Period) error {
	if p == nil {
		return errors.New("nil period")
	} else if other.IsEmptyPeriod() {
		return nil
	}

	currentSize := len(p.elements)
	otherSize := len(other.elements)
	unionOfElements := make([]Interval[time.Time], currentSize+otherSize)

	for index := 0; index < currentSize; index++ {
		unionOfElements[index] = p.elements[index]
	}

	for index := 0; index < otherSize; index++ {
		unionOfElements[currentSize+index] = other.elements[index]
	}

	if len(unionOfElements) >= 2 {
		p.elements = periodComparator.Union(unionOfElements[0], unionOfElements[1:]...)
	} else {
		p.elements = unionOfElements
	}

	return nil
}

// Intersection keeps intervals both in p and other.
// Formally, if p = union of p_i and other = union of o_j,
// then result is union over i and j of (p_i inter o_j )
func (p *Period) Intersection(other Period) {
	if p.IsEmptyPeriod() || other.IsEmptyPeriod() {
		return
	}

	var union []Interval[time.Time]
	for _, currentInterval := range p.elements {
		for _, otherInterval := range other.elements {
			intersection := periodComparator.Intersection(currentInterval, otherInterval)
			if !intersection.IsEmpty() {
				union = append(union, intersection)
			}
		}
	}

	if len(union) <= 1 {
		p.elements = union
	} else {
		p.elements = periodComparator.Union(union[0], union[1:]...)
	}
}

// Remove starts with p and remove all the intervals from other.
// Formally, let p_i be the content of p and o_j be the content of other
// New content for p is Union over i of (intersections over j ( p_i minus o_j ))
func (p *Period) Remove(other Period) {
	if p.IsEmptyPeriod() || other.IsEmptyPeriod() {
		return
	}

	var newElements []Interval[time.Time]
	for _, interval := range p.elements {

		var intersections []Interval[time.Time]
		for _, otherInterval := range other.elements {
			differences := periodComparator.Remove(interval, otherInterval)
			if len(differences) != 0 {
				intersections = append(intersections, differences...)
			}
		}

		// For a given interval, that is, for a given i,
		// intersection is the intersection over j of all (p_i minus o_j)
		intersection := periodComparator.Intersection(interval, intersections...)
		if !intersection.IsEmpty() {
			newElements = append(newElements, intersection)
		}
	}

	switch len(newElements) {
	case 0:
		p.elements = []Interval[time.Time]{periodComparator.NewEmptyInterval()}
	case 1:
		p.elements = newElements
	default:
		p.elements = periodComparator.Union(newElements[0], newElements[1:]...)
	}
}

// Complement finds the period such as they are partition of the full space.
// Formally, given p = union of p_i, we want other = union of o_j such as
// o_j inter p_i is empty, and (union of o_j) union (union of p_i) is full.
func (p *Period) Complement() {
	if p.IsEmptyPeriod() {
		p.elements = []Interval[time.Time]{periodComparator.NewFullInterval()}
		return
	}

	// then, len of p.elements is at least 1 with no empty.
	result := make([]Interval[time.Time], 0)

	// first, full - union of p_i = ((full - p_1) - p_2) - ...
	// So we start with result = full - p_1, and we add p_{i+1} from current result
	for _, complement := range periodComparator.Complement(p.elements[0]) {
		if !complement.IsEmpty() {
			result = append(result, complement)
		} else if complement.IsFull() {
			p.elements = make([]Interval[time.Time], 0)
			return
		}
	}

	// Then, if result is currently A_1 union A_2 etc...
	// result - p_{i+1} = union over j of (A_j - p_{i+1})
	for _, element := range p.elements[1:] {
		// result may be empty, or containing empty. No need to go on then
		if len(result) == 0 || result[0].IsEmpty() {
			p.elements = make([]Interval[time.Time], 0)
			return
		}

		// Otherwise, result is not empty, we may go on.
		// At this point, result is then union of separated intervals, A_1 ... A_m
		// And we want the union of A_j - element
		unions := make([]Interval[time.Time], 0)
		for _, value := range result {
			for _, remaining := range periodComparator.Remove(value, element) {
				if !remaining.IsEmpty() {
					unions = append(unions, remaining)
				}
			}
		}

		// unions is the union of each A_j - elements, but its items may not be separated.
		// To ensure we group all separated elements, we take the union of what is left.
		if len(unions) >= 2 {
			unions = periodComparator.Union(unions[0], unions[1:]...)
		}

		// and finally, we go on with recursive removal formula
		result = unions
	}

	p.elements = result
}
