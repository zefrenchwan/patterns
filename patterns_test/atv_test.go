package patterns_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestActiveTimeValues(t *testing.T) {
	a := patterns.NewActiveTimeValues()

	now := time.Now().UTC()
	after := now.AddDate(1, 0, 0)
	before := now.AddDate(-1, 0, 0)

	active, _ := patterns.NewFiniteTimeInterval(before, after, true, true)
	attrInterval := patterns.NewRightInfiniteTimeInterval(now, true)

	a.SetActivity(patterns.NewPeriod(active))
	a.SetValue("attr", "test")
	a.AddValue("attr", "other", patterns.NewPeriod(attrInterval))

	valueIntervalMap, errRead := a.TimeValuesForAttribute("attr")
	if errRead != nil {
		t.Fail()
	} else if len(valueIntervalMap) != 2 {
		t.Fail()
	} else {
		for _, intervals := range valueIntervalMap {
			if len(intervals) != 1 {
				t.Error("expecting one interval exactly per attribute")
			}
		}
	}

	// expected is between before and now => test, and between now and after, other
	expectedBefore, _ := patterns.NewFiniteTimeInterval(before, now, true, false)
	expectedAfter, _ := patterns.NewFiniteTimeInterval(now, after, true, true)
	valueBefore := valueIntervalMap["test"][0]
	valueAfter := valueIntervalMap["other"][0]

	if patterns.TimeIntervalsCompare(expectedBefore, valueBefore) != 0 {
		t.Error("error when intersecting period and activity")
	}

	if patterns.TimeIntervalsCompare(expectedAfter, valueAfter) != 0 {
		t.Error("error when intersecting period and activity")
	}
}
