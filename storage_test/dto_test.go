package storage_test

import (
	"slices"
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/patterns"
	"github.com/zefrenchwan/patterns.git/storage"
)

func TestEntitySerde(t *testing.T) {
	now := time.Now().UTC().Truncate(1 * time.Second)
	leftInterval := patterns.NewLeftInfiniteTimeInterval(now, false)
	rightInterval := patterns.NewRightInfiniteTimeInterval(now, true)
	leftPeriod := patterns.NewPeriod(leftInterval)
	rightPeriod := patterns.NewPeriod(rightInterval)

	entity := patterns.NewEntity([]string{"Person"})
	entity.AddValue("last name", "MEEE", leftPeriod)
	entity.AddValue("first name", "Me", rightPeriod)

	dto := storage.SerializeEntity(entity)
	reverse, errReverse := storage.DerializeEntity(dto)
	if errReverse != nil {
		t.Errorf("failing deserialization %s", errReverse.Error())
	} else if reverse.Id() != entity.Id() {
		t.Fail()
	} else if slices.Compare(entity.Traits(), reverse.Traits()) != 0 {
		t.Fail()
	} else if len(reverse.Attributes()) != 2 {
		t.Fail()
	} else if !reverse.ContainsAttribute("first name") {
		t.Fail()
	} else if !reverse.ContainsAttribute("last name") {
		t.Fail()
	}

	period := reverse.ActivePeriod()
	if !period.IsFullPeriod() {
		t.Fail()
	}

	values, errValues := reverse.PeriodValuesForAttribute("first name")
	if errValues != nil {
		t.Fail()
	} else if len(values) != 1 {
		t.Fail()
	} else if value, found := values["Me"]; !found {
		t.Fail()
	} else if !value.IsSameAs(rightPeriod) {
		t.Fail()
	}

	values, errValues = reverse.PeriodValuesForAttribute("last name")
	if errValues != nil {
		t.Fail()
	} else if len(values) != 1 {
		t.Fail()
	} else if value, found := values["MEEE"]; !found {
		t.Fail()
	} else if !value.IsSameAs(leftPeriod) {
		t.Fail()
	}
}

func TestRelationSerde(t *testing.T) {
	now := time.Now().UTC().Truncate(1 * time.Second)
	leftInterval := patterns.NewLeftInfiniteTimeInterval(now, false)
	leftPeriod := patterns.NewPeriod(leftInterval)

	links := map[string][]string{
		patterns.SUBJECT_ROLE: {"X"},
		patterns.OBJECT_ROLE:  {"Y"},
	}

	relation := patterns.NewRelationWithIdAndRoles("popo", []string{"Couple"}, links)
	relation.SetActivePeriod(leftPeriod)

	dto := storage.SerializeRelation(relation)
	reverse, ErrReverse := storage.DeserializeRelation(dto)
	if ErrReverse != nil {
		t.Errorf("error while deserialize: %s", ErrReverse.Error())
	}

	period := reverse.ActivePeriod()
	if !period.IsSameAs(relation.ActivePeriod()) {
		t.Fail()
	} else if relation.Id() != reverse.Id() {
		t.Fail()
	}

	reverseRoles := reverse.GetValuesPerRole()
	if len(reverseRoles) != len(links) {
		t.Fail()
	}

	for k, v := range links {
		if slices.Compare(v, reverseRoles[k]) != 0 {
			t.Fail()
		}
	}
}
