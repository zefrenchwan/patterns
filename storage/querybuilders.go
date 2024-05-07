package storage

import (
	"fmt"
)

// queryForEntitiesAtDate returns the query to find entities at given date with given attributes
func queryForEntitiesAtDate(trait string, valuesQuery map[string]string) string {
	base := `
	with all_active_values as (
		select * 
		from sgraphs.activeentitiesvaluesat($1)
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

func queryForEntitesTraits(valuesQuery map[string]string) string {
	base := `
	with elements_traits as (
		select ELE.element_id, TRA.trait
		from sgraphs.elements ELE 
		join sgraphs.element_trait ETR on ETR.element_id = ELE.element_id
		join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id
		where ELE.element_type in (1,10)
	), 
	elements_attributes as (
		select EAT.entity_id as element_id, EAT.attribute_name, EAT.attribute_value
		from sgraphs.entity_attributes EAT
	),
	elements_periods as (
		select ELE.element_id, PER.period_full, PER.period_value
		from sgraphs.elements ELE 
		join sgraphs.periods PER on PER.period_id = ELE.element_period
		where ELE.element_type in (1,10)
	),
	elements_match as (
		select 
		EAT.element_id,
		EAT.attribute_name, EAT.attribute_value, 
		PER.period_full, PER.period_value,
		ETR.trait
		from elements_attributes EAT
		join elements_periods PER on PER.element_id = EAT.element_id
		left outer join elements_traits ETR on ETR.element_id = EAT.element_id
		%s
	)
	select distinct EMA.element_id, EMA.period_full, EMA.period_value,
	array_agg(EMA.trait) as traits
	from elements_match EMA 
	group by EMA.element_id, EMA.period_full, EMA.period_value
	`

	whereClause := ""
	index := 0
	for attr, value := range valuesQuery {
		if index == 0 {
			whereClause = "where "
		} else {
			whereClause = whereClause + " and "
		}

		whereClause = whereClause + "attribute_name = '" + attr + "' and "
		whereClause = whereClause + "attribute_value = '" + value + "'"

		index = index + 1
	}

	return fmt.Sprintf(base, whereClause)
}
