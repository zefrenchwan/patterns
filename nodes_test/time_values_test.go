package nodes_test

import (
	"slices"
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestTimeValuesSetAttribute(t *testing.T) {
	instance := nodes.NewTimeValues()

	if err := instance.SetValue("attr", "a value"); err != nil {
		t.Fail()
	}

	if v, err := instance.ValuesForAttribute("otherAttr"); v != nil || err != nil {
		t.Error("failed test when loading values in unused attribute")
	}

	instance.SetValue("attr", "final value")

	if values, err := instance.ValuesForAttribute("attr"); values == nil || err != nil {
		t.Error("no value found for existing attribute")
	} else if len(values) != 1 {
		t.Error("error when finding existing attribute")
	} else if values[0] != "final value" {
		t.Error("error when finding value")
	}

	// final value should be set for full period
	if values, err := instance.TimeValuesForAttribute("attr"); err != nil {
		t.Fail()
	} else if len(values) != 1 {
		t.Error("map for attributes error")
	} else if !values["final value"][0].IsFull() {
		t.Error("setting value should make full")
	}
}

func TestTimeValuesAddAttribute(t *testing.T) {
	instance := nodes.NewTimeValues()

	now := time.Now().UTC()
	beforeNow := nodes.NewLeftInfiniteTimeInterval(now, false)
	afterNow := nodes.NewRightInfiniteTimeInterval(now, true)

	instance.AddValue("attr", "before", nodes.NewPeriod(beforeNow))
	instance.AddValue("attr", "after", nodes.NewPeriod(afterNow))

	// test values, not periods
	var values []string
	if v, err := instance.ValuesForAttribute("attr"); err != nil {
		t.Fail()
	} else if len(v) != 2 {
		t.Error("missing values when many values")
	} else {
		slices.Sort(v)
		values = v
	}

	if slices.Compare([]string{"after", "before"}, values) != 0 {
		t.Error("no match for many values")
	}

	// test periods
	if valuesMap, err := instance.TimeValuesForAttribute("attr"); err != nil {
		t.Fail()
	} else if len(valuesMap) != 2 {
		t.Error("missing values in map of values")
	} else if beforeValue := valuesMap["before"]; len(beforeValue) != 1 {
		t.Error("intervals test failed")
	} else if nodes.TimeIntervalsCompare(beforeNow, beforeValue[0]) != 0 {
		t.Error("intervals test failed")
	} else if afterValue := valuesMap["after"]; len(afterValue) != 1 {
		t.Error("intervals test failed")
	} else if nodes.TimeIntervalsCompare(afterValue[0], afterNow) != 0 {
		t.Error("intervals test failed")
	}
}

func TestTimeValuesPeriodChange(t *testing.T) {
	instance := nodes.NewTimeValues()

	now := time.Now().UTC()
	beforeNow := nodes.NewLeftInfiniteTimeInterval(now, false)
	afterNow := nodes.NewRightInfiniteTimeInterval(now, true)

	instance.SetValue("attr", "before")
	instance.AddValue("attr", "after", nodes.NewPeriod(afterNow))

	// test values, not periods
	var values []string
	if v, err := instance.ValuesForAttribute("attr"); err != nil {
		t.Fail()
	} else if len(v) != 2 {
		t.Error("missing values when many values")
	} else {
		slices.Sort(v)
		values = v
	}

	if slices.Compare([]string{"after", "before"}, values) != 0 {
		t.Error("no match for many values")
	}

	// test periods
	if valuesMap, err := instance.TimeValuesForAttribute("attr"); err != nil {
		t.Fail()
	} else if len(valuesMap) != 2 {
		t.Error("missing values in map of values")
	} else if beforeValue := valuesMap["before"]; len(beforeValue) != 1 {
		t.Error("intervals test failed")
	} else if nodes.TimeIntervalsCompare(beforeNow, beforeValue[0]) != 0 {
		t.Error("intervals test failed")
	} else if afterValue := valuesMap["after"]; len(afterValue) != 1 {
		t.Error("intervals test failed")
	} else if nodes.TimeIntervalsCompare(afterValue[0], afterNow) != 0 {
		t.Error("intervals test failed")
	}
}

func TestPeriodForAttributes(t *testing.T) {
	instance := nodes.NewTimeValues()

	now := time.Now().UTC()
	before := now.AddDate(-1, 0, 0)
	after := now.AddDate(1, 0, 0)

	beforeInterval := nodes.NewRightInfiniteTimeInterval(before, false)
	beforePeriod := nodes.NewPeriod(beforeInterval)
	nowInterval := nodes.NewRightInfiniteTimeInterval(now, false)
	nowPeriod := nodes.NewPeriod(nowInterval)
	afterInterval := nodes.NewRightInfiniteTimeInterval(after, false)
	afterPeriod := nodes.NewPeriod(afterInterval)
	expectedIntervalValueBefore, _ := nodes.NewFiniteTimeInterval(before, now, false, true)
	expectedPeriodValueBefore := nodes.NewPeriod(expectedIntervalValueBefore)

	// afterPeriod should be the period for value since after, before of set and not add
	// BUT, between second and third line, value since before should reduce to (before, now)
	instance.SetPeriodForValue("attr", "value since before", beforePeriod)
	instance.SetPeriodForValue("attr", "value since after", nowPeriod)
	instance.SetPeriodForValue("attr", "value since after", afterPeriod)

	mapValuePeriods, errPeriods := instance.PeriodsForAttribute("attr")
	if errPeriods != nil {
		t.Errorf("unexpected error %s", errPeriods.Error())
	} else if len(mapValuePeriods) != 2 {
		t.Error("missing value, both should be here")
	} else if afterPeriodResult := mapValuePeriods["value since after"]; !afterPeriod.IsSameAs(afterPeriodResult) {
		t.Error("forcing value failure")
	} else if beforePeriodResult := mapValuePeriods["value since before"]; !expectedPeriodValueBefore.IsSameAs(beforePeriodResult) {
		t.Error("removeing other periods failure")
	}
}

func TestRemovePeriodInTimeValue(t *testing.T) {
	instance := nodes.NewTimeValues()

	now := time.Now().UTC()
	after := now.AddDate(1, 0, 0)
	expectedInterval := nodes.NewLeftInfiniteTimeInterval(now, true)
	afterInterval := nodes.NewRightInfiniteTimeInterval(after, false)

	instance.SetValue("attr", "default value")
	instance.SetPeriodForValue("attr", "after value won't appear", nodes.NewPeriod(afterInterval))
	instance.RemovePeriodForAttribute("attr", nodes.NewPeriod(nodes.NewRightInfiniteTimeInterval(now, false)))

	valueIntervalsMap, _ := instance.TimeValuesForAttribute("attr")
	if len(valueIntervalsMap) != 1 {
		t.Error("keeping old value")
	} else if intervals := valueIntervalsMap["default value"]; len(intervals) != 1 {
		t.Error("intervals cut failure")
	} else if nodes.TimeIntervalsCompare(expectedInterval, intervals[0]) != 0 {
		t.Error("removing part of interval failed")
	}
}
