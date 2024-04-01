package patterns

import (
	"errors"
)

// TypedComparator defines a compare function over a type
type TypedComparator[T any] struct {
	comparator func(T, T) int
}

// NewTypedComparator returns an interval manager based on a compare function.
// Contract for compareFn(a, b) is:
// * if a < b, return a negative value
// * if a > b, return a positive value
// * if a == b, return 0
// * no error returned
// * comparison should be quick
func NewTypedComparator[T any](compareFn func(T, T) int) TypedComparator[T] {
	var result TypedComparator[T]
	result.comparator = compareFn
	return result
}

// Compare decorates the comparator function
func (t TypedComparator[T]) Compare(a, b T) int {
	return t.comparator(a, b)
}

// Min returns the min of values
func (t TypedComparator[T]) Min(a T, v ...T) T {
	min := a
	for _, other := range v {
		if t.Compare(other, min) < 0 {
			min = other
		}
	}

	return min
}

// Interval is the definition of intervals based on the comparator.
type Interval[T any] struct {
	// true for empty interval
	empty bool
	// true if interval is not left bounded
	minInfinite bool
	// min of the interval, if not minInfinite
	min T
	// if not minInfinite, true if the min is the interval, false otherwise
	minIncluded bool
	// true if interval is not right bounded, false otherwise
	maxInfinite bool
	// max of the interval, if not maxInfinite
	max T
	// if not maxInfinite, true if the max is the interval, false otherwise
	maxIncluded bool
}

// IsFull returns true for an unbounded interval
func (i Interval[T]) IsFull() bool {
	return i.maxInfinite && i.minInfinite
}

// IsEmpty is true for an empty interval, false otherwise
func (i Interval[T]) IsEmpty() bool {
	return i.empty
}

// IsCompact returns true for a bounded interval with both values in it (hence, not empty)
func (i Interval[T]) IsCompact() bool {
	return !i.minInfinite && !i.maxInfinite && i.minIncluded && i.maxIncluded && !i.empty
}

// NewEmptyInterval returns a new empty interval
func (t TypedComparator[T]) NewEmptyInterval() Interval[T] {
	var result Interval[T]
	result.empty = true
	return result
}

// NewFullInterval returns a full interval
func (t TypedComparator[T]) NewFullInterval() Interval[T] {
	var result Interval[T]
	result.minInfinite = true
	result.maxInfinite = true
	return result
}

// NewLeftInfiniteInterval returns an interval ] -oo, rightValue )
func (t TypedComparator[T]) NewLeftInfiniteInterval(rightValue T, rightIncluded bool) Interval[T] {
	var result Interval[T]
	result.max = rightValue
	result.maxIncluded = rightIncluded
	result.minInfinite = true
	return result
}

// NewRightInfiniteInterval returns an interval ( leftValue, +oo [
func (t TypedComparator[T]) NewRightInfiniteInterval(leftValue T, leftIncluded bool) Interval[T] {
	var result Interval[T]
	result.min = leftValue
	result.minIncluded = leftIncluded
	result.maxInfinite = true
	return result
}

// NewFiniteInterval returns a finite interval or an error if interval would be empty
func (t TypedComparator[T]) NewFiniteInterval(left, right T, leftIn, rightIn bool) (Interval[T], error) {
	var result Interval[T]
	comparison := t.Compare(left, right)
	if comparison > 0 || (comparison == 0 && !(leftIn && rightIn)) {
		return result, errors.New("interval parameters would make empty interval")
	}

	result.max = right
	result.min = left
	result.minIncluded = leftIn
	result.maxIncluded = rightIn

	return result, nil
}

// CompareInterval is an order based on the lexicographic order.
// Same sets are equals (return 0).
func (t TypedComparator[T]) CompareInterval(a, b Interval[T]) int {
	// deal with empty or full intervals
	switch {
	case a.IsEmpty():
		if b.IsEmpty() {
			return 0
		}

		return 1
	case b.IsEmpty():
		return -1
	case a.IsFull():
		if b.minInfinite && b.maxInfinite {
			return 0
		}

		return 1
	case b.IsFull():
		return -1
	}

	// deal with left boundaries.
	// If both left infinite, they are equals
	switch {
	case a.minInfinite && !b.minInfinite:
		return -1
	case b.minInfinite && !a.minInfinite:
		return 1
	case !a.minInfinite && !b.minInfinite:
		minCompare := t.Compare(a.min, b.min)
		if minCompare != 0 {
			return minCompare
		} else if a.minIncluded && !b.minIncluded {
			return 1
		} else if !a.minIncluded && b.minIncluded {
			return -1
		}
	}

	// left boundaries are equals, result is now based on right boundaries
	switch {
	case a.maxInfinite && b.maxInfinite:
		return 0
	case a.maxInfinite:
		return 1
	case b.maxInfinite:
		return -1
	}

	comparison := t.Compare(a.max, b.max)
	if comparison != 0 {
		return comparison
	} else if a.maxIncluded == b.maxIncluded {
		return 0
	} else if a.maxIncluded {
		return 1
	} else {
		return -1
	}

}

// Complement returns the complement of the interval.
// It may be a single set (empty => full, full => empty, etc) or two (for finite intervals)
func (t TypedComparator[T]) Complement(i Interval[T]) []Interval[T] {
	var result Interval[T]
	switch {
	case i.empty:
		result.maxInfinite = true
		result.minInfinite = true
		return []Interval[T]{result}
	case i.minInfinite && i.maxInfinite:
		result.empty = true
		return []Interval[T]{result}
	case i.minInfinite:
		result.maxInfinite = true
		result.min = i.max
		result.minIncluded = !i.maxIncluded
		return []Interval[T]{result}
	case i.maxInfinite:
		result.minInfinite = true
		result.max = i.min
		result.maxIncluded = !i.minIncluded
		return []Interval[T]{result}
	}

	// remaining case is (a,b) with finite values
	// Then, result is ]-oo, a( and )b, +oo[
	var otherResult Interval[T]
	result.minInfinite = true
	result.max = i.min
	result.maxIncluded = !i.minIncluded
	otherResult.maxInfinite = true
	otherResult.minIncluded = !i.maxIncluded
	otherResult.min = i.max

	return []Interval[T]{result, otherResult}
}

// Intersection returns the intersection of base and others
func (t TypedComparator[T]) Intersection(base Interval[T], others ...Interval[T]) Interval[T] {
	current := base
	for _, other := range others {
		// perform the intersection of current and other
		if other.IsEmpty() || current.IsEmpty() {
			current = t.NewEmptyInterval()
			break
		} else if current.IsFull() {
			current = other
			continue
		} else if other.IsFull() {
			continue
		}

		var resMin, resMax T
		var resInfiniteMin, resInfiniteMax bool
		var resInMin, resInMax bool

		// find left borders
		if current.minInfinite && other.minInfinite {
			resInfiniteMin = true
		} else if current.minInfinite {
			resMin = other.min
			resInMin = other.minIncluded
		} else if other.minInfinite {
			resMin = current.min
			resInMin = current.minIncluded
		} else if leftCompare := t.Compare(current.min, other.min); leftCompare == 0 {
			resMin = current.min
			resInMin = current.minIncluded && other.minIncluded
		} else if leftCompare < 0 {
			resMin = other.min
			resInMin = other.minIncluded
		} else {
			resMin = current.min
			resInMin = current.minIncluded
		}

		// find right borders
		if current.maxInfinite && other.maxInfinite {
			resInfiniteMax = true
		} else if current.maxInfinite {
			resMax = other.max
			resInMax = other.maxIncluded
		} else if other.maxInfinite {
			resMax = current.max
			resInMax = current.maxIncluded
		} else if rightCompare := t.Compare(current.max, other.max); rightCompare == 0 {
			resMax = current.max
			resInMax = current.maxIncluded && other.maxIncluded
		} else if rightCompare < 0 {
			resMax = current.max
			resInMax = current.maxIncluded
		} else {
			resMax = other.max
			resInMax = other.maxIncluded
		}

		// make interval if possible.
		// It not, it means that result is empty, and stop.
		// If it is possible, it is the new current
		current = Interval[T]{
			empty:       false,
			min:         resMin,
			max:         resMax,
			minIncluded: resInMin,
			maxIncluded: resInMax,
			minInfinite: resInfiniteMin,
			maxInfinite: resInfiniteMax,
		}

		if !current.maxInfinite && !current.minInfinite {
			compare := t.Compare(resMin, resMax)
			if compare > 0 {
				current.empty = true
			} else if compare == 0 {
				current.empty = !(resInMax && resInMin)
			}
		}
	}

	return current
}

// areSeparated returns true if sets are non joinable.
// Intervals are joinable if their union form a set and not two. Formally:
// their intersection is not empty
// OR if one is ..., b) and the other is (b, ...) and b belongs in one of them.
// Joinable sets may be joined to create their union.
func (t TypedComparator[T]) areSeparated(a, b Interval[T]) bool {
	if a.IsEmpty() || b.IsEmpty() {
		return false
	}

	if a.minInfinite && b.minInfinite {
		return false
	} else if a.maxInfinite && b.maxInfinite {
		return false
	} else if a.minInfinite {
		compare := t.Compare(a.max, b.min)
		if compare < 0 {
			return true
		} else if compare == 0 {
			return !a.maxIncluded && !b.minIncluded
		} else {
			return false
		}
	} else if b.minInfinite {
		compare := t.Compare(b.max, a.min)
		if compare < 0 {
			return true
		} else if compare == 0 {
			return !b.maxIncluded && !a.minIncluded
		} else {
			return false
		}
	} else if a.maxInfinite {
		compare := t.Compare(a.min, b.max)
		if compare > 0 {
			return true
		} else if compare == 0 {
			return !a.minIncluded && !b.maxIncluded
		} else {
			return false
		}
	} else if b.maxInfinite {
		compare := t.Compare(a.max, b.min)
		if compare < 0 {
			return true
		} else if compare == 0 {
			return !a.maxIncluded && !b.minIncluded
		} else {
			return false
		}
	}

	// both are finite
	compareMaxMin := t.Compare(a.max, b.min)
	if compareMaxMin < 0 {
		return true
	} else if compareMaxMin == 0 {
		return !a.maxIncluded && !b.minIncluded
	}

	compareMinMax := t.Compare(a.min, b.max)
	if compareMinMax > 0 {
		return true
	} else if compareMinMax == 0 {
		return !a.minIncluded && !b.maxIncluded
	}

	return false
}

// Union returns the union of intervals.
// Result is a set of intervals, all separated from each other.
// Special case: if all sets are empty, result is just one empty set
func (t TypedComparator[T]) Union(base Interval[T], others ...Interval[T]) []Interval[T] {
	result := []Interval[T]{base}
	if base.IsFull() {
		return result
	}

	// try to join other with all elements in result, update result with max separated intervals
	for _, other := range others {
		// special cases, no need to process
		if other.IsFull() {
			return []Interval[T]{t.NewFullInterval()}
		} else if other.IsEmpty() {
			continue
		}

		// result so far is the set of all intervals such as no more join is possible.
		// When we consider other, we may then group intervals together again.

		// otherJoinableWithCurrents is true when other is not separated with at least one current element
		otherJoinableWithCurrents := false
		// separatedIntervals is the set of all separated intervals
		separatedIntervals := make([]Interval[T], 0)
		// toJoin is the set of intervals to reduce to an unique one.
		// Once it is done, then the resulting set will go to unions and unions is ready
		toJoin := make([]Interval[T], 0)

		for _, current := range result {
			if current.IsFull() {
				return []Interval[T]{t.NewFullInterval()}
			}

			if t.areSeparated(current, other) {
				separatedIntervals = append(separatedIntervals, current)
			} else {
				toJoin = append(toJoin, current)
				otherJoinableWithCurrents = true
			}
		}

		// don't forget to add other in one list of elements, depending on whether it may be joined
		if otherJoinableWithCurrents {
			toJoin = append(toJoin, other)
		} else {
			separatedIntervals = append(separatedIntervals, other)
		}

		// no join is possible.
		// If so, we are done for this iteration
		if len(toJoin) == 0 {
			result = separatedIntervals
			continue
		}

		// the union of all elements in toJoin is the min of left borders and max of right borders
		var minRes, maxRes T
		var minInRes, maxInRes bool
		var minInfRes, maxInfRes bool

		for index, element := range toJoin {
			if index == 0 {
				minRes, maxRes = element.min, element.max
				minInRes, maxInRes = element.minIncluded, element.maxIncluded
				minInfRes, maxInfRes = element.minInfinite, element.maxInfinite
				continue
			}

			if element.minInfinite {
				minInfRes = true
			} else if !minInfRes {
				compare := t.Compare(minRes, element.min)
				if compare > 0 {
					minRes, minInRes = element.min, element.minIncluded
				} else if compare == 0 && !minInRes {
					minRes, minInRes = element.min, element.minIncluded
				}
			}

			if element.maxInfinite {
				maxInfRes = true
			} else if !maxInfRes {
				compare := t.Compare(maxRes, element.max)
				if compare < 0 {
					maxRes, maxInRes = element.max, element.maxIncluded
				} else if compare == 0 && !maxInRes {
					maxRes, maxInRes = element.max, element.maxIncluded
				}
			}
		}

		var resultingInterval Interval[T]
		resultingInterval.max = maxRes
		resultingInterval.min = minRes
		resultingInterval.maxIncluded = maxInRes
		resultingInterval.minIncluded = minInRes
		resultingInterval.maxInfinite = maxInfRes
		resultingInterval.minInfinite = minInfRes

		separatedIntervals = append(separatedIntervals, resultingInterval)
		result = separatedIntervals
	}

	if len(result) == 0 {
		result = []Interval[T]{t.NewEmptyInterval()}
	}

	return result
}

// Remove returns base - (union of elements) as a set of separated elements
func (t TypedComparator[T]) Remove(base Interval[T], elements ...Interval[T]) []Interval[T] {
	if len(elements) == 0 {
		return []Interval[T]{base}
	} else if base.IsEmpty() {
		return []Interval[T]{base}
	}

	// first, group all elements so that we start with separated elements
	var elementsToRemove []Interval[T]
	if len(elements) == 1 {
		elementsToRemove = []Interval[T]{elements[0]}
	} else {
		elementsToRemove = t.Union(elements[0], elements[1:]...)
	}

	result := []Interval[T]{base}
	newResult := make([]Interval[T], 0)
	for _, elementToRemove := range elementsToRemove {
		for _, current := range result {
			// deal with basic cases
			basicCase := false
			switch {
			case current.IsEmpty():
				// resulting interval is empty, no need to add it
				basicCase = true
			case current.IsFull():
				newResult = append(newResult, t.Complement(elementToRemove)...)
				basicCase = true
			case elementToRemove.IsFull():
				// no matter what, removing full returns empty
				result = []Interval[T]{t.NewEmptyInterval()}
				return result
			case elementToRemove.IsEmpty():
				newResult = append(newResult, current)
				basicCase = true
			}

			if basicCase {
				continue
			}

			// no full interval, no empty interval
			// For A, B two intervals:
			// A - B = A inter (full - B)
			// If full - B is one interval, just make inter
			// Else, full - B is A1 union A2, and A inter (A1 union A2) = (A inter A1) union (A inter A2)
			complement := t.Complement(elementToRemove)
			switch len(complement) {
			case 1:
				value := t.Intersection(current, complement[0])
				newResult = append(newResult, value)
			case 2:
				value1 := t.Intersection(current, complement[0])
				value2 := t.Intersection(current, complement[1])
				if value1.IsEmpty() && value2.IsEmpty() {
					continue
				} else if value1.IsEmpty() {
					newResult = append(newResult, value2)
				} else if value2.IsEmpty() {
					newResult = append(newResult, value1)
				} else {
					newResult = append(newResult, t.Union(value1, value2)...)
				}
			}
		}

		if len(newResult) == 1 {
			result = newResult
		} else if len(newResult) == 0 {
			// no more current element, result is empty
			return []Interval[T]{t.NewEmptyInterval()}
		} else {
			result = t.Union(newResult[0], newResult[1:]...)
		}
	}

	return result
}
