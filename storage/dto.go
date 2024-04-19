package storage

import (
	"errors"
	"slices"

	"github.com/zefrenchwan/patterns.git/patterns"
)

const (
	// DATE_SERDE_FORMAT is the format for dates to use in json
	DATE_SERDE_FORMAT = "2006-01-02T15:04:05"
)

// EntityDTO is a DTO to deal with entities
type EntityDTO struct {
	Id       string           `json:"id"`
	Traits   []string         `json:"traits"`
	Values   []EntityValueDTO `json:"values"`
	Activity []string         `json:"activity,omitempty"`
}

// EntityValueDTO is a DTO for entity values
type EntityValueDTO struct {
	AttributeName  string   `json:"attribute"`
	AttributeValue string   `json:"value"`
	Periods        []string `json:"vaiidity,omitempty"`
}

// RelationDTO is a DTO for relations
type RelationDTO struct {
	Id       string              `json:"id"`
	Traits   []string            `json:"traits"`
	Activity []string            `json:"activity,omitempty"`
	Roles    map[string][]string `json:"roles,omitempty"`
}

// SerializePeriodsForDTO returns the serialized period as a slice, one value per interval
func SerializePeriodsForDTO(p patterns.Period) []string {
	return patterns.SerializePeriod(p, DATE_SERDE_FORMAT)
}

// DeserializePeriodForDTO uses DTO date format to deserialize a slice of strings representing a period
func DeserializePeriodForDTO(intervals []string) (patterns.Period, error) {
	return patterns.DeserializePeriod(intervals, DATE_SERDE_FORMAT)
}

// SerializeEntity returns the dto from an entity
func SerializeEntity(e patterns.Entity) EntityDTO {
	var dto EntityDTO
	dto.Id = e.Id()
	dto.Traits = append(dto.Traits, e.Traits()...)
	dto.Activity = SerializePeriodsForDTO(e.ActivePeriod())

	for _, attr := range e.Attributes() {
		attributeValues, _ := e.PeriodValuesForAttribute(attr)
		for attributeValue, periodValue := range attributeValues {
			value := EntityValueDTO{
				AttributeName:  attr,
				AttributeValue: attributeValue,
				Periods:        SerializePeriodsForDTO(periodValue),
			}

			dto.Values = append(dto.Values, value)
		}
	}

	return dto
}

// DeserializeEntity reads a dto and parses it to make an entity
func DerializeEntity(dto EntityDTO) (patterns.Entity, error) {
	var result patterns.Entity
	activity, errActive := DeserializePeriodForDTO(dto.Activity)
	if errActive != nil {
		return result, errActive
	} else if newResult, err := patterns.NewEntityWithId(dto.Id, dto.Traits, activity); err != nil {
		return result, err
	} else {
		result = newResult
	}

	if len(dto.Values) == 0 {
		return result, nil
	}

	var globalErr error
	for _, value := range dto.Values {
		if len(value.Periods) == 0 {
			continue
		} else if attrPeriod, errPeriod := DeserializePeriodForDTO(value.Periods); errPeriod != nil {
			globalErr = errors.Join(globalErr, errPeriod)
		} else if attrPeriod.IsEmptyPeriod() {
			continue
		} else if err := result.AddValue(value.AttributeName, value.AttributeValue, attrPeriod); err != nil {
			globalErr = errors.Join(globalErr, errPeriod)
		}
	}

	return result, globalErr
}

// SerializeRelation serializes a relation to a dto (then a json)
func SerializeRelation(r patterns.Relation) RelationDTO {
	var result RelationDTO
	result.Id = r.Id()
	result.Activity = SerializePeriodsForDTO(r.ActivePeriod())
	result.Traits = append(result.Traits, r.Traits()...)

	result.Roles = make(map[string][]string)
	for role, values := range r.GetValuesPerRole() {
		result.Roles[role] = slices.Clone(values)
	}

	return result
}

// DeserializeRelation reads a dto to build a relation
func DeserializeRelation(r RelationDTO) (patterns.Relation, error) {
	var result patterns.Relation

	var period patterns.Period
	if len(r.Activity) == 0 {
		period = patterns.NewEmptyPeriod()
	} else if p, err := DeserializePeriodForDTO(r.Activity); err != nil {
		return result, err
	} else {
		period = p
	}

	result = patterns.NewRelationWithIdAndRoles(r.Id, r.Traits, r.Roles)
	result.SetActivePeriod(period)

	return result, nil
}
