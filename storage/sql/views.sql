

drop view if exists spat.v_active_entites_now;
drop view if exists spat.v_full_relations;
drop view if exists spat.v_full_entities;
drop view if exists spat.v_traitslinks;


create or replace view spat.v_traitslinks as 
with all_linked_elements as (
	select TPAT.pattern_name, TPL.reftype, TTSUB.trait as subtrait, 
	array_agg(TTSUP.trait order by TTSUP.trait)  as supertraits 
	from spat.patterns TPAT
	inner join spat.mdlinks TPL on TPL.pattern_id = TPAT.pattern_id 
	inner join spat.traits TTSUB on TTSUB.trait_id = TPL.subtrait_id
	inner join spat.traits TTSUP on TTSUP.trait_id = TPL.supertrait_id
	group by TPAT.pattern_name, TPL.reftype, TTSUB.trait
), all_unlinked_elements as (
	select  TPAT.pattern_name, TPL.reftype, TTSUP.trait as supertrait, array[]::text[] 
	from spat.patterns TPAT
	inner join spat.mdlinks TPL on TPL.pattern_id = TPAT.pattern_id 
	inner join spat.traits TTSUP on TTSUP.trait_id = TPL.supertrait_id
	where not exists (
		select 1 from spat.mdlinks MDL where MDL.subtrait_id = TTSUP.trait_id
	)
)
select * from all_linked_elements
UNION 
select * from all_unlinked_elements;

alter view spat.v_traitslinks owner to upa;


-- spat.v_full_entities displays all raw data for a given entity
create or replace view spat.v_full_entities as 
with 
all_entities_traits as (
	select ENT.entity_id, array_agg(TRA.trait) as traits
	from spat.entity_trait ENT
	join spat.traits TRA on TRA.trait_id = ENT.trait_id
	group by ENT.entity_id
), 
all_entity_values as (
	select ETA.entity_id, ETA.attribute_name, ETA.attribute_value, 
	PER.period_empty, PER.period_full, PER.period_value 
	from spat.entity_attributes ETA 
	join spat.periods PER on PER.period_id = ETA.period_id
)
select SEN.entity_id, 
EPE.period_empty as entity_period_empty, 
EPE.period_full as entity_period_full, 
EPE.period_value as entity_period_value, 
AET.traits, AEV.attribute_name, AEV.attribute_value, 
AEV.period_empty, AEV.period_full, AEV.period_value
from spat.entities SEN 
join spat.periods EPE on SEN.entity_period = EPE.period_id
left outer join all_entities_traits AET on AET.entity_id = SEN.entity_id 
left outer join all_entity_values AEV on AEV.entity_id = SEN.entity_id;

alter view spat.v_full_entities owner to upa;

-- spat.v_full_relations returns the relations data (period, roles, etc)
create or replace view spat.v_full_relations as 
with 
all_relations_traits as (
	select RTR.relation_id, array_agg(RTA.trait) as traits
	from spat.relation_trait RTR 
	join spat.traits RTA on RTA.trait_id = RTR.trait_id
	group by RTR.relation_id
)
select REL.relation_id, PEREL.period_empty, PEREL.period_full, PEREL.period_value,
ART.traits, RRO.role_in_relation, RRO.role_values
from spat.relations REL 
join spat.periods PEREL on PEREL.period_id = REL.relation_activity
join spat.relation_role RRO ON RRO.relation_id = REL.relation_id 
left outer join all_relations_traits ART on ART.relation_id = REL.relation_id;

alter view spat.v_full_relations owner to upa;
