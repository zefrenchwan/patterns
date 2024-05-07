package nodes_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestEntityFullActive(t *testing.T) {
	entity := nodes.NewEntity([]string{"city"})
	entity.SetValue("name", "Paris")

	activity := entity.ActivePeriod()
	if !activity.IsFullPeriod() {
		t.Error("entity should be fully active by default")
	}

	values, errValues := entity.PeriodValuesForAttribute("name")
	if errValues != nil {
		t.Error("not nil entity should have values for that attribute")
	} else if len(values) != 1 {
		t.Error("expecting one value for attribute")
	} else if period := values["Paris"]; !period.IsFullPeriod() {
		t.Error("period for name should be always")
	}
}

func TestEntityActive(t *testing.T) {
	activityInterval := nodes.NewRightInfiniteTimeInterval(time.Now().UTC(), false)
	activity := nodes.NewPeriod(activityInterval)
	entity, _ := nodes.NewEntityDuring([]string{"city"}, activity)
	entity.SetValue("name", "Paris")

	values, errValues := entity.PeriodValuesForAttribute("name")
	if errValues != nil {
		t.Error("not nil entity should have values for that attribute")
	} else if len(values) != 1 {
		t.Error("expecting one value for attribute")
	} else if period := values["Paris"]; !period.IsSameAs(activity) {
		t.Error("period for name should match activity")
	}
}
