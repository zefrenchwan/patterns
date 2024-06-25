package storage

import (
	"errors"
	"time"

	"github.com/zefrenchwan/patterns.git/graphs"
	"github.com/zefrenchwan/patterns.git/nodes"
)

const (
	// DATE_SERDE_FORMAT is the format for dates to use in json
	DATE_SERDE_FORMAT = "2006-01-02T15:04:05"
)

// AuthDTO provides resources, given role and class
type AuthDTO struct {
	AuthorizedResources   []string `json:"authorized,omitempty"`
	UnauthorizedResources []string `json:"unauthorized,omitempty"`
	AllAuthorized         bool     `json:"all"`
}

// UserAuthsDTO provides what an user may access
type UserAuthsDTO struct {
	UserId                  string                        `json:"user_id"`
	Login                   string                        `json:"login"`
	ActiveUser              bool                          `json:"active_user"`
	ClassRoleAuthorizations map[string]map[string]AuthDTO `json:"authorizations,omitempty"`
}

// AuthGraphDTO contains graphs an user has access to
type AuthGraphDTO struct {
	Id          string              `json:"id"`
	Name        string              `json:"name"`
	Roles       []string            `json:"roles"`
	Description string              `json:"description"`
	Metadata    map[string][]string `json:"metadata"`
}

// GraphWithElementsDTO is a full graph representation
type GraphWithElementsDTO struct {
	Id          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Metadata    map[string][]string `json:"metadata"`
	Nodes       []GraphNodeDTO      `json:"nodes"`
}

// EquivalenceObjectDTO defines an equivalence entry as a source graph and source element.
// It is easier to understand than a map[element id] graph id
type EquivalenceObjectDTO struct {
	SourceGraph   string `json:"graph"`
	SourceElement string `json:"element"`
}

// GraphNodeDTO represents a DTO for a node in a graph (same structure)
type GraphNodeDTO struct {
	EquivalenceClassGraph []EquivalenceObjectDTO `json:"equivalents,omitempty"`
	SourceGraph           string                 `json:"source"`
	Value                 ElementDTO             `json:"element"`
	Editable              bool                   `json:"editable"`
}

// ElementDTO regroups entity and relation content into a single DTO
type ElementDTO struct {
	Id       string   `json:"id"`
	Traits   []string `json:"traits,omitempty"`
	Activity []string `json:"activity,omitempty"`

	Attributes []EntityValueDTO                  `json:"attributes,omitempty"`
	Roles      map[string][]RelationRoleValueDTO `json:"roles,omitempty"`
}

// ElementDTOSerializer defines general contract for element to element to dto serialiazer
type ElementDTOSerializer func(e nodes.Element) (ElementDTO, error)

// IsEmpty returns true if element dto is non significant
func (e ElementDTO) IsEmpty() bool {
	return len(e.Id) == 0 || (len(e.Attributes) == 0 && len(e.Roles) == 0)
}

// EntityValueDTO is a DTO for entity values
type EntityValueDTO struct {
	AttributeName  string   `json:"attribute"`
	AttributeValue string   `json:"value"`
	Periods        []string `json:"validity,omitempty"`
}

// RelationRoleValueDTO is a single operand in a relation for a given role
type RelationRoleValueDTO struct {
	Operand string   `json:"operand"`
	Periods []string `json:"validity,omitempty"`
}

// SerializePeriodsForDTO returns the serialized period as a slice, one value per interval
func SerializePeriodsForDTO(p nodes.Period) []string {
	return nodes.SerializePeriod(p, DATE_SERDE_FORMAT)
}

// DeserializePeriodForDTO uses DTO date format to deserialize a slice of strings representing a period
func DeserializePeriodForDTO(intervals []string) (nodes.Period, error) {
	return nodes.DeserializePeriod(intervals, DATE_SERDE_FORMAT)
}

// SerializeElement returns the dto content
func SerializeElement(e nodes.Element) (ElementDTO, error) {
	var dto ElementDTO
	dto.Id = e.Id()
	dto.Traits = append(dto.Traits, e.Traits()...)
	dto.Activity = SerializePeriodsForDTO(e.ActivePeriod())

	if relation, ok := e.(nodes.FormalRelation); ok {
		dto.Roles = make(map[string][]RelationRoleValueDTO)
		for role, operands := range relation.PeriodValuesPerRole() {
			values := make([]RelationRoleValueDTO, 0)
			for value, period := range operands {
				values = append(values, RelationRoleValueDTO{
					Operand: value,
					Periods: SerializePeriodsForDTO(period),
				})
			}

			if len(values) != 0 {
				dto.Roles[role] = values
			}
		}
	} else if entity, ok := e.(*nodes.Entity); ok {
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

	return dto, nil
}

// SerializeElementAtMoment returns a serialized element with values set for that moment, no period.
// It is basically a freeze: only take values
func SerializeElementAtMoment(element nodes.Element, moment time.Time) (ElementDTO, error) {
	var result ElementDTO
	if element == nil {
		return result, nil
	}

	activity := element.ActivePeriod()
	if !activity.Contains(moment) {
		return result, nil
	} else {
		result.Id = element.Id()
		result.Traits = element.Traits()
	}

	var globalErr error
	formalInstance, isFormalInstance := element.(nodes.FormalInstance)
	formalRelation, isFormalRelation := element.(nodes.FormalRelation)

	if isFormalInstance {
		for _, attribute := range formalInstance.Attributes() {
			values, errAttr := formalInstance.PeriodValuesForAttribute(attribute)
			if errAttr != nil {
				globalErr = errors.Join(globalErr, errAttr)
				continue
			}

			var matchingValue string
			for value, period := range values {
				if period.Contains(moment) {
					matchingValue = value
					break
				}
			}

			attributeValueDTO := EntityValueDTO{
				AttributeName:  attribute,
				AttributeValue: matchingValue,
			}

			if len(matchingValue) == 0 {
				continue
			} else {
				result.Attributes = append(result.Attributes, attributeValueDTO)
			}
		}
	}

	if globalErr != nil {
		return result, globalErr
	}

	if isFormalRelation {
		values := formalRelation.PeriodValuesPerRole()
		for role, links := range values {
			var matchingLinks []RelationRoleValueDTO
			// remember that many links may match
			for link, period := range links {
				if period.Contains(moment) {
					newLink := RelationRoleValueDTO{
						Operand: link,
					}

					matchingLinks = append(matchingLinks, newLink)
				}
			}

			if len(matchingLinks) != 0 {
				if len(result.Roles) == 0 {
					result.Roles = make(map[string][]RelationRoleValueDTO)
				}

				result.Roles[role] = matchingLinks
			}
		}
	}

	return result, globalErr
}

// SerializerAtMoment builds a dto serializer that serializes data at a given time
func SerializerAtMoment(moment time.Time) ElementDTOSerializer {
	return func(element nodes.Element) (ElementDTO, error) {
		return SerializeElementAtMoment(element, moment)
	}
}

// DeserializeElement returns an element from a dto
func DeserializeElement(dto ElementDTO) (nodes.Element, error) {
	var result nodes.Element
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
		var globalErr error
		relation := nodes.NewRelationWithId(id, dto.Traits)
		for role, values := range roles {
			for _, value := range values {
				if period, err := DeserializePeriodForDTO(value.Periods); err != nil {
					globalErr = errors.Join(globalErr, err)
				} else {
					relation.AddPeriodValueForRole(role, value.Operand, period)
				}
			}
		}

		if err := relation.SetActivePeriod(activity); err != nil {
			globalErr = errors.Join(globalErr, err)
			return result, globalErr
		}

		return &relation, globalErr
	} else {
		entity, errEntity := nodes.NewEntityWithId(id, dto.Traits, activity)
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

// SerializeFullGraph maps a graph (value is huge) to its dto (huge object, going as pointer)
func SerializeFullGraph(g *graphs.Graph, nodeSerializer ElementDTOSerializer) (*GraphWithElementsDTO, error) {
	if g == nil {
		return nil, nil
	}

	result := new(GraphWithElementsDTO)
	// copy graph data
	result.Id = g.Id
	result.Name = g.Name
	result.Description = g.Description
	if len(g.Metadata) != 0 {
		result.Metadata = make(map[string][]string)
		for name, values := range g.Metadata {
			sizeValues := len(values)
			if sizeValues == 0 {
				result.Metadata[name] = nil
			} else {
				copyValues := make([]string, sizeValues)
				copy(copyValues, values)
				result.Metadata[name] = copyValues
			}
		}
	}

	var globalErr error
	// copy each element if significant (not empty) and not error
	for _, node := range g.Nodes() {
		mappedValue, errMapping := nodeSerializer(node.Value)
		if errMapping != nil {
			globalErr = errors.Join(globalErr, errMapping)
			continue
		} else if mappedValue.IsEmpty() {
			continue
		}

		mappedNode := GraphNodeDTO{
			Editable:    node.Editable,
			SourceGraph: node.SourceGraph,
			Value:       mappedValue,
		}

		equivalenceSize := len(node.EquivalenceClass)
		if equivalenceSize > 0 {
			for localElement, localGraph := range node.EquivalenceClass {
				newEquivalentDTO := EquivalenceObjectDTO{
					SourceGraph:   localGraph,
					SourceElement: localElement,
				}

				mappedNode.EquivalenceClassGraph = append(mappedNode.EquivalenceClassGraph, newEquivalentDTO)
			}
		}

		result.Nodes = append(result.Nodes, mappedNode)
	}

	return result, globalErr
}
