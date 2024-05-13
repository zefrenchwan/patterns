-- sgraphs.create_graph adds a graph. 
-- Third argument contains the id of the generated graph.
create or replace procedure sgraphs.create_graph(p_name in text, p_description in text, p_id out text)
language plpgsql as $$
declare 
	l_id text;
begin 
	select gen_random_uuid() into l_id;
	insert into sgraphs.graphs(graph_id, graph_name, graph_description) values(l_id, p_name, p_description);
	select l_id into p_id;
end; $$;

alter procedure sgraphs.create_graph owner to upa;