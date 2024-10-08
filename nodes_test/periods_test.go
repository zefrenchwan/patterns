package nodes_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestPeriodAddRemoveIntervals(t *testing.T) {
	comparator := nodes.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	after := now.AddDate(1, 0, 0)

	afterInterval := comparator.NewLeftInfiniteInterval(after, true)
	beforeInterval := comparator.NewLeftInfiniteInterval(before, false)

	pAfter := nodes.NewPeriod(afterInterval)
	pBefore := nodes.NewPeriod(beforeInterval)

	// remove same interval
	pBefore.Remove(pBefore)
	if !pBefore.IsEmptyPeriod() {
		t.Error("period minus itself should be empty")
	}

	pAfter.Remove(pAfter)
	if !pAfter.IsEmptyPeriod() {
		t.Error("period minus itself should be empty")
	}

	// reset
	pAfter = nodes.NewPeriod(afterInterval)
	pBefore = nodes.NewPeriod(beforeInterval)

	// remove when other contains receiver
	pBefore.Remove(pAfter)
	if !pBefore.IsEmptyPeriod() {
		t.Error("before included in after, so before - after should be empty")
	}

	// test when period is larger that removed part
	pAfter = nodes.NewPeriod(afterInterval)
	pBefore = nodes.NewPeriod(beforeInterval)
	pAfter.Remove(pBefore)
	expected := comparator.Remove(afterInterval, beforeInterval)[0]
	result := pAfter.AsIntervals()
	if len(result) != 1 || comparator.CompareInterval(expected, result[0]) != 0 {
		t.Error("failed to remive a single interval in a single interval")
	}
}

func TestPeriodRemoveManyIntervals(t *testing.T) {
	comparator := nodes.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	longAgo := before.AddDate(-2, 0, 0)
	after := now.AddDate(1, 0, 0)
	// longAgo < before < now < after
	longAgoInterval := comparator.NewLeftInfiniteInterval(longAgo, true)
	beforeToNow, _ := comparator.NewFiniteInterval(before, now, false, false)
	nowToAfter, _ := comparator.NewFiniteInterval(now, after, true, true)

	// period is ]-oo, longAgo ] union ]before, now[
	period := nodes.NewPeriod(longAgoInterval)
	period.AddInterval(beforeToNow)
	// otherPeriod is [now, after]
	otherPeriod := nodes.NewPeriod(nowToAfter)
	period.Remove(otherPeriod)
	// period should be the same as before
	result := period.AsIntervals()
	if len(result) != 2 ||
		comparator.CompareInterval(result[0], longAgoInterval) != 0 ||
		comparator.CompareInterval(result[1], beforeToNow) != 0 {
		t.Error("removing other elements should not change current period")
	}
}

func TestPeriodsIntersection(t *testing.T) {
	comparator := nodes.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	longAgo := before.AddDate(-2, 0, 0)
	after := now.AddDate(1, 0, 0)
	// longAgo < before < now < after

	nowInterval := comparator.NewLeftInfiniteInterval(now, true)
	afterInterval := comparator.NewRightInfiniteInterval(after, false)
	longAgoToBefore, _ := comparator.NewFiniteInterval(longAgo, before, false, false)
	nowOtherInterval := comparator.NewRightInfiniteInterval(now, true)
	singletonNow, _ := comparator.NewFiniteInterval(now, now, true, true)

	// period is ]-oo, now] union ]after, +oo[
	period := nodes.NewPeriod(nowInterval)
	period.AddInterval(afterInterval)
	// otherPeriod is ]longAgo, before[ union [now, +oo[
	otherPeriod := nodes.NewPeriod(longAgoToBefore)
	otherPeriod.AddInterval(nowOtherInterval)
	// intersection should be ]longAgo, before[ union ]after, +oo[
	period.Intersection(otherPeriod)
	intersection := period.AsIntervals()
	if len(intersection) != 3 ||
		comparator.CompareInterval(longAgoToBefore, intersection[0]) != 0 ||
		comparator.CompareInterval(singletonNow, intersection[1]) != 0 ||
		comparator.CompareInterval(afterInterval, intersection[2]) != 0 {
		t.Error("failed periods intersection")
	}
}

func TestPeriodsComplement(t *testing.T) {
	comparator := nodes.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	longAgo := before.AddDate(-2, 0, 0)
	after := now.AddDate(1, 0, 0)
	// longAgo < before < now < after

	afterInterval := comparator.NewRightInfiniteInterval(after, false)
	intermedInterval, _ := comparator.NewFiniteInterval(longAgo, before, false, true)
	period := nodes.NewPeriod(afterInterval)
	period.AddInterval(intermedInterval)

	// complement should be ]-oo, longAgo] union ]before, after]
	period.Complement()
	expectedInterval, _ := comparator.NewFiniteInterval(before, after, false, true)
	expected := []nodes.Interval[time.Time]{
		comparator.NewLeftInfiniteInterval(longAgo, true),
		expectedInterval,
	}

	complements := period.AsIntervals()
	if len(complements) != 2 ||
		comparator.CompareInterval(complements[0], expected[0]) != 0 ||
		comparator.CompareInterval(complements[1], expected[1]) != 0 {
		t.Error("error when taking complement")
	}
}

func TestPeriodSerde(t *testing.T) {
	full := nodes.NewFullPeriod()
	if result := nodes.SerializePeriod(full, "2006-01-02"); slices.Compare(result, []string{"]-oo;+oo["}) != 0 {
		t.Errorf("expected ]-oo;+oo[ for full period, got %s", strings.Join(result, ","))
	}

	empty := nodes.NewEmptyPeriod()
	if result := nodes.SerializePeriod(empty, "2006-01-02"); slices.Compare(result, []string{"];["}) != 0 {
		t.Errorf("expected ];[ for empty period, got %s", strings.Join(result, ","))
	}

	comparator := nodes.NewTimeComparator()
	now := time.Now().UTC().Truncate(24 * time.Hour)
	before := now.AddDate(-1, 0, 0)
	longAgo := before.AddDate(-2, 0, 0)
	after := now.AddDate(1, 0, 0)
	leftInterval := comparator.NewLeftInfiniteInterval(longAgo, false)
	middleInterval, _ := comparator.NewFiniteInterval(before, now, true, true)
	rightInterval := comparator.NewRightInfiniteInterval(after, false)

	reunion := nodes.NewPeriod(leftInterval)
	reunion.AddInterval(middleInterval)
	reunion.AddInterval(rightInterval)

	resultStr := nodes.SerializePeriod(reunion, "2006-01-02")
	result, errStr := nodes.DeserializePeriod(resultStr, "2006-01-02")
	if errStr != nil {
		t.Errorf("error while reading serialized values: %s", errStr.Error())
	} else if !reunion.IsSameAs(result) {
		t.Error("serde failure, not same values")
	}
}

func TestPeriodContainingInterval(t *testing.T) {
	full := nodes.NewFullPeriod()
	container := full.ContainingTimeInterval()
	if !container.IsFull() {
		t.Fail()
	}

	empty := nodes.NewEmptyPeriod()
	container = empty.ContainingTimeInterval()
	if !container.IsEmpty() {
		t.Fail()
	}

	now := time.Now()
	before := now.AddDate(-10, 0, 0)
	after := now.AddDate(10, 0, 0)
	otherInterval, _ := nodes.NewFiniteTimeInterval(now, after, false, true)
	partial := nodes.NewPeriod(nodes.NewLeftInfiniteTimeInterval(before, false))
	partial.AddInterval(otherInterval)
	container = partial.ContainingTimeInterval()
	expected := nodes.NewLeftInfiniteTimeInterval(after, true)
	comparator := nodes.NewTimeComparator()
	if comparator.CompareInterval(expected, container) != 0 {
		t.Fail()
	}
}
