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

// RelationalStatstDTO provides, for an id at a given time, the stats about its relationships
type RelationalStatstDTO struct {
	// Trait of the relations
	Trait string `json:"trait,omitempty"`
	// Role of the element within the relations
	Role string `json:"role"`
	// Active is true if the period for the relations is active
	Active bool `json:"active"`
	// Counter is the number of matching relations with this criteria
	Counter int64 `json:"counter"`
}

// ElementDTO regroups entity and relation content into a single DTO
type ElementDTO struct {
	Id       string   `json:"id"`
	Traits   []string `json:"traits,omitempty"`
	Activity []string `json:"activity,omitempty"`

	Attributes []EntityValueDTO    `json:"attributess,omitempty"`
	Roles      map[string][]string `json:"roles,omitempty"`
}

// EntityValueDTO is a DTO for entity values
type EntityValueDTO struct {
	AttributeName  string   `json:"attribute"`
	AttributeValue string   `json:"value"`
	Periods        []string `json:"vaiidity,omitempty"`
}

// IsEntityDTO returns true for an entity
func (e ElementDTO) IsEntityDTO() bool {
	return len(e.Roles) == 0
}

// SerializePeriodsForDTO returns the serialized period as a slice, one value per interval
func SerializePeriodsForDTO(p patterns.Period) []string {
	return patterns.SerializePeriod(p, DATE_SERDE_FORMAT)
}

// DeserializePeriodForDTO uses DTO date format to deserialize a slice of strings representing a period
func DeserializePeriodForDTO(intervals []string) (patterns.Period, error) {
	return patterns.DeserializePeriod(intervals, DATE_SERDE_FORMAT)
}

// SerializeElement returns the dto content
func SerializeElement(e patterns.Element) ElementDTO {
	var dto ElementDTO
	dto.Id = e.Id()
	dto.Traits = append(dto.Traits, e.Traits()...)
	dto.Activity = SerializePeriodsForDTO(e.ActivePeriod())

	if relation, ok := e.(patterns.FormalRelation); ok {
		dto.Roles = make(map[string][]string)
		for role, values := range relation.GetValuesPerRole() {
			dto.Roles[role] = slices.Clone(values)
		}
	} else if entity, ok := e.(*patterns.Entity); ok {
		for _, attr := range entity.Attributes() {
			attributeValues, _ := entity.PeriodValuesForAttribute(attr)
			for attributeValue, periodValue := range attributeValues {
				value := EntityValueDTO{
					AttributeName:  attr,
					AttributeValue: attributeValue,
					Periods:        SerializePeriodsForDTO(periodValue),
				}

				dto.Attributes = append(dto.Attributes, value)
			}
		}
	}

	return dto
}

// DeserializeElement returns an element from a dto
func DeserializeElement(dto ElementDTO) (patterns.Element, error) {
	var result patterns.Element
	if len(dto.Roles) != 0 && len(dto.Attributes) != 0 {
		return result, errors.New("both relation and entity parts. Not supported")
	}

	activity, errActive := DeserializePeriodForDTO(dto.Activity)
	if errActive != nil {
		return result, errActive
	}

	var globalErr error
	id := dto.Id
	roles := dto.Roles
	if len(roles) != 0 {
		relation := patterns.NewRelationWithIdAndRoles(id, dto.Traits, roles)
		if err := relation.SetActivePeriod(activity); err != nil {
			return result, err
		}

		return &relation, nil
	} else {
		entity, errEntity := patterns.NewEntityWithId(id, dto.Traits, activity)
		if errEntity != nil {
			return result, errEntity
		}

		for _, value := range dto.Attributes {
			if len(value.Periods) == 0 {
				continue
			} else if attrPeriod, errPeriod := DeserializePeriodForDTO(value.Periods); errPeriod != nil {
				globalErr = errors.Join(globalErr, errPeriod)
			} else if attrPeriod.IsEmptyPeriod() {
				continue
			} else if err := entity.AddValue(value.AttributeName, value.AttributeValue, attrPeriod); err != nil {
				globalErr = errors.Join(globalErr, errPeriod)
			}
		}

		return &entity, globalErr
	}
}
