package storage

import (
	"fmt"
)

// queryForEntitiesAtDate returns the query to find entities at given date with given attributes
func queryForEntitiesAtDate(trait string, valuesQuery map[string]string) string {
	base := `
	with all_active_values as (
		select * 
		from spat.activeentitiesvaluesat($1)
	)
	select * 
	from all_active_values AAV
	where '%s' = ANY (AAV.traits)
	`

	query := fmt.Sprintf(base, trait)

	for attribute, value := range valuesQuery {
		query = query + "and attribute_name = '" + attribute + "' and attribute_value = '" + value + "'"
	}

	return query
}
