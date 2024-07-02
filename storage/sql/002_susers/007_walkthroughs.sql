-- susers.authorized_graphs gets authorized graphs for all users. 
-- Columns are user id, graph id, and editable set to true for modifiable graphs. 
-- Note that all graphs are visible. 
create or replace view susers.authorized_graphs(auth_user_id, graph_id, editable) as 
with all_source_auths as (
	select
	AUT.auth_user_id, AUT.auth_id, ROL.role_name,
	AUT.auth_all_resources, AUT.auth_inclusion, RAT.resource
	from susers.authorizations AUT
	join susers.classes CLA on AUT.auth_class_id = CLA.class_id
	join susers.roles ROL on ROL.role_id = AUT.auth_role_id
    join susers.users USR on USR.user_id = AUT.auth_user_id
	left outer join susers.resources_authorizations RAT on RAT.auth_id = AUT.auth_id
	where CLA.class_name = 'graph'
    and USR.user_active = true
	and ROL.role_name in ('observer','modifier')
), auths_resources as (
	select 
	ASA.auth_user_id,
	GRA.graph_id, ASA.role_name, ASA.auth_inclusion  
	from all_source_auths ASA 
	cross join sgraphs.graphs GRA
	where ASA.resource is null 
	UNION 
	select 
	ASA.auth_user_id,
	GRA.graph_id, ASA.role_name, ASA.auth_inclusion  
	from all_source_auths ASA 
	join sgraphs.graphs GRA on GRA.graph_id = ASA.resource
	where ASA.resource is not null 
), auths_diff as (
	select ATR.auth_user_id,
	ATR.graph_id, ATR.role_name, array_agg(distinct ATR.auth_inclusion) as auth_inclusion
	from auths_resources ATR
	group by ATR.auth_user_id, ATR.graph_id, ATR.role_name
	having count(*) >= 1
), auth_agg as (
	select ADI.auth_user_id, 
	ADI.graph_id, array_agg(distinct role_name) as role_names
	from auths_diff ADI
	where not (false = ANY(ADI.auth_inclusion))
	group by ADI.auth_user_id, ADI.graph_id
)
select AAG.auth_user_id, 
AAG.graph_id, 
('modifier' = ANY(AAG.role_names)) as editable 
from auth_agg AAG;

alter view susers.authorized_graphs owner to upa;


-- susers.init_walkthrough_structures creates base structure for walkthroughs
create or replace procedure susers.init_walkthrough_structures() 
language plpgsql as $$
declare 
begin 
    create temporaty table if not exists temp_walkthroughs (
        walkthrough_id text, 
        element_id text,
        element_operand text,
        height int 
    );
end; $$;

alter procedure susers.init_walkthrough_structures owner to upa;


-- susers.delete_values_for_walkthrough deletes values for a given walkthrough. 
-- It assumes table temp_walkthroughs exists
create or replace procedure susers.delete_values_for_walkthrough(p_walkthrough_id text) 
language plpgsql as $$
declare 
begin 
	delete from  temp_walkthroughs where walkthrough_id = p_walkthrough_id; 
end; $$;

alter procedure susers.delete_values_for_walkthrough owner to upa;


-- susers.find_neighbors_for_walkthrough fills walkthrough table to find elements around given values
-- for that walkthrough (provided with id).   
create or replace susers.find_neighbors_for_walkthrough(p_user_login, p_walkthrough_id text, p_period text) 
language plpgsql as $$
declare
	l_current_height int;
	l_max_previous_height int;
begin 
	-- first hight is 0
	l_current_height = 0;
	-- include parent relation for elements ONLY ONCE. 
	-- Algorithm is:
	-- find authorized graphs
	-- find visible elements 
	-- find parent relations of visible elements
	-- exclude relation values that are not visible (either relation not visible or operand not visible)
	-- insert remaining relation parents
    with all_authorized_graphs as (
        select AAG.graph_id
        from susers.authorized_graphs AAG 
        join susers.users USR on USR.user_id = AAG.auth_user_id 
        where user_login = p_user_login
    ) all_valid_elements as (
		-- element is valid if its graph is visible
		-- AND this element is active during period parameter 
        select ELT.graph_id, ELT.element_id, 
		case ELT.element_type when 1 then true when 2 then false when 10 then true else false end as is_entity 
        from sgraphs.elements ELT
        join temp_walkthroughs TWA on TWA.element_id = ELT.element_id
        join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		join sgraphs.periods PER on PER.period_id = ELT.element_period
		where not sgraphs.are_periods_disjoin(p_period, PER.period_value)
    ), relations_with_operands_in_elements (
		-- pick relations having one operand in the valid elements, valid during period,
		-- AND that are not already inserted. 
		-- REMEMBER: we only want relations with a child in the valid entities ONLY
		-- (otherwise we would load relations on relations) 
		select  distinct RRO.relation_id, RRV.relation_value 
		from sgraphs.relation_role RRO
		join sgraphs.elements ELT on ELT.element_id = RRO.relation_id 
		join sgraphs.periods PER on PER.period_id = ELT.element_period
		join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id
		join sgraphs.periods PERROL on PERROD.period_id = RRV.relation_period_id
		join all_valid_elements AVE on AVE.element_id = RRV.relation_value
		left outer join all_valid_elements EXAVE on EXAVE.element_id = RRO.relation_id
		where EXAVE.element_id is null  
		and not sgraphs.are_periods_disjoin(p_period, PER.period_value)
		and not sgraphs.are_periods_disjoin(p_period, PERROL.period_value)
		and AVE.is_entity = true -- entity childs only 
	), relations_with_operands_graphs as (
		-- This part is about operands that are visible or not. 
		-- It contains relation id and childs, 
		-- visible relation value is true if they are visible. 
		-- Condition on time for value excludes inactive operands 
		select RWOIE.relation_id, RRV.relation_value, 
		(AAG.graph_id is not null) as visible_relation_value
		from relations_with_operands_in_elements RWOIE 
		join sgraphs.elements ELTCHILD on RWOIE.relation_value = ELTCHILD.element_id
		join sgraphs.periods PERVAL on PERVAL.period_id = ELTCHILD.element_period
		left outer join all_authorized_graphs AAG on AAG.graph_id = ELTCHILD.graph_id 
		-- only restrict to elements that are active at that time 
		where not sgraphs.are_periods_disjoin(p_period, PERVAL.period_value)
	), visible_full_relations as (
		-- get all relations with all visible operands from relations_with_operands_graphs. 
		-- It means that those are all the relations with operands from initial walkthrough
		-- that are active and with active operands, all of them visible (operands and relations)
		select RWOG.relation_id, RWOG.relation_value 
		from relations_with_operands_graphs RWOG
		left outer join relations_with_operands_graphs RW on RW.relation_id = RWOG.relation_id 
		where RW.relation_id is null 
		and RW.visible_relation_value = false
	), new_elements_to_insert as (
		select EPR.relation_id, EPR.relation_value 
		from visible_full_relations VFR 
		join temp_walkthroughs TWA on TWA.element_id = VFR.relation_id
		where TWA.element_id is null 
		and TWA.walkthrough_id = p_walkthrough_id
	)
	insert into temp_walkthroughs(walkthrough_id,element_id,element_operand,height)
	select p_walkthrough_id, NETI.relation_id, NETI.relation_value, 0 
	from new_elements_to_insert NETI;
	-- force all hights to 0 for begining nodes
    update temp_walkthroughs
    set height = 0
    where walkthrough_id = p_walkthrough_id;

	-- then, add all relations and linked elements from visible graphs, 
	-- that are childs of relations in previous walk. 
	loop
		-- test if no data was previously inserted 
		select max(TW.height) into l_max_previous_height
		from temp_walkthroughs TW 
		where walkthrough_id = p_walkthrough_id;
		-- exit when last walk did not insert more data
		exit when l_max_previous_height < l_current_height;
		select l_current_height + 1 into  l_current_height ;

		with all_authorized_graphs as (
			select AAG.graph_id
			from susers.authorized_graphs AAG 
			join susers.users USR on USR.user_id = AAG.auth_user_id 
			where user_login = p_user_login
    	), all_relation_candidates as (
			-- get all relations that are active during period, in a visible graph, 
			-- from previous iteration
			select TW.element_operand as relation_id, RRV.relation_value
			from temp_walkthroughs TW 
			join sgraphs.relation_role RRO on RRO.relation_id = TW.element_operand
			join sgraphs.relation_role_value RRV on RRV.relation_role_id = RRO.relation_role_id
			join sgraphs.periods PERROL on PERROL.period_id = RRV.relation_period_id
			join sgraphs.elements ELT on ELT.element_id = TW.element_operand
			join sgraphs.periods PER on PER.period_id = ELT.element_period
			join all_authorized_graphs AGR on ELT.graph_id = AGR.graph_id -- element is visible
			where walkthrough_id = p_walkthrough_id -- for that walkthrough
			and element_operand is not null  -- child of relation, not relation 
			and TW.height = (l_current_height - 1) -- coming from previous run
			and not sgraphs.are_periods_disjoin(p_period, PER.period_value) -- active during period
			and not sgraphs.are_periods_disjoin(p_period, PERROL.period_value)
		), all_relations_operands (
			-- from valid and visible relations, now we get their operands. 
			-- Even if relation is valid, some operands may not be visible
			select ARC.relation_id, ARC.relation_value, 
			(AGR.graph_id is not null) as visible_operand 
			from all_relation_candidates ARC
			join sgraphs.elements ELT on ELT.element_id = ARC.relation_value 
			join sgraphs.periods PER on PER.period_id = ELT.element_period 
			left outer join  all_authorized_graphs AGR on ELT.graph_id = AGR.graph_id
			where not sgraphs.are_periods_disjoin(p_period, PER.period_value)
		), visible_relations as (
			-- visible relations define relations with all operands in visible graphs
			select ARO.relation_id, ARO.relation_value 
			from all_relations_operands ARO 
			join all_relations_operands AROEX on AROEX.relation_id = ARO.relation_id 
			where AROEX.relation_id  is null 
			and AROEX.visible_operand = false
		), new_visible_relations as (
			-- from visible relations, get only relations that were not inserted
			select VRE.relation_id, VRE.relation_value 
			from visible_relations VRE 
			left outer join temp_walkthroughs TW on TW.element = VRE.relation_id
			where TW.relation_id is null 
		)
		insert into temp_walkthroughs(walkthrough_id,element_id,element_operand,height)
		select p_walkthrough_id, NVIR.relation_id, NVIR.relation_value, l_current_height 
		from new_visible_relations NVIR; 
	end loop;

end; $$;

alter procedure susers.find_neighbors_for_walkthrough owner to upa;