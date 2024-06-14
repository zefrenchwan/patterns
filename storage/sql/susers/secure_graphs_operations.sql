-- susers.create_graph_from_scratch constructs a graph
create or replace procedure susers.create_graph_from_scratch(
	p_user in text, p_new_id text, p_name in text, p_description in text
) language plpgsql as $$
declare 
    l_global_auth text[];
    l_resource_auth text[];
    l_auth bool;
    l_current_graph text;
begin
    if exists (
        select 1 from sgraphs.graphs where graph_id = p_new_id
    ) then 
        raise exception 'graph already exists';
    end if;

    call susers.accept_any_user_access_to_resource_or_raise(p_user, 'graph', array['manager'], null);
    -- create graph
    call sgraphs.create_graph(p_new_id, p_name, p_description);
    -- grant graph access
    call susers.add_auth_for_user_on_resource(p_user, 'observer','graph', p_new_id);
    call susers.add_auth_for_user_on_resource(p_user, 'modifier','graph', p_new_id);
end; $$;

alter procedure susers.create_graph_from_scratch owner to upa;

-- susers.create_graph_from_imports builds a graph as the import of others
create or replace procedure susers.create_graph_from_imports(
	p_user in text, p_new_id text, p_name in text, p_description in text, p_imported_graphs in text[]
) language plpgsql as $$
declare 
    l_global_auth text[];
    l_resource_auth text[];
    l_auth bool;
    l_current_graph text;
begin
    if exists (select 1 from sgraphs.graphs where graph_id = p_new_id) then 
        raise exception 'graph already exists';
    end if;

    if array_length(p_imported_graphs, 1) = 0 then 
        raise exception 'need to import at least one graph';
    end if;

    -- Security operations:
    call susers.accept_any_user_access_to_resource_or_raise(p_user, 'graph', array['manager'], null);
    -- test if graph exists and if user may see it
    foreach l_current_graph in array p_imported_graphs loop 
        if not exists (
            select 1 from sgraphs.graphs where graph_id = l_current_graph 
        ) then 
            raise exception 'graph % does not exist', l_current_graph;
        end if;

        call susers.accept_any_user_access_to_resource_or_raise(p_user, 'graph', array['modifier', 'observer'], l_current_graph);
    end loop;

    -- create graph
    call sgraphs.create_graph_from_imports(p_new_id, p_name, p_description, p_imported_graphs);

    -- grant graph access
    call susers.add_auth_for_user_on_resource(p_user, 'observer','graph', p_new_id);
    call susers.add_auth_for_user_on_resource(p_user, 'modifier','graph', p_new_id);

end; $$;

alter procedure susers.create_graph_from_imports owner to upa;

create or replace procedure susers.clear_graph_metadata(p_actor text, p_graph_id text)
language plpgsql as $$
declare 

begin 
    call susers.accept_any_user_access_to_resource_or_raise(p_actor, 'graph', array['modifier'], p_graph_id);
    call sgraphs.clear_graph_metadata(p_graph_id);
end; $$;

alter procedure susers.clear_graph_metadata owner to upa;

create or replace procedure susers.upsert_graph_metadata_entry(p_actor text, p_graph_id text, p_key text, p_values text[]) 
language plpgsql as $$
declare 
begin 
    call susers.accept_any_user_access_to_resource_or_raise(p_actor, 'graph', array['modifier'], p_graph_id);
    call sgraphs.upsert_graph_metadata_entry(p_graph_id, p_key, p_values);
end; $$;

alter procedure susers.upsert_graph_metadata_entry owner to upa;

-- susers.list_graphs_for_user returns the graph data an user may use with user's roles
create or replace function susers.list_graphs_for_user(p_user text) 
returns table (
    graph_id text, graph_roles text[],
    graph_name text, graph_description text, 
    graph_md_key text, graph_md_values text[]
) language plpgsql as $$
declare 
    l_roles text[];
begin

   select array_agg(role_name) into l_roles
    from susers.roles;

    return query
    select 
    GRA.graph_id, GRO.role_names as graph_roles,
    GRA.graph_name, GRA.graph_description,
    ENT.entry_key as graph_md_key, ENT.entry_values as graph_md_values
    from sgraphs.graphs GRA
    join susers.list_authorized_graphs_for_any_roles(p_user, l_roles) GRO ON GRO.graph_id = GRA.graph_id
    left outer join sgraphs.graph_entries ENT on ENT.graph_id = GRA.graph_id;
end; $$;

alter function susers.list_graphs_for_user owner to upa;

-- susers.load_graph_metadata returns the name, description and associated map of a graph, if user is authorized
create or replace function susers.load_graph_metadata(p_user_login text, p_id text)
returns table (graph_name text, graph_description text, entry_key text, entry_values text[])
language plpgsql as $$
declare 
begin 
	call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['manager','observer','modifier'], p_id);
	return query select * from sgraphs.load_graph_metadata(p_id);
end; $$;

alter function susers.load_graph_metadata owner to upa;


-- susers.transitive_visible_graphs_since gets all dependent graphs and roles from a graph 
create or replace function susers.transitive_visible_graphs_since(p_user_login text, p_id text)
returns table (graph_id text, graph_roles text[]) language plpgsql as $$
declare
    l_counter int;
    l_user_id text;
    l_auth text[];
    l_walkthrough_id text;
    l_height int;
begin 
    -- test if user exists. If not, return null, null
    select user_id into l_user_id from susers.users where user_active and user_login = p_user_login;
    if l_user_id is null then 
        return query select null, null;
        return;
    end if;
    -- graph may not exist
    if not exists (
        select 1 from sgraphs.graphs GRA where GRA.graph_id = p_id
    ) then 
        return query select null, null;
        return;
    end if;

    -- user exists and is active, graph exists. 
    -- test if bootstrap graph is authorized
    select array_agg(distinct role_name) into l_auth
    from susers.authorizations AUT
    join susers.roles ROL on ROL.role_id = AUT.auth_role_id
    join susers.classes CLA on CLA.class_id = AUT.auth_class_id
    where AUT.auth_active = true
    AND CLA.class_name = 'graph'
    AND AUT.auth_user_id = l_user_id
    and (AUT.auth_resource is null or AUT.auth_resource = p_id);
    -- return id of the graph, null if first graph is not authorized
    if not ('observer' =ANY(l_auth) or 'modifier' =ANY(l_auth)) then 
        return query select p_id, null;
        return;
    end if;

    -- principle is: 
    -- inclusions table contains source (the parent graph) and its childs (the depenndant graphs). 
    -- So, starting from a graph, we consider it as the child and move back, as much as possible, 
    -- through its parents again and again until no more data is inserted

    -- creates walkthrough table if it does not exist
    create temporary table if not exists 
    temp_table_graphs_imports(walkthrough_id text, height int, graph_id text, roles text[]);
    -- id for walkthrough    
    select gen_random_uuid()::text into l_walkthrough_id;
    -- height of the values 
    l_height = 1;
    -- insert first value: the current graph 
    insert into temp_table_graphs_imports(walkthrough_id, height, graph_id, roles)
    select l_walkthrough_id, l_height, p_id, l_auth;

    loop 

        with all_current_childs as (
            select TTG1.graph_id 
            from temp_table_graphs_imports TTG1
            where walkthrough_id = l_walkthrough_id 
            and TTG1.height <= l_height
            and (
                'modifier' = ANY(TTG1.roles) or 
                'observer' = ANY(TTG1.roles)
            )
        ), all_parents as (
            select INC.source_id as graph_id
            from sgraphs.inclusions INC 
            join all_current_childs ACC on ACC.graph_id = INC.child_id 
        ), all_new_parents as (
            select APA.graph_id 
            from all_parents APA 
            left outer join all_current_childs ACC on ACC.graph_id = APA.graph_id 
            where ACC.graph_id is null 
        ), all_new_parents_extended_roles as (
            select ANP.graph_id, ROL.role_name 
            from susers.authorizations AUT
            join all_new_parents ANP on AUT.auth_resource = ANP.graph_id
            join susers.roles ROL on ROL.role_id = AUT.auth_role_id
            join susers.classes CLA on CLA.class_id = AUT.auth_class_id
            where AUT.auth_active = true
            and CLA.class_name = 'graph'
            and AUT.auth_user_id = l_user_id
            UNION ALL 
            select ANP.graph_id, ROL.role_name 
            from susers.authorizations AUT
            join susers.roles ROL on ROL.role_id = AUT.auth_role_id
            join susers.classes CLA on CLA.class_id = AUT.auth_class_id
            cross join all_new_parents ANP
            where AUT.auth_active = true
            and CLA.class_name = 'graph'
            and AUT.auth_user_id = l_user_id
            and AUT.auth_resource is null 
        ), all_new_parents_roles as (
            select ANPER.graph_id, array_agg(distinct ANPER.role_name) as roles 
            from all_new_parents_extended_roles ANPER
            group by ANPER.graph_id
        ), all_auth_parents as (
            select distinct ANPR.graph_id, ANPR.roles
            from all_new_parents_roles ANPR
            where  (
                'modifier' = ANY(ANPR.roles) or 
                'observer' = ANY(ANPR.roles)
            )
        )
        insert into temp_table_graphs_imports(walkthrough_id, height, graph_id, roles)
        select distinct l_walkthrough_id, l_height + 1, AAP.graph_id, AAP.roles
        from all_auth_parents AAP;

        -- test if we inserted something
        if not exists (
            select 1 
            from temp_table_graphs_imports
            where walkthrough_id = l_walkthrough_id
            and height = l_height + 1
        ) then 
            -- no now element inserted, just stop 
            exit;
        else
            select l_height + 1 into l_height ;
        end if;
    end loop;

    return query 
    select distinct TTGI.graph_id, TTGI.roles
    from temp_table_graphs_imports TTGI
    where TTGI.walkthrough_id = l_walkthrough_id;

    delete from temp_table_graphs_imports
    where walkthrough_id = l_walkthrough_id;
    return;
end; $$;

alter function susers.transitive_visible_graphs_since owner to upa;


-- susers.transitive_load_base_elements_in_graph loads all visible elements from a graph to all its dependencies
create or replace function susers.transitive_load_base_elements_in_graph(p_user_login text, p_id text)
returns table (
    graph_id text, editable bool, 
    element_id text, element_type int, activity text, traits text[], 
    equivalence_class text[], equivalence_class_graph text[]
) language plpgsql as $$
declare
begin 
    call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['observer','modifier'], p_id);
    
    return query
    with all_source_graphs as (
        select 
        TVGS.graph_id, 
        ('modifier' = ANY(TVGS.graph_roles)) as editable
        from susers.transitive_visible_graphs_since(p_user_login, p_id) TVGS 
    ), all_elements_in_graphs as (
        select 
        ELT.graph_id, 
        ASG.editable,
        ELT.element_id,
        ELT.element_type,
        sgraphs.serialize_period(PER.period_full, PER.period_empty, PER.period_value) as activity
        from sgraphs.elements ELT  
        join all_source_graphs ASG on ASG.graph_id = ELT.graph_id
        join sgraphs.periods PER on PER.period_id = ELT.element_period
    ), all_traits_for_elements as (
        select AEG.element_id, array_agg(TRA.trait) as traits 
        from all_elements_in_graphs AEG
        join sgraphs.element_trait ETR on AEG.element_id = ETR.element_id
        join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id 
        group by AEG.element_id
    ), all_accessible_equivalences as (
        select NOD.source_element_id as element_id, 
        array_agg(NOD.child_element_id order by AGR2.graph_id) as equivalence_class,
        array_agg(AGR2.graph_id order by AGR2.graph_id) as equivalence_class_graph
        from sgraphs.nodes NOD
        -- to ensure the source graph is visible. We take all source graphs, not only current
        join all_elements_in_graphs AGR1 on AGR1.element_id = NOD.source_element_id
        join all_elements_in_graphs AGR2 on AGR2.element_id = NOD.child_element_id
        group by NOD.source_element_id
    )
    select 
    AIG.graph_id, 
    AIG.editable,
    AIG.element_id, 
    AIG.element_type,
    AIG.activity,
    ATE.traits,
    AAE.equivalence_class,
    AAE.equivalence_class_graph
    from all_elements_in_graphs AIG 
    left outer join all_traits_for_elements ATE on ATE.element_id = AIG.element_id
    left outer join all_accessible_equivalences AAE on AAE.element_id = AIG.element_id;

end;$$;

alter function susers.transitive_load_base_elements_in_graph owner to upa;


-- susers.transitive_load_entities_in_graph gets all entities an user may use from a graph
create or replace function susers.transitive_load_entities_in_graph(p_user_login text, p_id text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], 
    equivalence_class text[], equivalence_class_graph text[],
    attribute_key text, attribute_values text[], attribute_periods text[]
) language plpgsql as $$
declare
begin 
    call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['observer','modifier'], p_id);
    return query
    with all_source_entities as (
        select TLB.graph_id, TLB.editable, 
        TLB.element_id, TLB.activity, TLB.traits, 
        TLB.equivalence_class, TLB.equivalence_class_graph
        from susers.transitive_load_base_elements_in_graph(p_user_login, p_id) TLB
        where TLB.element_type in (1,10)
    ), all_entities as (
        select ETA.entity_id as element_id, ETA.attribute_name as attribute_key,  
        array_agg(ETA.attribute_value order by ETA.period_id) as attribute_values,
        array_agg(sgraphs.serialize_period(PER.period_full, PER.period_empty, PER.period_value) order by PER.period_id) as attribute_periods
        from sgraphs.entity_attributes ETA 
        join all_source_entities ASE on ETA.entity_id = ASE.element_id 
        join sgraphs.periods PER on PER.period_id = ETA.period_id 
        group by ETA.entity_id, ETA.attribute_name
    )
    select 
    ASE.graph_id, 
    ASE.editable,
    ASE.element_id, 
    ASE.activity,
    ASE.traits,
    ASE.equivalence_class,
    ASE.equivalence_class_graph,
    ALE.attribute_key, 
    ALE.attribute_values,
    ALE.attribute_periods
    from  all_source_entities ASE 
    left outer join all_entities ALE on ALE.element_id = ASE.element_id;
end; $$;

alter function susers.transitive_load_entities_in_graph owner to upa;


create or replace function susers.transitive_load_relations_in_graph(p_user_login text, p_id text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], 
    equivalence_class text[], equivalence_class_graph text[],
    role_in_relation text, role_values text[]
) language plpgsql as $$
declare
begin 
    call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['observer','modifier'], p_id);
    return query
    with all_visible_graphs as (
        select TGS.graph_id
        from susers.transitive_visible_graphs_since(p_user_login, p_id) TGS
    ), all_source_elements as (
        select TLB.graph_id, TLB.editable, 
        TLB.element_id, TLB.activity, TLB.traits, 
        TLB.equivalence_class, TLB.equivalence_class_graph
        from susers.transitive_load_base_elements_in_graph(p_user_login, p_id) TLB
        where TLB.element_type in (2,10)
    ), all_relations as (
        select RRO.relation_id, 
        RRO.role_in_relation, 
        array_agg(RRV.relation_value) as role_values
        from sgraphs.relation_role RRO
        join all_source_elements ASE on ASE.element_id = RRO.relation_id
        join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id
        group by RRO.relation_id, RRO.role_in_relation 
    ), all_expanded_relations as (
        select ALR.relation_id, 
        unnest(ALR.role_values) as role_value 
        from all_relations ALR
    ), all_unauthorized_relations as (
        select distinct AER.relation_id
        from all_expanded_relations AER
        join sgraphs.elements ELT on ELT.element_id = AER.role_value
        left outer join all_visible_graphs AGR on AGR.graph_id = ELT.graph_id
        where AGR.graph_id is null 
    ), all_visible_relations as (
        select distinct ALR.relation_id 
        from all_relations ALR 
        left outer join all_unauthorized_relations AUR on AUR.relation_id = ALR.relation_id
        where AUR.relation_id is null
    )
    select distinct
    ASE.graph_id, 
    ASE.editable,
    ASE.element_id, 
    ASE.activity,
    ASE.traits,
    ASE.equivalence_class,
    ASE.equivalence_class_graph,
    ALR.role_in_relation, 
    ALR.role_values
    from all_source_elements ASE 
    join all_relations ALR on ALR.relation_id = ASE.element_id
    join all_visible_relations AVR on ALR.relation_id = AVR.relation_id;
end; $$;

alter function susers.transitive_load_relations_in_graph owner to upa;


-- susers.upsert_element_in_graph upserts an element from a graph, may raise an exception for auth
create or replace procedure susers.upsert_element_in_graph(
    p_user_login text,
	p_graph_id in text, 
	p_element_id in text, 
	p_element_type int, 
	p_activity in text,
	p_traits in text[]
) language plpgsql as $$
declare 
    l_graph_id text;
begin 
    call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], p_graph_id);
    
    select ELT.graph_id into l_graph_id
    from sgraphs.elements ELT 
    where ELT.element_id = p_element_id;
    if l_graph_id is not null and l_graph_id <> p_graph_id then 
        raise exception 'element graph and graph parameter mismatch';
    end if;

    call sgraphs.upsert_element_in_graph(p_graph_id, p_element_id, p_element_type, p_activity, p_traits); 
end; $$;

alter procedure susers.upsert_element_in_graph owner to upa;


-- susers.upsert_attributes performs a secure upsert on attributes
create or replace procedure susers.upsert_attributes(p_user_login text, p_id text, p_name text, p_values text[], p_periods text[])
language plpgsql as $$
declare 
    l_graph_id text;
begin 
    select ELT.graph_id into l_graph_id
    from sgraphs.elements ELT
    where element_id = p_id;

    if l_graph_id is null then 
        raise exception 'no element matching id %', p_id;
    end if;

    call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], l_graph_id); 
    call sgraphs.upsert_attributes(p_id, p_name, p_values, p_periods);

end; $$;

alter procedure susers.upsert_attributes owner to upa;


-- susers.upsert_links upserts links if user has access to all underlying graphs
create or replace procedure susers.upsert_links(p_user_login text, p_role_id text, p_role_name text, p_operands text[])
language plpgsql as $$
declare 
    l_graph_id text;
    l_all_auth bool;
begin 

    select ELT.graph_id into l_graph_id
    from sgraphs.elements ELT
    where ELT.element_id = p_role_id;

    if l_graph_id is null then 
        raise exception 'no element matching id %', p_role_id;
    end if;

    call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], l_graph_id); 


    with extended_relation_roles as (
        select unnest(p_operands) as relation_operand
    ),
    visible_graphs as (
        select TVGS.graph_id
        from susers.transitive_visible_graphs_since(p_user_login, l_graph_id) TVGS 
    ), 
    links_graphs as (
        select distinct 
        EXR.relation_operand, VIS.graph_id
        from extended_relation_roles EXR
        join sgraphs.elements ELT on ELT.element_id = EXR.relation_operand
        left outer join visible_graphs VIS on ELT.graph_id = VIS.graph_id
     )
    select (count(*) = 0) into l_all_auth
    from links_graphs LIG
    where LIG.graph_id is null;

    if not l_all_auth then 
        raise exception 'auth failure: missing auth for linked elements graphs';
    else
        call sgraphs.upsert_links(p_role_id, p_role_name, p_operands);
    end if;
end;$$;

alter procedure susers.upsert_links owner to upa;

create or replace function susers.load_element_by_id(p_user_login text, p_element_id text)
returns table (
	element_id text,
	traits text[], activity text,
	role_name text, role_values text[], 
	attribute_name text, attribute_values text[], attribute_periods text[]) 
language plpgsql as $$
declare 
	l_element_type int;
	l_graph_id text;
begin
	
select GRA.graph_id into l_graph_id 
from sgraphs.elements ELT 
join sgraphs.graphs GRA on ELT.graph_id = GRA.graph_id;

if l_graph_id is null then 
    -- just returns empty
	return query select null, null, null, null, null, null, null, null where 1 <> 1;
    return;
end if;

call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier','observer'], l_graph_id);

return query
with element_data as (
	select ELT.element_id,
	array_agg(TRA.trait) as traits,
 	max(sgraphs.serialize_period(PER.period_full, false, PER.period_value)) as activity  
	from sgraphs.elements ELT 
	join sgraphs.periods PER on PER.period_id = ELT.element_period
	join sgraphs.element_trait ETR on ETR.element_id = ELT.element_id
	join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id
	where ELT.element_id = p_element_id
	group by ELT.element_id
), element_roles as (
	select RRO.relation_id as element_id, 
	RRO.role_in_relation as role_name , 
	array_agg(RRV.relation_value order by RRV.relation_value) as role_values 
	from sgraphs.relation_role RRO 
	join sgraphs.relation_role_values RRV on RRO.relation_role_id = RRV.relation_role_id
	where RRO.relation_id = p_element_id
	group by RRO.relation_id, RRO.role_in_relation
), element_entity as (
	select ENA.entity_id as element_id,
	ENA.attribute_name, 
	array_agg(ENA.attribute_value order by ENA.attribute_id) as attribute_values,  
	array_agg(sgraphs.serialize_period(PER.period_full, false, PER.period_value) order by ENA.attribute_id) as attribute_periods
	from sgraphs.entity_attributes ENA 
	join sgraphs.periods PER on PER.period_id = ENA.period_id
	where not PER.period_empty
	and ENA.entity_id = p_element_id
	group by ENA.entity_id, ENA.attribute_name
)
select 
EDA.element_id,
EDA.traits,	
EDA.activity,
ERO.role_name,
ERO.role_values,
ELE.attribute_name, 
ELE.attribute_values, 
ELE.attribute_periods
from element_data EDA 
left outer join element_roles ERO on ERO.element_id = EDA.element_id 
left outer join element_entity ELE on ELE.element_id = EDA.element_id;
end;$$;

alter function susers.load_element_by_id owner to upa;

-- susers.delete_element deletes an element if it does not appear in a relation as a parameter
create or replace procedure susers.delete_element(p_user_login text, p_element_id text)
language plpgsql as $$
declare 
	l_graph_id text;
begin 
	select ELT.graph_id into l_graph_id 
	from sgraphs.elements ELT 
	where ELT.element_id = p_element_id;

	if l_graph_id is null then 
		-- no element, no action 
		return;
	end if;

	-- user may not modify graph 
	call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], l_graph_id);

	if exists (
		select 1
		from sgraphs.relation_role_values RRV
		where p_element_id = RRV.relation_value
	) then 
		raise exception 'a relation depends on current element to delete';
	end if;

	-- ok to delete 
	delete from sgraphs.relation_role where relation_id = p_element_id;
	delete from sgraphs.entity_attributes where entity_id = p_element_id;
	delete from sgraphs.elements where element_id = p_element_id;
end; $$;

alter procedure susers.delete_element owner to upa;

-- susers.delete_graph deletes a graph if no relation depends on an element within that graph
create or replace procedure susers.delete_graph(p_user_login text, p_graph_id text)
language plpgsql as $$
declare 
	l_counter int;
	l_graph_id text;
begin 
	select GRA.graph_id into l_graph_id 
	from sgraphs.graphs GRA 
	where GRA.graph_id = p_graph_id;

	if l_graph_id is null then 
		-- no matching id, no action 
		return;
	end if;

	-- user may not modify graph 
	call susers.accept_any_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['manager'], l_graph_id);

	with all_sources as (
		select ELT.element_id 
		from sgraphs.elements ELT 
		where ELT.graph_id = p_graph_id
	), all_external_operands as (
		select RRV.relation_value as role_operands  
		from sgraphs.relation_role RRO
        join sgraphs.elements ELT on ELT.element_id = RRO.relation_id
        join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id 
        where ELT.graph_id <> l_graph_id
	), all_dependencies as (
		select count(*) as counter
		from all_external_operands ALO 
		join all_sources ALS on ALO.role_operands = ALS.element_id
	) select counter into l_counter from all_dependencies;
		
	if l_counter > 0 then 
		raise exception 'a relation outside the graph depends on an element in the graph';
	end if;

	-- ok to delete 
	delete from sgraphs.elements where graph_id = p_graph_id;
    delete from sgraphs.graphs where graph_id = p_graph_id;
end; $$;

alter procedure susers.delete_graph owner to upa;

-- susers.clear_graphs clear all data in graphs schema.
create or replace procedure susers.clear_graphs(p_user_login text)
language plpgsql as $$
declare
	l_auth bool;
begin
	
with aggregated_roles_over_graphs as (
	select array_agg(ROL.role_name) as active_roles
	from susers.roles ROL
	join susers.authorizations AUT on AUT.auth_role_id = ROL.role_id
	join susers.classes CLA on CLA.class_id = AUT.auth_class_id
	join susers.users USR on USR.user_id = AUT.auth_user_id
	where AUT.auth_active
	and USR.user_active
	and CLA.class_name = 'graph'
	and AUT.auth_resource is null
	and USR.user_login = p_user_login
)
select ARRAY['modifier','manager']  <@ AROG.active_roles into l_auth
from aggregated_roles_over_graphs AROG;

if l_auth is null or not l_auth then 
	raise exception 'auth failure: cannot clear graphs';
end if;

delete from sgraphs.element_trait;
delete from sgraphs.elements;
delete from sgraphs.graphs;
delete from sgraphs.traits;

end; $$;

alter procedure susers.clear_graphs owner to upa;