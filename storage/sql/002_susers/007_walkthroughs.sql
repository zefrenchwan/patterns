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
	create temporary table if not exists temp_authorized_graphs (
        walkthrough_id text, 
		graph_id text,
		editable bool
    );

    create temporary table if not exists temp_walkthroughs (
        walkthrough_id text, 
        element_id text,
		relation_role text,
        relation_operand text,
        height int 
    );
end; $$;

alter procedure susers.init_walkthrough_structures owner to upa;


-- susers.delete_values_for_walkthrough deletes values for a given walkthrough. 
-- It assumes table temp_walkthroughs exists
create or replace procedure susers.delete_values_for_walkthrough(p_walkthrough_id text) 
language plpgsql as $$
begin
	delete from  temp_authorized_graphs where walkthrough_id = p_walkthrough_id;  
	delete from  temp_walkthroughs where walkthrough_id = p_walkthrough_id; 
end; $$;

alter procedure susers.delete_values_for_walkthrough owner to upa;


-- susers.find_neighbors_for_walkthrough fills walkthrough table to find elements around given values
-- for that walkthrough (provided with id).   
create or replace procedure susers.find_neighbors_for_walkthrough(p_user_login text, p_walkthrough_id text, p_period text) 
language plpgsql as $$
declare
	l_current_height int;
	l_max_previous_height int;
begin 
	-- first hight is 0
	l_current_height = 0;

	insert into temp_authorized_graphs(walkthrough_id, graph_id)
	select p_walkthrough_id, AAG.graph_id, AAG.editable
	from susers.authorized_graphs AAG 
	join susers.users USR on USR.user_id = AAG.auth_user_id 
	where user_login = p_user_login;

	-- delete elements that are NOT visible from said user. 
	-- We keep inactive elements, for user to deal with it. 
	with all_authorized_graphs as (
        select TAG.graph_id
        from temp_authorized_graphs TAG
		where walkthrough_id = p_walkthrough_id
    ), all_valid_elements as (
        select ELT.element_id 
        from temp_walkthroughs TWA 
		join sgraphs.elements ELT on TWA.element_id = ELT.element_id
        join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
    ), all_entities_to_delete as (
		select TWA.element_id 
		from temp_walkthroughs TWA
		left outer join all_valid_elements AVE on AVE.element_id = TWA.element_id 
		where AVE.element_id is null 
	), all_relations_to_delete as (
		select distinct TWA.element_id 
		from temp_walkthroughs TWA
		left outer join all_valid_elements AVE on AVE.element_id = TWA.relation_operand 
		where AVE.element_id is null 
	)
	delete from temp_walkthroughs TWD
	where TWD.element_id in (
		select AED.element_id 
		from all_entities_to_delete
		UNION ALL 
		select ARD.element_id 
		from all_relations_to_delete ARD 
	);


    with all_authorized_graphs as (
        select TAG.graph_id
        from temp_authorized_graphs TAG
		where walkthrough_id = p_walkthrough_id
    ), all_candidates_relations as (
		-- pick relations having one operand in the cleaned base table 
		-- AND that are not already inserted.
		-- This step loads relations which childs are valid entities. Done once.  
		select distinct RRO.relation_id, RRO.role_in_relation as relation_role, 
		RRV.relation_value, RRV.relation_period_id as link_period 
		from sgraphs.relation_role RRO
		join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id
		join temp_walkthroughs TWA on TWA.element_id = RRV.relation_value -- operands are valid
		left outer join temp_walkthroughs EXTWA on EXTWA.element_id = RRO.relation_id
		where EXTWA.element_id is null  -- to find new elements, not already inserted   
		and TWA.walkthrough_id = p_walkthrough_id
		and EXTWA.walkthrough_id = p_walkthrough_id
	), active_relations_with_operands as (
		-- We do: 
		-- NOT ask for operands to be active (for instance, X is the son of Y, and Y may not be active)
		-- BUT WE WANT roles to be active, and relation to be active.  
		-- We won't load recursively content of inactive data.
		-- We also test that relation is visible.  
		select  distinct ARC.relation_id, ARC.relation_role, ARC.relation_value 
		from all_candidates_relations ARC
		join sgraphs.elements ELT on ELT.element_id = ARC.relation_id 
		join sgraphs.periods PERLINK on ARC.link_period = PERLINK.period_id 
		join sgraphs.periods PER on ELT.element_period = PER.period_id 
		-- to restrict to visible graphs
		join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		where not sgraphs.are_periods_disjoin(p_period, PER.period_value)
		and not sgraphs.are_periods_disjoin(p_period, PERLINK.period_value)
	), active_relations_with_visible_operands as (
		select ARWO.relation_id, ARWO.relation_role, ARWO.relation_value  
		from active_relations_with_operands ARWO 
		where ARWO.relation_id not in (
			-- find relations with at least one NON visible operand. 
			-- Test on periods was done before and that was a good idea. 
			select ARWOIN.relation_id
			from active_relations_with_operands ARWOIN
			join sgraphs.elements ELTIN on ELTIN.element_id = ARWOIN.relation_value
			left outer join all_authorized_graphs AAGIN on AAG.graph_id = ELTIN.graph_id 
			where AAGIN.graph_id is null 
		)
	), new_elements_to_insert as (
		select EPR.relation_id, EPR.relation_role, EPR.relation_value 
		from active_relations_with_visible_operands ARWVO 
		join temp_walkthroughs TWA on TWA.element_id = ARWVO.relation_id
		where TWA.element_id is null 
		and TWA.walkthrough_id = p_walkthrough_id
	)
	insert into temp_walkthroughs(walkthrough_id,element_id, relation_role, relation_operand,height)
	select p_walkthrough_id, NETI.relation_id, NETI.relation_role, NETI.relation_value, 0 
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
			select TAG.graph_id
			from temp_authorized_graphs TAG
			where walkthrough_id = p_walkthrough_id
    	), all_relation_operands_starter as (
			-- find childs that are relation, active, to load from 
			select TW.relation_operand
			from temp_walkthroughs TW 
			-- pick only relations
			join sgraphs.relation_role RRO on RRO.relation_id = TW.relation_operand
			join sgraphs.elements ELT on ELT.element_id = RRO.relation_id
			-- find period for that relation  
			join sgraphs.periods PER on PER.period_id = ELT.element_period 
			-- restrict to active relations 
			where not sgraphs.are_periods_disjoin(p_period, PER.period_value)
			and TW.walkthrough_id = p_walkthrough_id
			-- and are inserted last
			and TW.height = l_max_previous_height
		), all_active_candidates_relations as (
			select AROS.relation_operand as relation_id, 
			RRO.role_in_relation as relation_role, RRV.relation_value
			from all_relation_operands_starter AROS 
			join sgraphs.relation_role RRO on AROS.relation_id = RRO.relation_id
			join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id
			join sgraphs.periods PER on PER.period_id = RRV.relation_period_id 
			where not sgraphs.are_periods_disjoin(p_period, PER.period_value)
		), all_active_visible_relations as (
			select AACR.relation_id, AACR.relation_role, AACR.relation_value 
			from all_active_candidates_relations AACR
			where AACR.relation_id not in (
				select AACRIN.relation_id
				from all_active_candidates_relations AACRIN
				join sgraphs.elements ELTIN on ELTIN.element_id = AACRIN.relation_value  
				left outer join all_authorized_graphs AAGIN on AAGIN.graph_id = ELTIN.graph_id
				where AAGIN.graph_id is null 
			)
		), new_visible_relations as (
			-- from visible relations, get only relations that were not inserted
			select AAVR.relation_id, AVR.relation_role, AAVR.relation_value 
			from all_active_visible_relations AAVR 
			left outer join temp_walkthroughs TW on TW.element = VRE.relation_id
			where TW.relation_id is null 
			and TW.walkthrough_id = p_walkthrough_id
		)
		insert into temp_walkthroughs(walkthrough_id,element_id,relation_role, relation_operand,height)
		select p_walkthrough_id, NVIR.relation_id, NVIR.relation_role, NVIR.relation_value, l_current_height 
		from new_visible_relations NVIR; 
	end loop;

end; $$;

alter procedure susers.find_neighbors_for_walkthrough owner to upa;


create or replace function susers.load_relations_from_walkthrough(p_walkthrough_id text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], 
    equivalence_parent text, equivalence_parent_graph text,
    role_in_relation text, role_values text[], role_periods text[]
) language plpgsql as $$
begin 
	with all_authorized_graphs as (
        select TAG.graph_id, TAG.editable 
        from temp_authorized_graphs TAG
		where walkthrough_id = p_walkthrough_id
    ), all_visible_relations as (
		select ELT.element_id, ELT.graph_id, AAG.editable, ELT.element_period as period_id
		from temp_walkthroughs TWA
		join sgraphs.elements ELT on ELT.element_id = TWA.element_id
		join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		where TWA.walkthrough_id = p_walkthrough_id
		and ELT.element_type in (2,10)
		UNION
		select TWA.relation_operand as element_id, 
		ELT.graph_id, AAG.editable, ELT.element_period as period_id
		from temp_walkthroughs TWA
		join sgraphs.elements ELT on ELT.element_id = TWA.relation_operand
		join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		where TWA.walkthrough_id = p_walkthrough_id
		and ELT.element_type in (2,10)
	), all_relations_main_data as (
		select AVR.element_id, AVR.graph_id, AVR.editable, 
		PER.period_value, 
		not susers.are_periods_disjoin(p_period_value, PER.period_value) is_active 
		from all_visible_relations AVR 
		join sgraphs.periods PER on AVR.period_id = PER.period_id
	), all_relations_traits as (
		select ARMD.element_id, array_agg(distinct TRA.trait) as traits 
		from all_relations_main_data ARMD 
		join sgraphs.element_trait ETR on ETR.element_id = ARMD.element_id 
		join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id 
		group by ARMD.element_id
	),  all_equivalences as (
		select ARMD.element_id, 
		ELTS.element_id as equivalence_parent,
		AAG.graph_id as equivalence_parent_graph 
		from all_relations_main_data ARMD
		join sgraphs.nodes NOD on NOD.child_element_id = ARMD.element_id 
		join sgraphs.elements ELTS on ELTS.element_id = NOD.source_element_id 
		join all_authorized_graphs AAG on AAG.graph_id = ELTS.graph_id 
	), all_relation_roles_from_walkthrough as (
		select ELT.element_id, TWA.relation_role, TWA.relation_operand
		from temp_walkthroughs TWA
		join sgraphs.elements ELT on ELT.element_id = TWA.element_id
		join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		where TWA.walkthrough_id = p_walkthrough_id
		and ELT.element_type in (2,10)
		and TWA.relation_role is not null 
	), all_relations_values as (
		select AARFW.element_id, 
		RRV.role_in_relation, 
		array_agg(RRV.relation_value order by RRV.relation_period_id) as role_values, 
		array_agg(PER.period_value) as role_periods
		from all_relation_roles_from_walkthrough ARRFW 
		join sgraphs.relation_role RRO on RRO.relation_id = ARRFW.relation_id
		join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id
		join sgraphs.periods PER on PER.period_id = RRV.relation_period_id 
		where ARRFW.relation_role = RRO.role_in_relation
		and ARRFW.relation_value = RRV.relation_value
		group by AARFW.element_id, RRV.role_in_relation 
	)
	select 
	ARMD.graph_id, ARMD.editable, 
    ARMD.element_id, ARMD.activity, ART.traits, 
    AEQ.equivalence_parent , AEQ.equivalence_parent_graph ,
    ARV.role_in_relation , ARV.role_values , ARV.role_periods 
	from all_relations_main_data ARMD 
	left outer join all_relations_traits ART on ART.element_id = ARMD.element_id
	left outer join all_equivalences AEQ on AEQ.element_id = ARMD.element_id 
	left outer join all_relations_values ARV on ARV.element_id = ARMD.element_id ;
end; $$;

alter function susers.load_relations_from_walkthrough owner to upa;

create or replace function susers.load_entities_from_walkthrough(p_walkthrough_id text, p_period_value text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], 
    equivalence_parent text, equivalence_parent_graph text,
    attribute_key text, attribute_values text[], attribute_periods text[]
) language plpgsql as $$
begin
	return query 
	with all_authorized_graphs as (
        select TAG.graph_id, TAG.editable 
        from temp_authorized_graphs TAG
		where walkthrough_id = p_walkthrough_id
    ), all_visible_entities as (
		select ELT.element_id, ELT.graph_id, AAG.editable, ELT.element_period as period_id
		from temp_walkthroughs TWA
		join sgraphs.elements ELT on ELT.element_id = TWA.element_id
		join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		where TWA.walkthrough_id = p_walkthrough_id
		and ELT.element_type in (1,10)
		UNION
		select TWA.relation_operand as element_id, 
		ELT.graph_id, AAG.editable, ELT.element_period as period_id
		from temp_walkthroughs TWA
		join sgraphs.elements ELT on ELT.element_id = TWA.relation_operand
		join all_authorized_graphs AAG on AAG.graph_id = ELT.graph_id
		where TWA.walkthrough_id = p_walkthrough_id
		and ELT.element_type in (1,10)
	), all_entities_main_data as (
		select AVE.element_id, AVE.graph_id, AVE.editable, 
		PER.period_value, 
		not susers.are_periods_disjoin(p_period_value, PER.period_value) is_active 
		from all_visible_entities AVE 
		join sgraphs.periods PER on AVE.period_id = PER.period_id
	), all_entities_traits as (
		select AEMD.element_id, array_agg(distinct TRA.trait) as traits 
		from all_entities_main_data AEMD 
		join sgraphs.element_trait ETR on ETR.element_id = AEMD.element_id 
		join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id 
		group by AEMD.element_id
	), all_entities_attributes as (
		select AEMD.element_id, EAT.attribute_name, 
		array_agg(EAT.attribute_value order by attribute_id) as attribute_values, 
		array_agg(PER.period_value order by attribute_id) as attribute_periods
		from all_entities_main_data AEMD
		join susers.entity_attributes EAT on EAT.entity_id = AEMD.element_id 
		join susers.periods PER on PER.period_id = EAT.period_id 
		where (
			-- either entity is not active and load it all (X loves socrate)
			-- or entity is active and load only relevant data 
			not AEMD.is_active or susers.are_periods_disjoin(PER.period_value, p_period_value)
		)
		group by AEMD.element_id, EAT.attribute_name
	), all_equivalences as (
		select AEMD.element_id, 
		ELTS.element_id as equivalence_parent,
		AAG.graph_id as equivalence_parent_graph 
		from all_entities_main_data AEMD
		join sgraphs.nodes NOD on NOD.child_element_id = AEMD.element_id 
		join sgraphs.elements ELTS on ELTS.element_id = NOD.source_element_id 
		join all_authorized_graphs AAG on AAG.graph_id = ELTS.graph_id 
	)
	select
	AEMD.graph_id, 
	AEMD.editable, 
	AEMD.element_id, 
	AEMD.period_value as activity,
	AET.traits, 
	AEQ.equivalence_parent, 
	AEQ.equivalence_parent_graph,
    AEA.attribute_key, 
	AEA.attribute_values, 
	AEA.attribute_periods 
	from all_entities_main_data AEMD 
	left outer join all_entities_traits AET on AET.element_id = AEMD.element_id 
	left outer join all_entities_attributes AEA on AEA.element_id = AEMD.element_id 
	left outer join all_equivalences AEQ on AEQ.element_id = AEMD.element_id;
end; $$;

alter function susers.load_entities_from_walkthrough owner to upa;