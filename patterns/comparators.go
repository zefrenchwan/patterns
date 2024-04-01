package patterns

import "time"

// NewIntComparator returns a tool to deal with intervals of int
func NewIntComparator() TypedComparator[int] {
	return NewTypedComparator(IntComparator)
}

// NewFloatComparator returns a tool to deal with intervals of float64
func NewFloatComparator() TypedComparator[int] {
	return NewTypedComparator(IntComparator)
}

// NewTimeComparator returns a tool to deal with intervals of time
func NewTimeComparator() TypedComparator[time.Time] {
	return NewTypedComparator(TimeComparator)
}

// IntComparator compares int.
// Use this function with intervals to create intervals of ints
func IntComparator(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}

	return 0
}

// FloatComparator compares flaots.
// Use this function with intervals to create intervals of floats
func FloatComparator(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}

	return 0
}

// TimeComparator compares time using their UTC values
// Use this function with intervals to create intervals of time
func TimeComparator(a, b time.Time) int {
	return a.UTC().Compare(b.UTC())
}
