-- susers.create_graph_from_scratch constructs a graph
create or replace procedure susers.create_graph_from_scratch(
	p_user in text, p_new_id text, p_name in text, p_description in text
) language plpgsql as $$
declare 
begin
    if exists (
        select 1 from sgraphs.graphs where graph_id = p_new_id
    ) then 
        raise exception 'graph already exists' using errcode = '42710';
    end if;

    call susers.accept_user_access_to_resource_or_raise(p_user, 'graph', array['manager'], true, null);
    -- create graph
    call sgraphs.create_graph(p_new_id, p_name, p_description);
    -- grant graph access
    call susers.insert_new_resource(p_user, 'graph', p_new_id);
    call susers.grant_access_to_user_for_resource(p_user, 'graph', 'observer', p_new_id);
    call susers.grant_access_to_user_for_resource(p_user, 'graph', 'modifier', p_new_id);
end; $$;

alter procedure susers.create_graph_from_scratch owner to upa;

-- susers.create_graph_from_imports builds a graph as the import of others
create or replace procedure susers.create_graph_from_imports(
	p_user in text, p_new_id text, p_name in text, p_description in text, p_imported_graphs in text[]
) language plpgsql as $$
declare 
    l_refused_auths text[];
begin
    if exists (select 1 from sgraphs.graphs where graph_id = p_new_id) then 
        raise exception 'graph already exists' using errcode = '42710';
    end if;

    if array_length(p_imported_graphs, 1) = 0 then 
        raise exception 'need to import at least one graph' using errcode = '22023';
    end if;

    -- Security operations:
    call susers.accept_user_access_to_resource_or_raise(p_user, 'graph', array['manager'], true, null);
    -- test if graph exists and if user may see it
    with all_imported_graphs as (
        select unnest(p_imported_graphs) as graph_id
    ), all_nonauth_graphs as (
        select AIG.graph_id
        from all_imported_graphs AIG 
        left outer join susers.all_graphs_authorized_for_user(p_user) AGA on AGA.resource = AIG.graph_id
        where ('observer' = ANY(AGA.role_names) or 'modifier' = ANY(AGA.role_names)) 
        and AGA.resource is null 
    ) 
    select array_agg(ANG.graph_id) into l_refused_auths
    from all_nonauth_graphs ANG ;
    
    if array_length(l_refused_auths, 1) != 0 then 
        raise exception 'graphs % do not exist', l_refused_auths using errcode = '42704';
    end if;

    -- create graph
    call sgraphs.create_graph_from_imports(p_new_id, p_name, p_description, p_imported_graphs);

    -- grant graph access
    call susers.insert_new_resource(p_user, 'graph', p_new_id);
    call susers.grant_access_to_user_for_resource(p_user, 'graph', 'observer', p_new_id);
    call susers.grant_access_to_user_for_resource(p_user, 'graph', 'modifier', p_new_id);

end; $$;

alter procedure susers.create_graph_from_imports owner to upa;

create or replace procedure susers.clear_graph_metadata(p_actor text, p_graph_id text)
language plpgsql as $$
declare 
begin 
    call susers.accept_user_access_to_resource_or_raise(p_actor, 'graph', array['modifier'], true, p_graph_id);
    call sgraphs.clear_graph_metadata(p_graph_id);
end; $$;

alter procedure susers.clear_graph_metadata owner to upa;

create or replace procedure susers.upsert_graph_metadata_entry(p_actor text, p_graph_id text, p_key text, p_values text[]) 
language plpgsql as $$
declare 
begin 
    call susers.accept_user_access_to_resource_or_raise(p_actor, 'graph', array['modifier'], true, p_graph_id);
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
    return query
    select 
    GRA.graph_id, GRO.role_names as graph_roles,
    GRA.graph_name, GRA.graph_description,
    ENT.entry_key as graph_md_key, ENT.entry_values as graph_md_values
    from sgraphs.graphs GRA
    join susers.all_graphs_authorized_for_user(p_user) GRO ON GRO.resource = GRA.graph_id
    left outer join sgraphs.graph_entries ENT on ENT.graph_id = GRA.graph_id;
end; $$;

alter function susers.list_graphs_for_user owner to upa;

-- susers.load_graph_metadata returns the name, description and associated map of a graph, if user is authorized
create or replace function susers.load_graph_metadata(p_user_login text, p_id text)
returns table (graph_name text, graph_description text, entry_key text, entry_values text[])
language plpgsql as $$
declare 
begin 
	call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['manager','observer','modifier'], false, p_id);
	return query select * from sgraphs.load_graph_metadata(p_id);
end; $$;

alter function susers.load_graph_metadata owner to upa;


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
        return query select null::text, null::text[] where 1 != 1;
        return;
    end if;


    -- create data structures to deal with walkthrough
    -- id for walkthrough    
    select gen_random_uuid()::text into l_walkthrough_id;
    -- create direct neighbors table if it does not exist
    create temporary table if not exists 
    temp_table_graphs_neighbors(walkthrough_id text, graph_id text, roles text[], parent_id text);
    create temporary table if not exists 
    temp_table_graphs_imports(walkthrough_id text, height int, graph_id text, roles text[]);
    -- insert values for neighbors 
    with visible_graphs as (
        select AGA.resource as graph_id, AGA.role_names
        from susers.all_graphs_authorized_for_user(p_user_login) AGA
        where 'modifier' =ANY(AGA.role_names) or 'observer' =ANY(AGA.role_names)
    ), parents_graphs as (
        select GRA.graph_id, GRA.role_names, 
        INC.source_id as parent_id
        from visible_graphs GRA
        left outer join sgraphs.inclusions INC on INC.child_id = GRA.graph_id
        left outer join visible_graphs PAR on INC.source_id = PAR.graph_id
        where ((INC.source_id is null and PAR.graph_id is null) 
            or (INC.source_id is not null and PAR.graph_id is not null))
    )
    insert into temp_table_graphs_neighbors(walkthrough_id, graph_id, roles, parent_id)
    select l_walkthrough_id, PGR.graph_id, PGR.role_names as roles, PGR.parent_id 
    from parents_graphs PGR;

    -- graph may not exist or may be invisible
    if not exists (select 1 from temp_table_graphs_neighbors TTG where TTG.graph_id = p_id) then 
        delete from temp_table_graphs_neighbors where walkthrough_id = l_walkthrough_id;
        return query select null::text, null::text[] where 1 != 1;
        return;
    end if;

    -- height of the values 
    l_height = 1;
    -- insert first value: the current graph 
    insert into temp_table_graphs_imports(walkthrough_id, height, graph_id, roles)
    select l_walkthrough_id, l_height, TTGN.graph_id, TTGN.roles 
    from temp_table_graphs_neighbors TTGN where TTGN.graph_id = p_id;

    loop 
        with all_current_childs as (
            select TTG1.graph_id, TTG1.roles 
            from temp_table_graphs_imports TTG1
            where walkthrough_id = l_walkthrough_id 
            and TTG1.height <= l_height
            and (
                'modifier' = ANY(TTG1.roles) or 
                'observer' = ANY(TTG1.roles)
            )
        ), all_parents_of_current_childs as (
            -- get parent of the graph (seen as a child, then, and linked roles)
            select TTGN.parent_id as graph_id, TTGN.roles 
            from temp_table_graphs_neighbors TTGN 
            join all_current_childs ACC on ACC.graph_id = TTGN.graph_id
            where TTGN.parent_id is not null 
            and (
                'modifier' = ANY(TTGN.roles) or 
                'observer' = ANY(TTGN.roles)
            ) 
        ), all_new_parents_roles as (
            -- get all parents but exclude the ones we have already added, 
            -- that is already in all_current_childs
            select distinct APA.graph_id, APA.roles
            from all_parents_of_current_childs APA 
            left outer join all_current_childs ACC on ACC.graph_id = APA.graph_id 
            where ACC.graph_id is null 
        )
        insert into temp_table_graphs_imports(walkthrough_id, height, graph_id, roles)
        select l_walkthrough_id, l_height + 1, ANP.graph_id, ANP.roles
        from all_new_parents_roles ANP;

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
    delete from temp_table_graphs_neighbors
    where walkthrough_id = l_walkthrough_id;
    return;
end; $$;


-- susers.transitive_load_base_elements_in_graph loads all visible elements from a graph to all its dependencies
create or replace function susers.transitive_load_base_elements_in_graph(p_user_login text, p_id text, p_period text)
returns table (
    graph_id text, editable bool, 
    element_id text, element_type int, activity text, traits text[], 
    equivalence_class text[], equivalence_class_graph text[]
) language plpgsql as $$
declare
begin 
    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['observer','modifier'], false, p_id);
    
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
        PER.period_value as activity
        from sgraphs.elements ELT  
        join all_source_graphs ASG on ASG.graph_id = ELT.graph_id
        join sgraphs.periods PER on PER.period_id = ELT.element_period
        where not sgraphs.are_periods_disjoin(p_period, PER.period_value)
    ), all_traits_for_elements as (
        select AEG.element_id, array_agg(TRA.trait) as traits 
        from all_elements_in_graphs AEG
        join sgraphs.element_trait ETR on AEG.element_id = ETR.element_id
        join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id 
        group by AEG.element_id
    ), all_accessible_equivalences as (
        select NOD.source_element_id as element_id, 
        array_agg(NOD.child_element_id order by AGR1.graph_id) as equivalence_class,
        array_agg(AGR1.graph_id order by AGR1.graph_id) as equivalence_class_graph
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
create or replace function susers.transitive_load_entities_in_graph(p_user_login text, p_id text, p_period text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], 
    equivalence_class text[], equivalence_class_graph text[],
    attribute_key text, attribute_values text[], attribute_periods text[]
) language plpgsql as $$
declare
begin 
    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['observer','modifier'], false, p_id);
    return query
    with all_source_entities as (
        select TLB.graph_id, TLB.editable, 
        TLB.element_id, TLB.activity, TLB.traits, 
        TLB.equivalence_class, TLB.equivalence_class_graph
        from susers.transitive_load_base_elements_in_graph(p_user_login, p_id, p_period) TLB
        where TLB.element_type in (1,10)
    ), all_entities as (
        select ETA.entity_id as element_id, ETA.attribute_name as attribute_key,  
        array_agg(ETA.attribute_value order by ETA.period_id) as attribute_values,
        array_agg(PER.period_value order by PER.period_id) as attribute_periods
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

-- susers.transitive_load_relations_in_graph loads all relations in dependent graphs starting at p_id a given graph
create or replace function susers.transitive_load_relations_in_graph(p_user_login text, p_id text, p_period text)
returns table (
    graph_id text, editable bool, 
    element_id text, activity text, traits text[], 
    equivalence_class text[], equivalence_class_graph text[],
    role_in_relation text, role_values text[], role_periods text[]
) language plpgsql as $$
declare
begin 
    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph',ARRAY['observer','modifier'], false, p_id);
    return query
    with all_visible_graphs as (
        select TGS.graph_id
        from susers.transitive_visible_graphs_since(p_user_login, p_id) TGS
    ), all_source_elements as (
        select TLB.graph_id, TLB.editable, 
        TLB.element_id, TLB.activity, TLB.traits, 
        TLB.equivalence_class, TLB.equivalence_class_graph, 
        TLB.element_type
        from susers.transitive_load_base_elements_in_graph(p_user_login, p_id, p_period) TLB
        where TLB.activity <> '];['
    ), all_visible_relations as (
        select 
        ALLSE.element_id
        from all_source_elements ALLSE
        where ALLSE.element_type in (2,10)
    ), all_relation_values as (
        select RRO.relation_id, 
        RRO.role_in_relation,
        RRV.relation_value as role_value,
        PER.period_value as role_period 
        from sgraphs.relation_role RRO
        join all_visible_relations AVR on AVR.element_id = RRO.relation_id
        join sgraphs.relation_role_values RRV on RRV.relation_role_id = RRO.relation_role_id
        join all_source_elements ELT on ELT.element_id = RRV.relation_value
        join sgraphs.periods PER on PER.period_id = RRV.relation_period_id
        where PER.period_value <> '];['
        and not sgraphs.are_periods_disjoin(p_period, PER.period_value)
    ), visible_relation_values as (
        select 
        ARV.relation_id, 
        ARV.role_in_relation,
        array_agg(ARV.role_value order by ARV.role_value) as role_values, 
        array_agg(ARV.role_period order by ARV.role_value) as role_periods
        from all_relation_values ARV 
        group by ARV.relation_id, ARV.role_in_relation 
    )
    select distinct
    ASE.graph_id, 
    ASE.editable,
    ASE.element_id, 
    ASE.activity,
    ASE.traits,
    ASE.equivalence_class,
    ASE.equivalence_class_graph,
    VRV.role_in_relation, 
    VRV.role_values,
    VRV.role_periods
    from all_source_elements ASE 
    join visible_relation_values VRV on ASE.element_id = VRV.relation_id;
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
    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], true, p_graph_id);
    
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

    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], true, l_graph_id); 
    call sgraphs.upsert_attributes(p_id, p_name, p_values, p_periods);

end; $$;

alter procedure susers.upsert_attributes owner to upa;


-- susers.upsert_links upserts links if user has access to all underlying graphs
create or replace procedure susers.upsert_links(
    p_user_login text, p_role_id text, p_role_name text, 
    p_operands text[], p_periods text[])
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

    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], true, l_graph_id); 


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
        call sgraphs.upsert_links(p_role_id, p_role_name, p_operands, p_periods);
    end if;
end;$$;

alter procedure susers.upsert_links owner to upa;

create or replace function susers.load_element_by_id(p_user_login text, p_element_id text)
returns table (
	element_id text,
	traits text[], activity text,
	role_name text, role_values text[], role_periods text[],
	attribute_name text, attribute_values text[], attribute_periods text[]) 
language plpgsql as $$
declare 
	l_element_type int;
	l_graph_id text;
begin
	
    select ELT.graph_id into l_graph_id 
    from sgraphs.elements ELT 
    where ELT.element_id = p_element_id;

    if l_graph_id is null then 
        -- just returns empty
        return query select null, null, null, null, null, null, null, null, null where 1 <> 1;
        return;
    end if;

    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier','observer'], false, l_graph_id);

    return query
    with element_data as (
        select ELT.element_id,
        array_agg(TRA.trait) as traits,
        max(PER.period_value) as activity  
        from sgraphs.elements ELT 
        join sgraphs.periods PER on PER.period_id = ELT.element_period
        join sgraphs.element_trait ETR on ETR.element_id = ELT.element_id
        join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id
        where ELT.element_id = p_element_id
        group by ELT.element_id
    ), element_roles as (
        select RRO.relation_id as element_id, 
        RRO.role_in_relation as role_name , 
        array_agg(RRV.relation_value order by RRV.relation_value) as role_values,
        array_agg(PER.period_value order by RRV.relation_value) as role_periods  
        from sgraphs.relation_role RRO 
        join sgraphs.relation_role_values RRV on RRO.relation_role_id = RRV.relation_role_id
        join sgraphs.periods PER on PER.period_id = RRV.relation_period_id
        where RRO.relation_id = p_element_id
        group by RRO.relation_id, RRO.role_in_relation
    ), element_entity as (
        select ENA.entity_id as element_id,
        ENA.attribute_name, 
        array_agg(ENA.attribute_value order by ENA.attribute_id) as attribute_values,  
        array_agg(PER.period_value order by ENA.attribute_id) as attribute_periods
        from sgraphs.entity_attributes ENA 
        join sgraphs.periods PER on PER.period_id = ENA.period_id
        where PER.period_value <> '];['
        and ENA.entity_id = p_element_id
        group by ENA.entity_id, ENA.attribute_name
    )
    select 
    EDA.element_id,
    EDA.traits,	
    EDA.activity,
    ERO.role_name,
    ERO.role_values,
    ERO.role_periods,
    ELE.attribute_name, 
    ELE.attribute_values, 
    ELE.attribute_periods
    from element_data EDA 
    left outer join element_roles ERO on ERO.element_id = EDA.element_id 
    left outer join element_entity ELE on ELE.element_id = EDA.element_id;
end;$$;

alter function susers.load_element_by_id owner to upa;


-- susers.create_equivalent_element_into_graph creates an equivalent node in a graph
create or replace procedure susers.create_equivalent_element_into_graph(
    p_user_login text, p_source_id text, p_destination_graph_id text, p_new_element_id text
) language plpgsql as $$
declare 
    l_graph_id text;
begin 
    select graph_id into l_graph_id 
    from sgraphs.elements 
    where element_id = p_source_id;

    if l_graph_id is null then 
        raise exception 'no graph' using errcode = '42704';
    end if;

    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier','observer'], false, l_graph_id);
    call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], false, p_destination_graph_id);
    call sgraphs.create_copy_node(p_source_id, p_destination_graph_id, p_new_element_id); 
end; $$;

alter procedure susers.create_equivalent_element_into_graph owner to upa;

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
	call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['modifier'], true, l_graph_id);

	if exists (
		select 1
		from sgraphs.relation_role_values RRV
		where RRV.relation_value = p_element_id
	) then 
		raise exception 'a relation depends on current element to delete' using errcode = '23503';
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
		raise exception 'resource not found: %', p_graph_id;
	end if;

	-- user may not modify graph 
	call susers.accept_user_access_to_resource_or_raise(p_user_login, 'graph', ARRAY['manager'], true, l_graph_id);

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
		raise exception 'forbidden: a relation outside the graph depends on an element in the graph' using errcode = '23503';
	end if;

	-- ok to delete 
	delete from sgraphs.elements where graph_id = p_graph_id;
    delete from sgraphs.graphs where graph_id = p_graph_id;
    call susers.delete_resource('graph', p_graph_id);
end; $$;

alter procedure susers.delete_graph owner to upa;

-- susers.clear_graphs clear all data in graphs schema.
create or replace procedure susers.clear_graphs(p_user_login text)
language plpgsql as $$
declare
	l_auth bool;
    l_class_id int;
begin

    with all_graphs as (
        select count(distinct GRA.graph_id) as all_counter 
        from sgraphs.graphs GRA
    ), all_auth_graphs as (
        select count(distinct AGA.resource) as auth_counter
        from susers.all_graphs_authorized_for_user(p_user_login) AGA
        where 'manager' = ANY(AGA.role_names) 
    ) select (auth_counter = all_counter) into l_auth
    from all_graphs ALLG 
    cross join all_auth_graphs AUTG;

    if l_auth is null or not l_auth then 
        raise exception 'some unauthorized graphs' using errcode = '42501';
        return;
    end if;

    select class_id into l_class_id from susers.classes where class_name = 'graph';

    delete from sgraphs.element_trait;
    delete from sgraphs.elements;
    delete from sgraphs.graphs;
    delete from sgraphs.traits;
    delete from sgraphs.periods;
    delete from susers.authorizations where not auth_all_resources and auth_class_id = l_class_id;
    delete from susers.resources where resource_type = l_class_id;

end; $$;

alter procedure susers.clear_graphs owner to upa;
