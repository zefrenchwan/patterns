package storage

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/zefrenchwan/patterns.git/graphs"
	"github.com/zefrenchwan/patterns.git/nodes"
)

func completeGraphWithEntitiesRows(rowsEntities pgx.Rows, result *graphs.Graph) error {
	var globalErr error
	for rowsEntities.Next() {
		// read data from current line
		var rawEntityAttr []any
		if rawLine, errAttr := rowsEntities.Values(); errAttr != nil {
			globalErr = errors.Join(globalErr, errAttr)
			continue
		} else {
			rawEntityAttr = rawLine
		}

		currentGraphId := rawEntityAttr[0].(string)
		currentGraphEditable := rawEntityAttr[1].(bool)
		elementId := rawEntityAttr[2].(string)
		activity := nodes.NewEmptyPeriod()
		if rawEntityAttr[3] != nil {
			if a, err := deserializePeriod(rawEntityAttr[3].(string)); err != nil {
				globalErr = errors.Join(globalErr, err)
				continue
			} else if a.IsEmptyPeriod() {
				globalErr = errors.Join(globalErr, errors.New("empty period for element"))
				continue
			} else {
				activity = a
			}
		}

		var traits []string
		if rawEntityAttr[4] != nil {
			traits = mapAnyToStringSlice(rawEntityAttr[4])
		}

		var equivalenceParent string
		var equivalenceParentGraph string
		if rawEntityAttr[5] != nil {
			equivalenceParent = rawEntityAttr[5].(string)
		}

		if rawEntityAttr[6] != nil {
			equivalenceParentGraph = rawEntityAttr[6].(string)
		}

		var attributeKey string
		var attributeValues []string
		var attributePeriodValues []string
		if rawEntityAttr[7] != nil {
			attributeKey = rawEntityAttr[7].(string)
		}

		if rawEntityAttr[8] != nil {
			attributeValues = mapAnyToStringSlice(rawEntityAttr[8])
		}

		if rawEntityAttr[9] != nil {
			attributePeriodValues = mapAnyToStringSlice(rawEntityAttr[9])
		}

		periodsError := false
		sizePeriodValues := len(attributePeriodValues)
		attributePeriods := make([]nodes.Period, sizePeriodValues)
		for index, periodValue := range attributePeriodValues {
			if newPeriod, err := deserializePeriod(periodValue); err != nil {
				globalErr = errors.Join(globalErr, err)
				periodsError = true
			} else {
				attributePeriods[index] = newPeriod
			}
		}

		if periodsError {
			continue
		}

		result.AddToFormalInstance(currentGraphId, currentGraphEditable,
			elementId, equivalenceParent, equivalenceParentGraph,
			traits, activity, attributeKey, attributeValues, attributePeriods,
		)
	}

	return globalErr
}

func completeGraphWithRelationsRows(rowsRelation pgx.Rows, result *graphs.Graph) error {
	var globalErr error

	for rowsRelation.Next() {
		// read data from current line
		var rawRelation []any
		if rawLine, errAttr := rowsRelation.Values(); errAttr != nil {
			globalErr = errors.Join(globalErr, errAttr)
			continue
		} else {
			rawRelation = rawLine
		}

		currentGraphId := rawRelation[0].(string)
		currentGraphEditable := rawRelation[1].(bool)
		elementId := rawRelation[2].(string)
		activity := nodes.NewEmptyPeriod()
		if rawRelation[3] != nil {
			if a, err := deserializePeriod(rawRelation[3].(string)); err != nil {
				globalErr = errors.Join(globalErr, err)
				continue
			} else if a.IsEmptyPeriod() {
				globalErr = errors.Join(globalErr, errors.New("empty period for element"))
				continue
			} else {
				activity = a
			}
		}

		var traits []string
		if rawRelation[4] != nil {
			traits = mapAnyToStringSlice(rawRelation[4])
		}

		var equivalenceParent string
		var equivalenceParentGraph string
		if rawRelation[5] != nil {
			equivalenceParent = rawRelation[5].(string)
		}

		if rawRelation[6] != nil {
			equivalenceParentGraph = rawRelation[6].(string)
		}

		var roleName string
		var roleValues []string
		var rolePeriods []string

		if rawRelation[7] != nil {
			roleName = rawRelation[7].(string)
		}

		if rawRelation[8] != nil {
			roleValues = mapAnyToStringSlice(rawRelation[8])
		}

		if len(roleValues) == 0 {
			globalErr = errors.Join(globalErr, errors.New("no value for a role in relation"))
			continue
		}

		if rawRelation[9] != nil {
			rolePeriods = mapAnyToStringSlice(rawRelation[9])
		}

		if len(rolePeriods) == 0 {
			globalErr = errors.Join(globalErr, errors.New("no value for a role in relation"))
			continue
		} else if len(rolePeriods) != len(roleValues) {
			globalErr = errors.Join(globalErr, errors.New("relation values and periods mismatch"))
			continue
		}

		for index := 0; index < len(roleValues); index++ {
			switch rolePeriod, errPeriod := deserializePeriod(rolePeriods[index]); errPeriod {
			case nil:
				result.AddToFormalRelation(currentGraphId, currentGraphEditable,
					elementId, equivalenceParent, equivalenceParentGraph,
					traits, activity, roleName, roleValues[index], rolePeriod)
			default:
				globalErr = errors.Join(globalErr, errPeriod)
			}
		}
	}

	return globalErr
}
