---------------------------------------------
-- TO PASS AFTER procedures and structures --
---------------------------------------------


-- sgraphs.NeighborsUntilEntities loads elements from an id:
-- only relations from the element
-- only elements in relations with the element
-- or recursively relations over relations raised by previous parts
-- 
-- From a technical point of view, it is a graph walkthrough 
create or replace function sgraphs.NeighborsUntilEntities(p_id text) 
returns table(
    element_id text, element_type integer, element_traits text[], 
    period_full boolean, period_value text, 
    role_in_relation text, role_values text[], 
    attribute_name text, attribute_value text, 
    attribute_period_full boolean, attribute_period_value text
) 
language plpgsql as $$
declare 
	l_walkthrough_id text := gen_random_uuid()::text;
	l_counter int;
	l_current_relation text;
begin

-- create temp table (if necessary) to deal with data walkthroughs
-- drop table if exists temp_walkthroughs;
-- drop table if exists temp_neighbors;
-- drop table if exists temp_relation_childs;

-- next element to process: relation, element in role, and type of element
create temporary table if not exists temp_walkthroughs (
	walkthrough_id text, 
	relation_id text
);

create temporary table if not exists temp_relation_childs (
	walkthrough_id text, 
	relation_id text,
    role_value text, 
    element_type int
);

-- all elements to return per walkthrough
create temporary table if not exists temp_neighbors 
(
	walkthrough_id text,
    element_id text, element_type integer, element_traits text[], 
    period_full boolean, period_value text, 
    role_in_relation text, role_values text[], 
    attribute_name text, attribute_value text, 
    attribute_period_full boolean, attribute_period_value text
);

-----------------------------------------
-- INIT WALKTHROUGH FROM FIRST ELEMENT --
-----------------------------------------

-- insert first element if it is an entity
-- and let the process deal with it if it is a relation
if exists (
    -- p_id matches a relation
    select 1 from sgraphs.elements ELT 
    where ELT.element_id = p_id
    and ELT.element_type in (1,10)
) then 
    -- by definition, did not exist before
    insert into temp_neighbors 
    select l_walkthrough_id as walkthrough_id, NUE.*
    from sgraphs.LoadElement(p_id) NUE ;
else 
    -- it is a relation, then, so add it to process
    insert into temp_walkthroughs(walkthrough_id, relation_id)
    select l_walkthrough_id as walkthrough_id, p_id as relation_id;
end if; 
-- add all relations that contain it, no matter parameter type
with parents_relations as (
	select RRO.relation_id
	from sgraphs.relation_role RRO
	where p_id = ANY(RRO.role_values)
)
insert into temp_walkthroughs(walkthrough_id, relation_id)
select distinct l_walkthrough_id as walkthrough_id, PAR.relation_id
from parents_relations PAR;

-----------------------------------------------
-- START ITERATION OVER WALKTHROUGH ELEMENTS --
-----------------------------------------------
LOOP

    -- PICK FIRST ELEMENT TO PROCESS 
	-- no more element to process means exit
	select count(*) into l_counter 
	from temp_walkthroughs
    where walkthrough_id = l_walkthrough_id;
	
	exit when l_counter = 0;

	-- temp_neighbors has one element at least, pick it and process it
	select WAL.relation_id into l_current_relation
	from temp_walkthroughs WAL
	where walkthrough_id = l_walkthrough_id
	limit 1;
	
	delete from temp_walkthroughs WAL
	where  WAL.walkthrough_id = l_walkthrough_id
	and WAL.relation_id = l_current_relation;


    -- current element to process is a relation for sure. 
    -- But, remember, a relation may be an operand of a relation
    -- So we want to look its parents if any
    -- Then, it has childs, so two case:
    -- * child is entity, just save it
    -- * child is relation, save it and add it for exploration

    if not exists (
        select 1 
        from temp_neighbors TMP 
        where TMP.element_id = l_current_relation
        and walkthrough_id = l_walkthrough_id
    ) then 
        insert into temp_neighbors 
        select l_walkthrough_id as walkthrough_id, NUE.*
        from sgraphs.LoadElement(l_current_relation) NUE ;
    end if ;

    -- save childs of the relation to ease processing
    with relation_childs as (
        select RRO.relation_id, unnest(RRO.role_values) as role_value
        from sgraphs.relation_role RRO
        where relation_id = l_current_relation
    ), relation_typed_childs as (
        -- add type to previous results
        select RCH.relation_id, RCH.role_value, ELT.element_type
        from relation_childs RCH 
        join sgraphs.elements ELT on ELT.element_id = RCH.role_value
    ), new_elements_to_process as (
        select RTC.relation_id, RTC.role_value, RTC.element_type
        from relation_typed_childs RTC
        where not exists (
            select 1
            from temp_neighbors TNE 
            where TNE.element_id = RTC.role_value
            and TNE.walkthrough_id = l_walkthrough_id
        )
    )
    insert into temp_relation_childs(walkthrough_id, relation_id, role_value, element_type)
    select l_walkthrough_id as walkthrough_id, 
    NEP.relation_id, NEP.role_value, NEP.element_type
    from new_elements_to_process NEP;
    
    -- save entities
    with entity_activity as (
        -- find entity id, exact type and period
        select ELT.element_id, ELT.element_type, PER.period_full, PER.period_value
        from temp_relation_childs TRC
        join sgraphs.elements ELT on ELT.element_id = TRC.role_value
        join sgraphs.periods PER on PER.period_id = ELT.element_period
        where TRC.element_type in (1,10)
    ), entity_traits as (
        select ACT.element_id, array_agg(TRA.trait) as traits
        from entity_activity ACT
        join sgraphs.element_trait ETR on ETR.element_id = ACT.element_id 
        join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id
        group by ACT.element_id
    ), entity_attributes as (
        select ACT.element_id, 
        ATR.attribute_name, ATR.attribute_value, 
        PER.period_full, PER.period_value
        from entity_activity ACT
        join sgraphs.entity_attributes ATR on ATR.entity_id = ACT.element_id 
        join sgraphs.periods PER on PER.period_id = ATR.period_id
    )
    insert into temp_neighbors(
        walkthrough_id, element_id, element_type, element_traits, 
        period_full, period_value, role_in_relation, role_values, 
        attribute_name, attribute_value, 
        attribute_period_full, attribute_period_value
    ) 
    select l_walkthrough_id as walkthrough_id,
    ACT.element_id, ACT.element_type, TRA.traits,
    ACT.period_full, ACT.period_value, 
    NULL as role_in_relation, NULL as role_values,
    ENA.attribute_name, ENA.attribute_value, 
    ENA.period_full as attribute_period_full, 
    ENA.period_value as attribute_period_value
    from entity_activity ACT 
    left outer join entity_traits TRA on TRA.element_id = ACT.element_id
    left outer join entity_attributes ENA on ENA.element_id = ACT.element_id;

    -- add relations for a later process 
    with child_relations as (
        select TRC.role_value as relation_id 
        from temp_relation_childs TRC
        where TRC.element_type in (2,10)
        and walkthrough_id = l_walkthrough_id
    ), parent_relations as ( 
        select RRO.relation_id 
        from sgraphs.relation_role RRO
        where l_current_relation = ANY (RRO.role_values)
    ), all_neighbors_relations as (
        select CRE.relation_id 
        from child_relations CRE
        UNION 
        select PRE.relation_id 
        from parent_relations PRE 
    ), all_new_relations as (
        select distinct ANR.relation_id
        from all_neighbors_relations ANR
        where not exists (
            select 1 
            from temp_neighbors TNE
            where TNE.element_id = ANR.relation_id
            and walkthrough_id = l_walkthrough_id
        )
    )
    insert into temp_walkthroughs(walkthrough_id, relation_id)
    select distinct l_walkthrough_id as walkthrough_id, 
    ALR.relation_id 
    from all_new_relations ALR;

    -- then, clean inserted data about relation id 
    delete from temp_relation_childs 
    where walkthrough_id = l_walkthrough_id
    and relation_id = l_current_relation;
END LOOP;

return query
select 
TNE.element_id, TNE.element_type, TNE.element_traits, 
TNE.period_full, TNE.period_value, 
TNE.role_in_relation, TNE.role_values, 
TNE.attribute_name, TNE.attribute_value, 
TNE.attribute_period_full, TNE.attribute_period_value
from temp_neighbors TNE
where TNE.walkthrough_id = l_walkthrough_id
order by TNE.element_id
;

delete from temp_walkthroughs where walkthrough_id = l_walkthrough_id;
delete from temp_neighbors where walkthrough_id = l_walkthrough_id;
delete from temp_relation_childs where walkthrough_id = l_walkthrough_id;
RETURN;

end; $$;