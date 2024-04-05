package patterns_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestTimeDependentRelation(t *testing.T) {
	relation := patterns.NewRelation("entity id", []string{"cat"})
	activePeriod := relation.ActivePeriod()
	if !activePeriod.IsFullPeriod() {
		t.Error("default validity should be full")
	}

	now := time.Now().UTC()
	period := patterns.NewPeriod(patterns.NewLeftInfiniteTimeInterval(now, true))
	relation = patterns.NewTimeDependentRelation("entity id", []string{"rel"}, period)
	activePeriod = relation.ActivePeriod()
	if !activePeriod.IsSameAs(period) {
		t.Error("period do not match parameter")
	}
}

func TestRelationsRole(t *testing.T) {
	relation := patterns.NewRelation("entity id", []string{"loves"})
	relation.SetValueForRole(patterns.OBJECT_ROLE, "other entity")

	subjects := relation.GetSubjects()
	if len(subjects) != 1 || subjects[0] != "entity id" {
		t.Error("subject value error")
	}

	values := relation.GetValuesPerRole()
	if len(values) != 2 {
		t.Error("expecting two values")
	} else if objectValue := values[patterns.OBJECT_ROLE]; len(objectValue) != 1 {
		t.Error("object role not set")
	} else if objectValue[0] != "other entity" {
		t.Error("object role not set")
	} else if subjectValue := values[patterns.SUBJECT_ROLE]; len(subjectValue) != 1 {
		t.Error("subject role not set")
	} else if subjectValue[0] != "entity id" {
		t.Error("subject role not correct")
	}
}
