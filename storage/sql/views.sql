drop view if exists spat.v_full_relations;
drop view if exists spat.v_full_entities;

-- spat.v_full_entities displays all raw data for a given entity
create or replace view spat.v_full_entities as 
with 
all_entities_traits as (
	select ELM.element_id as entity_id, array_agg(TRA.trait) as traits
	from spat.elements ELM
	join spat.element_trait ELT on ELT.element_id = ELM.element_id
	join spat.traits TRA on TRA.trait_id = ELT.trait_id
	where ELM.element_type in (1, 10) 
	group by ELM.element_id
), 
all_entity_values as (
	select ETA.entity_id, ETA.attribute_name, ETA.attribute_value, 
	PER.period_empty, PER.period_full, PER.period_value 
	from spat.entity_attributes ETA 
	join spat.periods PER on PER.period_id = ETA.period_id
)
select SEN.element_id as entity_id, 
EPE.period_empty as entity_period_empty, 
EPE.period_full as entity_period_full, 
EPE.period_value as entity_period_value, 
AET.traits, AEV.attribute_name, AEV.attribute_value, 
AEV.period_empty, AEV.period_full, AEV.period_value
from spat.elements SEN 
join spat.periods EPE on SEN.element_period = EPE.period_id
left outer join all_entities_traits AET on AET.entity_id = SEN.element_id 
left outer join all_entity_values AEV on AEV.entity_id = SEN.element_id
where SEN.element_type in (1, 10);

alter view spat.v_full_entities owner to upa;

-- spat.v_full_relations returns the relations data (period, roles, etc)
create or replace view spat.v_full_relations as 
with 
all_relations_traits as (
	select ELM.element_id as relation_id, array_agg(TRA.trait) as traits
	from spat.elements ELM
	join spat.element_trait ELT on ELT.element_id = ELM.element_id
	join spat.traits TRA on TRA.trait_id = ELT.trait_id
	where ELM.element_type in (2, 10) 
	group by ELM.element_id
)
select REL.element_id as relation_id, 
PEREL.period_empty, PEREL.period_full, PEREL.period_value,
ART.traits, RRO.role_in_relation, RRO.role_values
from spat.elements REL 
join spat.periods PEREL on PEREL.period_id = REL.element_period
join spat.relation_role RRO ON RRO.relation_id = REL.element_id 
left outer join all_relations_traits ART on ART.relation_id = REL.element_id
where REL.element_type in (2, 10);

alter view spat.v_full_relations owner to upa;
