package storage_test

import (
	"slices"
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
	"github.com/zefrenchwan/patterns.git/storage"
)

func TestEntitySerde(t *testing.T) {
	now := time.Now().UTC().Truncate(1 * time.Second)
	leftInterval := nodes.NewLeftInfiniteTimeInterval(now, false)
	rightInterval := nodes.NewRightInfiniteTimeInterval(now, true)
	leftPeriod := nodes.NewPeriod(leftInterval)
	rightPeriod := nodes.NewPeriod(rightInterval)

	entity := nodes.NewEntity([]string{"Person"})
	entity.AddValue("last name", "MEEE", leftPeriod)
	entity.AddValue("first name", "Me", rightPeriod)

	var reverseEntity *nodes.Entity
	dto := storage.SerializeElement(&entity)
	reverse, errReverse := storage.DeserializeElement(dto)
	if errReverse != nil {
		t.Errorf("failing deserialization %s", errReverse.Error())
	} else if reverse.Id() != entity.Id() {
		t.Fail()
	} else if slices.Compare(entity.Traits(), reverse.Traits()) != 0 {
		t.Fail()
	} else if r, ok := reverse.(*nodes.Entity); !ok {
		t.Error("invalid type found")
	} else {
		reverseEntity = r
	}

	period := reverseEntity.ActivePeriod()
	if !period.IsFullPeriod() {
		t.Fail()
	} else if len(reverseEntity.Attributes()) != 2 {
		t.Fail()
	} else if !reverseEntity.ContainsAttribute("first name") {
		t.Fail()
	} else if !reverseEntity.ContainsAttribute("last name") {
		t.Fail()
	}

	values, errValues := reverseEntity.PeriodValuesForAttribute("first name")
	if errValues != nil {
		t.Fail()
	} else if len(values) != 1 {
		t.Fail()
	} else if value, found := values["Me"]; !found {
		t.Fail()
	} else if !value.IsSameAs(rightPeriod) {
		t.Fail()
	}

	values, errValues = reverseEntity.PeriodValuesForAttribute("last name")
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

func TestMultipleValuesPerAttributeSerde(t *testing.T) {
	attribute_dto := storage.EntityValueDTO{
		AttributeName:  "name",
		AttributeValue: "value",
		Periods:        []string{"[2021-01-01T00:00:00;+oo["},
	}

	other_attribute_dto := storage.EntityValueDTO{
		AttributeName:  "name",
		AttributeValue: "other value",
		Periods:        []string{"]-oo;1999-01-01T00:00:00]"},
	}

	dto := storage.ElementDTO{
		Id:         "id",
		Traits:     []string{},
		Activity:   []string{"]-oo;+oo["},
		Attributes: []storage.EntityValueDTO{attribute_dto, other_attribute_dto},
	}

	element, errElement := storage.DeserializeElement(dto)
	if errElement != nil {
		t.Error(errElement.Error())
	}

	// specifically test if values were stored
	instance, ok := element.(nodes.FormalInstance)
	if !ok {
		t.Fail()
	}

	if values, err := instance.PeriodValuesForAttribute("name"); err != nil {
		t.Fail()
	} else if len(values) != 2 {
		t.Fail()
	} else {
		for value, period := range values {
			switch value {
			case "value":
				if expectedValue, errValue := storage.DeserializePeriodForDTO(attribute_dto.Periods); errValue != nil {
					t.Fail()
				} else if !period.IsSameAs(expectedValue) {
					t.Fail()
				}
			case "other value":
				if expectedValue, errValue := storage.DeserializePeriodForDTO(other_attribute_dto.Periods); errValue != nil {
					t.Fail()
				} else if !period.IsSameAs(expectedValue) {
					t.Fail()
				}
			default:
				t.Fail()
			}
		}
	}

	// and then do it back
	dto_back := storage.SerializeElement(element)
	if dto.Id != dto_back.Id {
		t.Fail()
	} else if slices.Compare(dto.Traits, dto_back.Traits) != 0 {
		t.Fail()
	} else if slices.Compare(dto.Activity, dto_back.Activity) != 0 {
		t.Fail()
	} else if len(dto.Attributes) != len(dto_back.Attributes) {
		t.Fail()
	}

	attr_value_back := dto_back.Attributes[0]
	attr_other_value_back := dto_back.Attributes[1]
	if attr_value_back.AttributeValue == "other value" {
		attr_value_back = dto_back.Attributes[1]
		attr_other_value_back = dto_back.Attributes[0]
	}

	if attribute_dto.AttributeName != attr_value_back.AttributeName {
		t.Fail()
	} else if attribute_dto.AttributeValue != attr_value_back.AttributeValue {
		t.Fail()
	} else if slices.Compare(attribute_dto.Periods, attr_value_back.Periods) != 0 {
		t.Fail()
	}

	if other_attribute_dto.AttributeName != attr_other_value_back.AttributeName {
		t.Fail()
	} else if other_attribute_dto.AttributeValue != attr_other_value_back.AttributeValue {
		t.Fail()
	} else if slices.Compare(other_attribute_dto.Periods, attr_other_value_back.Periods) != 0 {
		t.Fail()
	}
}

func TestRelationSerde(t *testing.T) {
	now := time.Now().UTC().Truncate(1 * time.Second)
	leftInterval := nodes.NewLeftInfiniteTimeInterval(now, false)
	leftPeriod := nodes.NewPeriod(leftInterval)

	links := map[string][]string{
		nodes.SUBJECT_ROLE: {"X"},
		nodes.OBJECT_ROLE:  {"Y"},
	}

	relation := nodes.NewRelationWithIdAndRoles("popo", []string{"Couple"}, links)
	relation.SetActivePeriod(leftPeriod)

	dto := storage.SerializeElement(&relation)
	reverse, ErrReverse := storage.DeserializeElement(dto)
	if ErrReverse != nil {
		t.Errorf("error while deserialize: %s", ErrReverse.Error())
	}

	period := reverse.ActivePeriod()
	if !period.IsSameAs(relation.ActivePeriod()) {
		t.Fail()
	} else if relation.Id() != reverse.Id() {
		t.Fail()
	}

	var reverseRelation *nodes.Relation
	if r, ok := reverse.(*nodes.Relation); !ok {
		t.Error("invalid dto type")
	} else {
		reverseRelation = r
	}

	reverseRoles := reverseRelation.GetValuesPerRole()
	if len(reverseRoles) != len(links) {
		t.Fail()
	}

	for k, v := range links {
		if slices.Compare(v, reverseRoles[k]) != 0 {
			t.Fail()
		}
	}
}
