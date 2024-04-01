package patterns_test

import (
	"slices"
	"testing"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestIntervalsCompare(t *testing.T) {
	comparator := patterns.NewIntComparator()
	var a, b patterns.Interval[int]
	// empty test
	a = comparator.NewEmptyInterval()
	b = comparator.NewEmptyInterval()

	if !a.IsEmpty() {
		t.Fail()
	}

	if comparator.CompareInterval(a, b) != 0 {
		t.Error("failed empty equals empty")
	}

	a = comparator.NewFullInterval()
	if comparator.CompareInterval(a, b) > 0 {
		t.Error("failed empty is less than anything")
	}

	// full is more than anything
	a = comparator.NewLeftInfiniteInterval(10, false)
	b = comparator.NewFullInterval()

	if a.IsFull() || !b.IsFull() {
		t.Fail()
	}

	if comparator.CompareInterval(b, b) != 0 {
		t.Error("failed test on fulll is full")
	} else if comparator.CompareInterval(b, a) <= 0 {
		t.Error("failed full is more than anything")
	} else if comparator.CompareInterval(a, b) >= 0 {
		t.Error("failed full is more than anything")
	}

	// test cases on left infinite
	a = comparator.NewLeftInfiniteInterval(5, false)
	b = comparator.NewLeftInfiniteInterval(5, true)

	if a.IsEmpty() || b.IsEmpty() || a.IsFull() || b.IsFull() {
		t.Fail()
	}

	if comparator.CompareInterval(a, a) != 0 {
		t.Error("test equality failure")
	} else if comparator.CompareInterval(a, b) >= 0 {
		t.Error("failed test on left infinite: check right comparison")
	} else if comparator.CompareInterval(b, a) <= 0 {
		t.Error("failed test on left infinite: check right comparison")
	}

	a = comparator.NewLeftInfiniteInterval(2, true)
	b = comparator.NewLeftInfiniteInterval(5, true)
	if comparator.CompareInterval(a, b) >= 0 {
		t.Error("failed test on left infinite: check right value comparison")
	} else if comparator.CompareInterval(b, a) <= 0 {
		t.Error("failed test on left infinite: check right value comparison")
	}

	// test cases on right infinite
	a = comparator.NewRightInfiniteInterval(10, false)
	b = comparator.NewRightInfiniteInterval(10, true)

	if a.IsEmpty() || b.IsEmpty() || a.IsFull() || b.IsFull() {
		t.Fail()
	}

	if comparator.CompareInterval(a, a) != 0 {
		t.Error("equality failure")
	} else if comparator.CompareInterval(b, b) != 0 {
		t.Error("equality failure")
	} else if comparator.CompareInterval(a, b) >= 0 {
		t.Error("check left comparison")
	} else if comparator.CompareInterval(b, a) <= 0 {
		t.Error("check left comparison")
	}

	a = comparator.NewRightInfiniteInterval(1, true)
	if comparator.CompareInterval(a, b) >= 0 {
		t.Error("check left comparison")
	} else if comparator.CompareInterval(b, a) <= 0 {
		t.Error("check left comparison")
	}

	// test finite impossible intervals
	if _, err := comparator.NewFiniteInterval(10, 2, false, false); err == nil {
		t.Fail()
	} else if _, err := comparator.NewFiniteInterval(10, 10, false, true); err == nil {
		t.Fail()
	} else if _, err := comparator.NewFiniteInterval(10, 10, false, true); err == nil {
		t.Fail()
	}

	// test many combinations
	a = comparator.NewLeftInfiniteInterval(1, false)
	b = comparator.NewRightInfiniteInterval(10, true)
	if comparator.CompareInterval(a, b) >= 0 {
		t.Error("mixed failure for ]-oo, 1[ and [10, +oo[")
	} else if comparator.CompareInterval(b, a) <= 0 {
		t.Error("mixed failure for ]-oo, 1[ and [10, +oo[")
	}
}

func TestIntervalComplement(t *testing.T) {
	comparator := patterns.NewIntComparator()
	var a patterns.Interval[int]
	// empty test
	a = comparator.NewEmptyInterval()
	result := comparator.Complement(a)
	if len(result) != 1 || !result[0].IsFull() {
		t.Error("complement of empty should be full")
	}

	// full test
	a = comparator.NewFullInterval()
	result = comparator.Complement(a)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("complement of full should be empty")
	}

	// semi bounded intervals
	a = comparator.NewLeftInfiniteInterval(10, false)
	expected := comparator.NewRightInfiniteInterval(10, true)
	result = comparator.Complement(a)
	if len(result) != 1 || comparator.CompareInterval(expected, result[0]) != 0 {
		t.Error("complement failure for semi bounded intervals")
	}

	a = comparator.NewRightInfiniteInterval(1, false)
	expected = comparator.NewLeftInfiniteInterval(1, true)
	result = comparator.Complement(a)
	if len(result) != 1 || comparator.CompareInterval(expected, result[0]) != 0 {
		t.Error("complement failure for semi bounded intervals")
	}

	// bounded intervals
	a, errInterval := comparator.NewFiniteInterval(1, 10, true, false)
	if errInterval != nil {
		t.Fail()
	}

	result = comparator.Complement(a)
	if len(result) != 2 {
		t.Error("complement failure for semi bounded intervals")
	}

	exp1 := result[0]
	exp2 := result[1]

	if comparator.CompareInterval(exp1, exp2) > 0 {
		exp1, exp2 = exp2, exp1
	}

	if comparator.CompareInterval(exp1, comparator.NewLeftInfiniteInterval(1, false)) != 0 {
		t.Error("complement failure for semi bounded intervals")
	} else if comparator.CompareInterval(exp2, comparator.NewRightInfiniteInterval(10, true)) != 0 {
		t.Error("complement failure for semi bounded intervals")
	}
}

func TestIntervalComplementMixed(t *testing.T) {
	comparator := patterns.NewIntComparator()

	complements := comparator.Complement(comparator.NewLeftInfiniteInterval(10, false))
	expected := comparator.NewRightInfiniteInterval(10, true)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, complements) {
		t.Error("left infinite failure")
	}

	complements = comparator.Complement(comparator.NewLeftInfiniteInterval(10, true))
	expected = comparator.NewRightInfiniteInterval(10, false)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, complements) {
		t.Error("left infinite failure")
	}
}

func TestIntervalIntersection(t *testing.T) {
	comparator := patterns.NewIntComparator()
	var a, b, result, expected patterns.Interval[int]

	// test empty and full
	a = comparator.NewFullInterval()
	b = comparator.NewEmptyInterval()
	result = comparator.Intersection(a, b)
	if !result.IsEmpty() {
		t.Fail()
	}

	result = comparator.Intersection(b, a)
	if !result.IsEmpty() {
		t.Fail()
	}

	result = comparator.Intersection(a, a)
	if !result.IsFull() {
		t.Fail()
	}

	// test semi bounded
	a = comparator.NewLeftInfiniteInterval(10, true)
	b, _ = comparator.NewFiniteInterval(0, 20, true, false)
	expected, _ = comparator.NewFiniteInterval(0, 10, true, true)
	result = comparator.Intersection(a, b)
	if comparator.CompareInterval(expected, result) != 0 {
		t.Fail()
	}

	result = comparator.Intersection(b, a)
	if comparator.CompareInterval(expected, result) != 0 {
		t.Fail()
	}

	a = comparator.NewLeftInfiniteInterval(10, true)
	b = comparator.NewLeftInfiniteInterval(50, false)
	expected = comparator.NewLeftInfiniteInterval(10, true)
	result = comparator.Intersection(a, b)
	if comparator.CompareInterval(expected, result) != 0 {
		t.Fail()
	}

	a = comparator.NewLeftInfiniteInterval(10, true)
	b = comparator.NewRightInfiniteInterval(0, false)
	expected, _ = comparator.NewFiniteInterval(0, 10, false, true)
	result = comparator.Intersection(a, b)
	if comparator.CompareInterval(expected, result) != 0 {
		t.Fail()
	}

	// test bounded
	a, _ = comparator.NewFiniteInterval(0, 5, true, false)
	b, _ = comparator.NewFiniteInterval(100, 105, true, false)
	result = comparator.Intersection(a, b)
	if !result.IsEmpty() {
		t.Fail()
	}

	a, _ = comparator.NewFiniteInterval(0, 102, true, false)
	b, _ = comparator.NewFiniteInterval(100, 105, true, false)
	result = comparator.Intersection(a, b)
	expected, _ = comparator.NewFiniteInterval(100, 102, true, false)
	if comparator.CompareInterval(result, expected) != 0 {
		t.Fail()
	}
}

func slicesIntervalCompare[T any](comparator patterns.TypedComparator[T], expected []patterns.Interval[T], got []patterns.Interval[T]) bool {
	if len(expected) != len(got) {
		return false
	}

	for _, value := range expected {
		localTest := func(a patterns.Interval[T]) bool { return comparator.CompareInterval(a, value) == 0 }
		if !slices.ContainsFunc(got, localTest) {
			return false
		}
	}

	return true
}

func TestIntervalUnion(t *testing.T) {
	comparator := patterns.NewIntComparator()

	// test union of empty sets
	unions := []patterns.Interval[int]{
		comparator.NewEmptyInterval(), comparator.NewEmptyInterval(),
	}

	result := comparator.Union(comparator.NewEmptyInterval(), unions...)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("union of empty sets should be empty set")
	}

	// test with one full
	result = comparator.Union(comparator.NewFullInterval(), unions...)
	if len(result) != 1 || !result[0].IsFull() {
		t.Error("union of empty sets should be empty set")
	}

	// test max values only
	result = comparator.Union(comparator.NewLeftInfiniteInterval(10, true), comparator.NewLeftInfiniteInterval(0, false))
	if len(result) != 1 && comparator.CompareInterval(result[0], comparator.NewLeftInfiniteInterval(10, true)) != 0 {
		t.Error("error when filling max limit")
	}

	result = comparator.Union(comparator.NewLeftInfiniteInterval(10, true), comparator.NewLeftInfiniteInterval(10, false))
	if len(result) != 1 && comparator.CompareInterval(result[0], comparator.NewLeftInfiniteInterval(10, true)) != 0 {
		t.Error("error when filling max limit")
	}

	result = comparator.Union(comparator.NewLeftInfiniteInterval(10, true), comparator.NewLeftInfiniteInterval(100, false))
	if len(result) != 1 && comparator.CompareInterval(result[0], comparator.NewLeftInfiniteInterval(100, false)) != 0 {
		t.Error("error when filling max limit")
	}

	// test min values only
	result = comparator.Union(comparator.NewRightInfiniteInterval(10, true), comparator.NewRightInfiniteInterval(0, false))
	if len(result) != 1 && comparator.CompareInterval(result[0], comparator.NewRightInfiniteInterval(0, false)) != 0 {
		t.Error("error when filling min limit")
	}

	result = comparator.Union(comparator.NewRightInfiniteInterval(0, true), comparator.NewRightInfiniteInterval(0, false))
	if len(result) != 1 && comparator.CompareInterval(result[0], comparator.NewRightInfiniteInterval(0, true)) != 0 {
		t.Error("error when filling min limit")
	}

	result = comparator.Union(comparator.NewRightInfiniteInterval(0, true), comparator.NewRightInfiniteInterval(10, false))
	if len(result) != 1 && comparator.CompareInterval(result[0], comparator.NewRightInfiniteInterval(0, true)) != 0 {
		t.Error("error when filling min limit")
	}

	// test all disjoin sets
	finite, _ := comparator.NewFiniteInterval(10, 100, false, true)
	left := comparator.NewLeftInfiniteInterval(0, false)
	right := comparator.NewRightInfiniteInterval(1000, false)

	result = comparator.Union(left, finite, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{left, right, finite}, result) {
		t.Error("grouping separated intervals")
	}

	// group two infinite intervals if possible
	left = comparator.NewLeftInfiniteInterval(0, false)
	right = comparator.NewRightInfiniteInterval(0, false)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{left, right}, result) {
		t.Error("grouping intervals with same values but separated borders")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	left = comparator.NewLeftInfiniteInterval(0, false)
	right = comparator.NewRightInfiniteInterval(10, true)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{left, right}, result) {
		t.Error("grouping intervals with same values but separated borders")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	left = comparator.NewLeftInfiniteInterval(0, false)
	right = comparator.NewRightInfiniteInterval(0, true)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{comparator.NewFullInterval()}, result) {
		t.Error("grouping intervals with same values but separated borders")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	left = comparator.NewLeftInfiniteInterval(10, false)
	right = comparator.NewRightInfiniteInterval(0, false)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{comparator.NewFullInterval()}, result) {
		t.Error("grouping non separated intervals")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}
}

func TestIntervalUnionFinite(t *testing.T) {
	comparator := patterns.NewIntComparator()

	// separated despite same values
	left := comparator.NewLeftInfiniteInterval(10, false)
	right, _ := comparator.NewFiniteInterval(10, 100, false, true)
	result := comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{left, right}, result) {
		t.Error("boundaries failure: same values, both excluded")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	// totally separated
	left, _ = comparator.NewFiniteInterval(0, 5, false, false)
	right, _ = comparator.NewFiniteInterval(10, 100, false, true)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{left, right}, result) {
		t.Error("totally separated intervals failure")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	// right contains left
	left, _ = comparator.NewFiniteInterval(0, 5, false, false)
	right, _ = comparator.NewFiniteInterval(-5, 50, false, true)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{right}, result) {
		t.Error("one contains the other")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	// right contains left with common values
	left, _ = comparator.NewFiniteInterval(0, 5, true, false)
	right, _ = comparator.NewFiniteInterval(0, 5, false, true)
	expected, _ := comparator.NewFiniteInterval(0, 5, true, true)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("one contains the other with same values")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}

	// not empty intersection
	left, _ = comparator.NewFiniteInterval(0, 5, true, false)
	right, _ = comparator.NewFiniteInterval(2, 10, false, true)
	expected, _ = comparator.NewFiniteInterval(0, 10, true, true)
	result = comparator.Union(left, right)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("non separated intervals")
	} else if !slicesIntervalCompare(comparator, result, comparator.Union(right, left)) {
		t.Error("broken symetry")
	}
}

func TestIntervalIntersectionMixed(t *testing.T) {
	comparator := patterns.NewIntComparator()

	a := comparator.NewLeftInfiniteInterval(10, true)
	b := comparator.NewRightInfiniteInterval(0, false)
	expected, _ := comparator.NewFiniteInterval(0, 10, false, true)
	if comparator.CompareInterval(comparator.Intersection(a, b), expected) != 0 {
		t.Error("Failed mixed boundaries test")
	} else if comparator.CompareInterval(comparator.Intersection(b, a), expected) != 0 {
		t.Error("broken symetry")
	}

	a = comparator.NewRightInfiniteInterval(100, true)
	b = comparator.NewRightInfiniteInterval(10, false)
	expected = comparator.NewRightInfiniteInterval(100, true)
	if comparator.CompareInterval(comparator.Intersection(a, b), expected) != 0 {
		t.Error("Failed right boundaries test")
	} else if comparator.CompareInterval(comparator.Intersection(b, a), expected) != 0 {
		t.Error("broken symetry")
	}
}

func TestRemoveSingle(t *testing.T) {
	comparator := patterns.NewIntComparator()

	// test remove nothing
	base := comparator.NewFullInterval()
	result := comparator.Remove(base)
	if len(result) != 1 || !result[0].IsFull() {
		t.Fail()
	}

	base, _ = comparator.NewFiniteInterval(10, 100, false, true)
	result = comparator.Remove(base)
	if len(result) != 1 || comparator.CompareInterval(base, result[0]) != 0 {
		t.Fail()
	}

	// test remove empty
	base, _ = comparator.NewFiniteInterval(10, 100, false, true)
	result = comparator.Remove(base, comparator.NewEmptyInterval())
	if len(result) != 1 || comparator.CompareInterval(base, result[0]) != 0 {
		t.Fail()
	}

	// test remove full should make empty
	base, _ = comparator.NewFiniteInterval(10, 100, false, true)
	result = comparator.Remove(base, comparator.NewFullInterval())
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing full should make empty")
	}

	// removing itself should make empty
	base, _ = comparator.NewFiniteInterval(10, 100, false, true)
	result = comparator.Remove(base, base)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing itself should make empty")
	}

	base = comparator.NewLeftInfiniteInterval(10, true)
	result = comparator.Remove(base, base)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing itself should make empty")
	}

	base = comparator.NewRightInfiniteInterval(10, true)
	result = comparator.Remove(base, base)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing itself should make empty")
	}

	// test infinite boundaries
	base, _ = comparator.NewFiniteInterval(10, 100, false, true)
	other := comparator.NewLeftInfiniteInterval(1000, true)
	result = comparator.Remove(base, other)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing larger interval should make empty")
	}

	base, _ = comparator.NewFiniteInterval(10, 100, false, true)
	other = comparator.NewRightInfiniteInterval(0, true)
	result = comparator.Remove(base, other)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing larger interval should make empty")
	}

	base = comparator.NewRightInfiniteInterval(100, false)
	other = comparator.NewRightInfiniteInterval(0, true)
	result = comparator.Remove(base, other)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing larger interval should make empty")
	}

	base = comparator.NewLeftInfiniteInterval(0, false)
	other = comparator.NewLeftInfiniteInterval(100, true)
	result = comparator.Remove(base, other)
	if len(result) != 1 || !result[0].IsEmpty() {
		t.Error("removing larger interval should make empty")
	}

	// test infinite boundaries with non empty intersection
	// ]-oo, 100[ - [0, +oo[ = ]-oo, 0[
	base = comparator.NewLeftInfiniteInterval(100, false)
	other = comparator.NewRightInfiniteInterval(0, true)
	expected := comparator.NewLeftInfiniteInterval(0, false)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove, should be ]0, 100[")
	}

	// ]-oo, 100[ - ] -oo, 0] = ]0, 100[
	base = comparator.NewLeftInfiniteInterval(100, false)
	other = comparator.NewLeftInfiniteInterval(0, true)
	expected, _ = comparator.NewFiniteInterval(0, 100, false, false)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove, should be ]0, 100[")
	}

	// [10, +oo[ - ]100, +oo[ = [10, 100]
	base = comparator.NewRightInfiniteInterval(10, true)
	other = comparator.NewRightInfiniteInterval(100, false)
	expected, _ = comparator.NewFiniteInterval(10, 100, true, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove, should be [10, 100]")
	}

	// [10, +oo[ - ]10, +oo[ = [10, 10]
	base = comparator.NewRightInfiniteInterval(10, true)
	other = comparator.NewRightInfiniteInterval(10, false)
	expected, _ = comparator.NewFiniteInterval(10, 10, true, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed reduction to point")
	}

	// ] -oo, 10] - [10, +oo[ =]-oo, 10[]
	base = comparator.NewLeftInfiniteInterval(10, true)
	other = comparator.NewRightInfiniteInterval(10, true)
	expected = comparator.NewLeftInfiniteInterval(10, false)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed reduction to point")
	}

	// ] -oo, 10[ - ]10, +oo[ = ]-oo, 10[
	base = comparator.NewLeftInfiniteInterval(10, false)
	other = comparator.NewRightInfiniteInterval(10, false)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{base}, result) {
		t.Error("failed empty intersection test")
	}

	// ] -oo, 10] - ]-oo, 10[ = [10, 10]
	base = comparator.NewLeftInfiniteInterval(10, true)
	other = comparator.NewLeftInfiniteInterval(10, false)
	expected, _ = comparator.NewFiniteInterval(10, 10, true, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed reduction to point")
	}

	// [10, +oo[ - ]10, +oo[ = [10, 10]
	base = comparator.NewRightInfiniteInterval(10, true)
	other = comparator.NewRightInfiniteInterval(10, false)
	expected, _ = comparator.NewFiniteInterval(10, 10, true, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed reduction to point")
	}

	// [10, +oo [ - ]100 +oo[ = [10, 100]
	base = comparator.NewRightInfiniteInterval(10, true)
	other = comparator.NewRightInfiniteInterval(100, false)
	expected, _ = comparator.NewFiniteInterval(10, 100, true, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed getting finite part from two right infinites")
	}

}

func TestRemoveSingleFinite(t *testing.T) {
	comparator := patterns.NewIntComparator()

	base, _ := comparator.NewFiniteInterval(10, 100, true, false)
	other := comparator.NewRightInfiniteInterval(50, false)
	expected, _ := comparator.NewFiniteInterval(10, 50, true, true)
	result := comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed finite interval remove")
	}

	base, _ = comparator.NewFiniteInterval(10, 100, true, true)
	other = comparator.NewLeftInfiniteInterval(50, false)
	expected, _ = comparator.NewFiniteInterval(50, 100, true, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed finite interval remove")
	}

	// test remove point
	base, _ = comparator.NewFiniteInterval(100, 200, true, true)
	other, _ = comparator.NewFiniteInterval(100, 100, true, true)
	expected, _ = comparator.NewFiniteInterval(100, 200, false, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove point")
	}

	base, _ = comparator.NewFiniteInterval(100, 200, true, true)
	other, _ = comparator.NewFiniteInterval(200, 200, true, true)
	expected, _ = comparator.NewFiniteInterval(100, 200, true, false)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove point")
	}

	base, _ = comparator.NewFiniteInterval(100, 200, false, true)
	other, _ = comparator.NewFiniteInterval(100, 100, true, true)
	expected, _ = comparator.NewFiniteInterval(100, 200, false, true)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove point")
	}

	base, _ = comparator.NewFiniteInterval(100, 200, true, false)
	other, _ = comparator.NewFiniteInterval(200, 200, true, true)
	expected, _ = comparator.NewFiniteInterval(100, 200, true, false)
	result = comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expected}, result) {
		t.Error("failed remove point")
	}
}

func TestRemoveSplit(t *testing.T) {
	comparator := patterns.NewIntComparator()

	// split parts
	base := comparator.NewRightInfiniteInterval(10, true)
	other, _ := comparator.NewFiniteInterval(100, 200, true, false)
	expectedPartOne, _ := comparator.NewFiniteInterval(10, 100, true, false)
	expectedPartTwo := comparator.NewRightInfiniteInterval(200, true)
	result := comparator.Remove(base, other)
	if !slicesIntervalCompare(comparator, []patterns.Interval[int]{expectedPartOne, expectedPartTwo}, result) {
		t.Error("failed splitting right infinite")
	}
}
