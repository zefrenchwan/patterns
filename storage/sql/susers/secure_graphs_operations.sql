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

    select susers.roles_for_resource(p_user,'graph',null) into l_global_auth;
    select 'creator' = ANY(l_global_auth) into l_auth;
    if not l_auth then 
        raise exception 'auth failure: unauthorization to create graphs';
    end if;
    
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
    -- 1. test if user may create graphs. 
    -- 2. test if user has access to the imported graphs
    select susers.roles_for_resource(p_user,'graph',null) into l_global_auth;
    select 'creator' = ANY(l_global_auth) into l_auth;
    if not l_auth then 
        raise exception 'auth failure: unauthorization to create graphs';
    end if;
    -- test if graph exists and if user may see it
    foreach l_current_graph in array p_imported_graphs loop 
        if not exists (
            select 1 from sgraphs.graphs where graph_id = l_current_graph 
        ) then 
            raise exception 'graph % does not exist', l_current_graph;
        end if;

        select ('modifier' = ANY(susers.roles_for_resource(p_user,'graph', l_current_graph)) 
            or 'observer' = ANY(susers.roles_for_resource(p_user,'graph', l_current_graph))) 
        into l_auth;

        if not l_auth then 
            raise exception 'auth failure: unauthorized to use graph %', l_current_graph;
        end if;
    end loop;

    -- create graph
    call sgraphs.create_graph_from_imports(p_new_id, p_name, p_description, p_imported_graphs);

    -- grant graph access
    call susers.add_auth_for_user_on_resource(p_user, 'observer','graph', p_new_id);
    call susers.add_auth_for_user_on_resource(p_user, 'modifier','graph', p_new_id);

end; $$;

alter procedure susers.create_graph_from_imports owner to upa;