package patterns_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestPeriodAddRemoveIntervals(t *testing.T) {
	comparator := patterns.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	after := now.AddDate(1, 0, 0)

	afterInterval := comparator.NewLeftInfiniteInterval(after, true)
	beforeInterval := comparator.NewLeftInfiniteInterval(before, false)

	pAfter := patterns.NewPeriod(afterInterval)
	pBefore := patterns.NewPeriod(beforeInterval)

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
	pAfter = patterns.NewPeriod(afterInterval)
	pBefore = patterns.NewPeriod(beforeInterval)

	// remove when other contains receiver
	pBefore.Remove(pAfter)
	if !pBefore.IsEmptyPeriod() {
		t.Error("before included in after, so before - after should be empty")
	}

	// test when period is larger that removed part
	pAfter = patterns.NewPeriod(afterInterval)
	pBefore = patterns.NewPeriod(beforeInterval)
	pAfter.Remove(pBefore)
	expected := comparator.Remove(afterInterval, beforeInterval)[0]
	result := pAfter.AsIntervals()
	if len(result) != 1 || comparator.CompareInterval(expected, result[0]) != 0 {
		t.Error("failed to remive a single interval in a single interval")
	}
}

func TestPeriodRemoveManyIntervals(t *testing.T) {
	comparator := patterns.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	longAgo := before.AddDate(-2, 0, 0)
	after := now.AddDate(1, 0, 0)
	// longAgo < before < now < after
	longAgoInterval := comparator.NewLeftInfiniteInterval(longAgo, true)
	beforeToNow, _ := comparator.NewFiniteInterval(before, now, false, false)
	nowToAfter, _ := comparator.NewFiniteInterval(now, after, true, true)

	// period is ]-oo, longAgo ] union ]before, now[
	period := patterns.NewPeriod(longAgoInterval)
	period.AddInterval(beforeToNow)
	// otherPeriod is [now, after]
	otherPeriod := patterns.NewPeriod(nowToAfter)
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
	comparator := patterns.NewTimeComparator()
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
	period := patterns.NewPeriod(nowInterval)
	period.AddInterval(afterInterval)
	// otherPeriod is ]longAgo, before[ union [now, +oo[
	otherPeriod := patterns.NewPeriod(longAgoToBefore)
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
	comparator := patterns.NewTimeComparator()
	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	longAgo := before.AddDate(-2, 0, 0)
	after := now.AddDate(1, 0, 0)
	// longAgo < before < now < after

	afterInterval := comparator.NewRightInfiniteInterval(after, false)
	intermedInterval, _ := comparator.NewFiniteInterval(longAgo, before, false, true)
	period := patterns.NewPeriod(afterInterval)
	period.AddInterval(intermedInterval)

	// complement should be ]-oo, longAgo] union ]before, after]
	period.Complement()
	expectedInterval, _ := comparator.NewFiniteInterval(before, after, false, true)
	expected := []patterns.Interval[time.Time]{
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
