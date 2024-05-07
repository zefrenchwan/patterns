package nodes_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestActiveTimeValues(t *testing.T) {
	a := nodes.NewActiveTimeValues()

	now := time.Now().UTC()
	after := now.AddDate(1, 0, 0)
	before := now.AddDate(-1, 0, 0)

	active, _ := nodes.NewFiniteTimeInterval(before, after, true, true)
	attrInterval := nodes.NewRightInfiniteTimeInterval(now, true)

	a.SetActivity(nodes.NewPeriod(active))
	a.SetValue("attr", "test")
	a.AddValue("attr", "other", nodes.NewPeriod(attrInterval))

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
	expectedBefore, _ := nodes.NewFiniteTimeInterval(before, now, true, false)
	expectedAfter, _ := nodes.NewFiniteTimeInterval(now, after, true, true)
	valueBefore := valueIntervalMap["test"][0]
	valueAfter := valueIntervalMap["other"][0]

	if nodes.TimeIntervalsCompare(expectedBefore, valueBefore) != 0 {
		t.Error("error when intersecting period and activity")
	}

	if nodes.TimeIntervalsCompare(expectedAfter, valueAfter) != 0 {
		t.Error("error when intersecting period and activity")
	}
}
