-- sgraphs.create_graph adds a graph. 
create or replace procedure sgraphs.create_graph(p_id in text, p_name in text, p_description in text)
language plpgsql as $$
declare 
begin 
	insert into sgraphs.graphs(graph_id, graph_name, graph_description) values(p_id, p_name, p_description);
end; $$;

alter procedure sgraphs.create_graph owner to upa;