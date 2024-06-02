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