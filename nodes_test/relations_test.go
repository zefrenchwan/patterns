package nodes_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestTimeDependentRelation(t *testing.T) {
	relation := nodes.NewRelation([]string{"cat"})
	relation.SetValuesForRole(nodes.RELATION_ROLE_SUBJECT, []string{"entity id"})
	activePeriod := relation.ActivePeriod()
	if !activePeriod.IsFullPeriod() {
		t.Error("default validity should be full")
	}

	now := time.Now().UTC()
	period := nodes.NewPeriod(nodes.NewLeftInfiniteTimeInterval(now, true))
	relation.SetActivePeriod(period)
	activePeriod = relation.ActivePeriod()
	if !activePeriod.IsSameAs(period) {
		t.Error("set active period has no effect")
	}
}

func TestRelationsRole(t *testing.T) {
	relation := nodes.NewRelation([]string{"loves"})
	relation.SetValuesForRole(nodes.RELATION_ROLE_SUBJECT, []string{"entity id"})
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

func TestPeriodRelationRole(t *testing.T) {
	now := time.Now()
	before := now.AddDate(-1, 0, 0)
	after := now.AddDate(1, 0, 0)
	beforePeriod := nodes.NewPeriod(nodes.NewRightInfiniteTimeInterval(before, true))
	afterPeriod := nodes.NewPeriod(nodes.NewRightInfiniteTimeInterval(after, true))
	nowPeriod := nodes.NewPeriod(nodes.NewRightInfiniteTimeInterval(now, true))

	company := nodes.NewEntity([]string{"Company"})
	relation := nodes.NewRelation([]string{"sells"})
	goodProduct := nodes.NewEntity([]string{"Product"})
	superProduct := nodes.NewEntity([]string{"Product"})

	relation.SetValuesForRole(nodes.RELATION_ROLE_SUBJECT, []string{company.Id()})
	relation.AddPeriodValueForRole(nodes.RELATION_ROLE_OBJECT, goodProduct.Id(), beforePeriod)
	relation.AddPeriodValueForRole(nodes.RELATION_ROLE_OBJECT, superProduct.Id(), afterPeriod)

	// test add period when there is no element
	values := relation.PeriodValuesPerRole()
	if values == nil || len(values) != 2 {
		t.Fail()
	} else if subjects := values[nodes.RELATION_ROLE_SUBJECT]; subjects == nil {
		t.Fail()
	} else if objects := values[nodes.RELATION_ROLE_OBJECT]; objects == nil {
		t.Fail()
	} else if len(subjects) != 1 {
		t.Fail()
	} else if periodSubject := subjects[company.Id()]; !periodSubject.IsFullPeriod() {
		t.Fail()
	} else if len(objects) != 2 {
		t.Fail()
	} else if goodPeriod := objects[goodProduct.Id()]; !goodPeriod.IsSameAs(beforePeriod) {
		t.Fail()
	} else if superPeriod := objects[superProduct.Id()]; !superPeriod.IsSameAs(afterPeriod) {
		t.Fail()
	}

	// test adding period
	relation.AddPeriodValueForRole(nodes.RELATION_ROLE_OBJECT, superProduct.Id(), nowPeriod)
	values = relation.PeriodValuesPerRole()
	if values == nil || len(values) != 2 {
		t.Fail()
	} else if subjects := values[nodes.RELATION_ROLE_SUBJECT]; subjects == nil {
		t.Fail()
	} else if objects := values[nodes.RELATION_ROLE_OBJECT]; objects == nil {
		t.Fail()
	} else if len(subjects) != 1 {
		t.Fail()
	} else if periodSubject := subjects[company.Id()]; !periodSubject.IsFullPeriod() {
		t.Fail()
	} else if len(objects) != 2 {
		t.Fail()
	} else if goodPeriod := objects[goodProduct.Id()]; !goodPeriod.IsSameAs(beforePeriod) {
		t.Fail()
	} else if superPeriod := objects[superProduct.Id()]; !superPeriod.IsSameAs(nowPeriod) {
		t.Fail()
	}

	// test remove
	relation.RemovePeriodValueForRole(nodes.RELATION_ROLE_OBJECT, goodProduct.Id(), nodes.NewFullPeriod())
	relation.RemovePeriodValueForRole(nodes.RELATION_ROLE_OBJECT, superProduct.Id(), nodes.NewFullPeriod())
	// only subject left
	values = relation.PeriodValuesPerRole()
	if values == nil || len(values) != 1 {
		t.Fail()
	} else if subjects := values[nodes.RELATION_ROLE_SUBJECT]; subjects == nil {
		t.Fail()
	} else if len(subjects) != 1 {
		t.Fail()
	} else if periodSubject := subjects[company.Id()]; !periodSubject.IsFullPeriod() {
		t.Fail()
	}
}
