package nodes_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestTimeDependentRelation(t *testing.T) {
	relation := nodes.NewRelation("entity id", []string{"cat"})
	activePeriod := relation.ActivePeriod()
	if !activePeriod.IsFullPeriod() {
		t.Error("default validity should be full")
	}

	now := time.Now().UTC()
	period := nodes.NewPeriod(nodes.NewLeftInfiniteTimeInterval(now, true))
	relation = nodes.NewTimeDependentRelation("entity id", []string{"rel"}, period)
	activePeriod = relation.ActivePeriod()
	if !activePeriod.IsSameAs(period) {
		t.Error("period do not match parameter")
	}
}

func TestRelationsRole(t *testing.T) {
	relation := nodes.NewRelation("entity id", []string{"loves"})
	relation.SetValuesForRole(nodes.RELATION_ROLE_OBJECT, []string{"other entity"})

	subjects := relation.ValuesPerRole()[nodes.RELATION_ROLE_SUBJECT]
	if len(subjects) != 1 || subjects[0] != "entity id" {
		t.Error("subject value error")
	}

	values := relation.ValuesPerRole()
	if len(values) != 2 {
		t.Error("expecting two values")
	} else if objectValue := values[nodes.RELATION_ROLE_OBJECT]; len(objectValue) != 1 {
		t.Error("object role not set")
	} else if objectValue[0] != "other entity" {
		t.Error("object role not set")
	} else if subjectValue := values[nodes.RELATION_ROLE_SUBJECT]; len(subjectValue) != 1 {
		t.Error("subject role not set")
	} else if subjectValue[0] != "entity id" {
		t.Error("subject role not correct")
	}
}
