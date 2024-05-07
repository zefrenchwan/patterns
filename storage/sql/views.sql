drop view if exists sgraphs.v_full_relations;
drop view if exists sgraphs.v_full_entities;

-- sgraphs.v_full_entities displays all raw data for a given entity
create or replace view sgraphs.v_full_entities as 
with 
all_entities_traits as (
	select ELM.element_id as entity_id, array_agg(TRA.trait) as traits
	from sgraphs.elements ELM
	join sgraphs.element_trait ELT on ELT.element_id = ELM.element_id
	join sgraphs.traits TRA on TRA.trait_id = ELT.trait_id
	where ELM.element_type in (1, 10) 
	group by ELM.element_id
), 
all_entity_values as (
	select ETA.entity_id, ETA.attribute_name, ETA.attribute_value, 
	PER.period_empty, PER.period_full, PER.period_value 
	from sgraphs.entity_attributes ETA 
	join sgraphs.periods PER on PER.period_id = ETA.period_id
)
select SEN.element_id as entity_id, 
EPE.period_empty as entity_period_empty, 
EPE.period_full as entity_period_full, 
EPE.period_value as entity_period_value, 
AET.traits, AEV.attribute_name, AEV.attribute_value, 
AEV.period_empty, AEV.period_full, AEV.period_value
from sgraphs.elements SEN 
join sgraphs.periods EPE on SEN.element_period = EPE.period_id
left outer join all_entities_traits AET on AET.entity_id = SEN.element_id 
left outer join all_entity_values AEV on AEV.entity_id = SEN.element_id
where SEN.element_type in (1, 10);

alter view sgraphs.v_full_entities owner to upa;

-- sgraphs.v_full_relations returns the relations data (period, roles, etc)
create or replace view sgraphs.v_full_relations as 
with 
all_relations_traits as (
	select ELM.element_id as relation_id, array_agg(TRA.trait) as traits
	from sgraphs.elements ELM
	join sgraphs.element_trait ELT on ELT.element_id = ELM.element_id
	join sgraphs.traits TRA on TRA.trait_id = ELT.trait_id
	where ELM.element_type in (2, 10) 
	group by ELM.element_id
)
select REL.element_id as relation_id, 
PEREL.period_empty, PEREL.period_full, PEREL.period_value,
ART.traits, RRO.role_in_relation, RRO.role_values
from sgraphs.elements REL 
join sgraphs.periods PEREL on PEREL.period_id = REL.element_period
join sgraphs.relation_role RRO ON RRO.relation_id = REL.element_id 
left outer join all_relations_traits ART on ART.relation_id = REL.element_id
where REL.element_type in (2, 10);

alter view sgraphs.v_full_relations owner to upa;
