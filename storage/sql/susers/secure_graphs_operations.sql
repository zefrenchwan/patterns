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
returns table (graph_name text, graph_description name, entry_key text, entry_values text[])
language plpgsql as $$
declare 
begin 
	call accept_any_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['manager','observer','modifier'], p_id);
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
    l_stop bool;
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

    l_stop = false;
    loop 

        with all_sources as (
            select TTG1.graph_id 
            from temp_table_graphs_imports TTG1
            where walkthrough_id = l_walkthrough_id 
            and TTG1.height <= l_height
            and (
                'modifier' = ANY(TTG1.roles) or 
                'observer' = ANY(TTG1.roles)
            )
        ), all_childs as (
            select INC.child_id as graph_id
            from sgraphs.inclusions INC 
            join all_sources ASO on ASO.graph_id = INC.source_id 
        ), all_new_childs as (
            select ACI.graph_id 
            from all_childs ACI 
            left outer join all_sources ASO on ASO.graph_id = ACI.graph_id 
            where ASO.graph_id is null 
        ), all_new_childs_roles as (
            select ANC.graph_id, array_agg(distinct ROL.role_name) as roles 
            from susers.authorizations AUT
            join all_new_childs ANC on AUT.auth_resource = ANC.graph_id
            join susers.roles ROL on ROL.role_id = AUT.auth_role_id
            join susers.classes CLA on CLA.class_id = AUT.auth_class_id
            where AUT.auth_active = true
            and CLA.class_name = 'graph'
            and AUT.auth_user_id = l_user_id
            group by ANC.graph_id
        ), all_auth_childs as (
            select ANCR.graph_id, ANCR.roles
            from all_new_childs_roles ANCR
            where  (
                'modifier' = ANY(ANCR.roles) or 
                'observer' = ANY(ANCR.roles)
            )
        )
        insert into temp_table_graphs_imports(walkthrough_id, height, graph_id, roles)
        select distinct l_walkthrough_id, l_height + 1, p_id, l_auth;

        select l_height + 1 into l_height ;

        select (count(*) > 0) into l_stop 
        from temp_table_graphs_imports
        where walkthrough_id = l_walkthrough_id
        and height = l_height; -- no + 1 because we just increased it

        exit when l_stop;
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


-- susers.transitive_load_entities_in_graph gets all entities an user may use from a graph
create or replace function susers.transitive_load_entities_in_graph(p_user_login text, p_id text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], equivalence_class text[],
    attribute_key text, attribute_values text[], attribute_periods text[]
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
        sgraphs.serialize_period(PER.period_full, PER.period_empty, PER.period_value) as activity
        from sgraphs.elements ELT  
        join all_source_graphs ASG on ASG.graph_id = ELT.graph_id
        join sgraphs.periods PER on PER.period_id = ELT.element_period
        where element_type in (1, 10)
    ), all_traits_for_elements as (
        select AEG.element_id, array_agg(TRA.trait) as traits 
        from all_elements_in_graphs AEG
        join sgraphs.element_trait ETR on AEG.element_id = ETR.element_id
        join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id 
        group by AEG.element_id
    ), all_entities as (
        select ETA.entity_id as element_id, ETA.attribute_name as attribute_key,  
        array_agg(ETA.attribute_value order by ETA.period_id) as attribute_values,
        array_agg(sgraphs.serialize_period(PER.period_full, PER.period_empty, PER.period_value) order by PER.period_id) as attribute_periods
        from sgraphs.entity_attributes ETA 
        join all_elements_in_graphs AEIG on ETA.entity_id = AEIG.element_id 
        join sgraphs.periods PER on PER.period_id = ETA.period_id 
        group by ETA.entity_id, ETA.attribute_name
    ), all_accessible_equivalences as (
        select NOD.source_element_id as element_id, array_agg(NOD.child_element_id) as equivalence_class
        from sgraphs.nodes NOD
        where exists (
            select 1 from all_elements_in_graphs AGR1
            where AGR1.element_id = NOD.source_element_id 
        ) and exists (
            select 1 from all_elements_in_graphs AGR2
            where AGR2.element_id = NOD.child_element_id
        )
        group by NOD.source_element_id
    )
    select 
    AIG.graph_id, 
    AIG.editable,
    AIG.element_id, 
    AIG.activity,
    ATE.traits,
    AAE.equivalence_class,
    ALE.attribute_key, 
    ALE.attribute_values,
    ALE.attribute_periods
    from all_elements_in_graphs AIG 
    left outer join all_traits_for_elements ATE on ATE.element_id = AIG.element_id 
    left outer join all_entities ALE on ALE.element_id = AIG.element_id 
    left outer join all_accessible_equivalences AAE on AAE.element_id = AIG.element_id;
end; $$;

alter function susers.transitive_load_entities_in_graph owner to upa;


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