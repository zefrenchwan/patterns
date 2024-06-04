-- sgraphs.create_graph adds a graph. 
create or replace procedure sgraphs.create_graph(p_id in text, p_name in text, p_description in text)
language plpgsql as $$
declare 
	l_description text;
begin 
	if length(p_description) = 0 then 
		select null into l_description;
	else 
		select p_description into l_description;
	end if;

	insert into sgraphs.graphs(graph_id, graph_name, graph_description) values(p_id, p_name, l_description);
end; $$;

alter procedure sgraphs.create_graph owner to upa;

create or replace procedure sgraphs.create_graph_from_imports(
	p_id in text, p_name in text, p_description in text, p_sources in text[])
language plpgsql as $$
declare
	l_current_graph text;
	l_description text;
begin 
	if length(p_description) = 0 then 
		select null into l_description;
	else 
		select p_description into l_description;
	end if;

	insert into sgraphs.graphs(graph_id, graph_name, graph_description) values(p_id, p_name, l_description);

	foreach l_current_graph in array p_sources loop
		if not exists (select 1 from sgraphs.graphs where graph_id = l_current_graph) then 
			raise exception 'no graph %', l_current_graph;
		end if;

		insert into sgraphs.inclusions(source_id, child_id) values (l_current_graph, p_id);
	end loop;
end; $$;

-- sgraphs.clear_graph_metadata deletes all entries for this graph
create or replace procedure sgraphs.clear_graph_metadata(p_graph_id text)
language plpgsql as $$
declare 
begin 
	delete from sgraphs.graph_entries where graph_id = p_graph_id;
end; $$;

alter procedure sgraphs.clear_graph_metadata owner to upa;

-- sgraphs.upsert_graph_metadata_entry sets values for a given entry
create or replace procedure sgraphs.upsert_graph_metadata_entry(p_graph_id text, p_key text, p_values text[]) 
language plpgsql as $$
declare 
begin
	delete from sgraphs.graph_entries where graph_id = p_graph_id and entry_key = p_key;
	insert into sgraphs.graph_entries(graph_id, entry_key, entry_values) values (p_graph_id, p_key, p_values);
end; $$;

alter procedure sgraphs.upsert_graph_metadata_entry owner to upa;


-- sgraphs.insert_period inserts a new period and returns its new id via p_new_id
create or replace procedure sgraphs.insert_period(p_activity in text, p_new_id out bigint)
language plpgsql as $$
declare 
	-- is there a non empty activity 
	l_activity_found bool;
	-- current element in activity loop
	l_period_element text;
	-- split of period as min,max
	l_period_parts text[];
	-- left part of the activity interval
	l_period_left text;
	l_period_left_value timestamp without time zone;
	-- right part of the activity interval 
	l_period_right text; 
	l_period_right_value timestamp without time zone;
	-- if activity full 
	l_period_full bool;
	-- min of activity
	l_period_min timestamp without time zone;
	-- max of activity 
	l_period_max timestamp without time zone;
begin 
	-- find min and max of period 
	select null into l_period_min;
	select null into l_period_max;
	select false into l_activity_found;
	--
	foreach l_period_element in array string_to_array(p_activity,'U') loop 
		if l_period_element <> '];[' then 
			-- split parts to find left and right parts 
			select string_to_array(replace(replace(l_period_element,'[',''),']',''), ';') into l_period_parts;
			l_period_left = l_period_parts[1];
			l_period_right = l_period_parts[2];
			-- If first non empty, set values. Else compare.
			if found then 
				-- if current value is already null, cannot do better.
				-- Deal with min value 
				if l_period_min is not null then 
					if l_period_left = '-oo' then 
						select null into l_period_min;
					else 
						select l_period_left::timestamp without time zone into l_period_left_value;
						if l_period_left_value < l_period_min then 
							l_period_min = l_period_left_value;
						end if;
					end if;
				end if;
				-- deal with max value 
				if l_period_max is not null then 
					if l_period_right = '+oo' then 
						select null into l_period_right_value;
					else
						select l_period_right::timestamp without time zone into l_period_right_value;
						if l_period_right_value > l_period_max then 
							l_period_max = l_period_right_value;
						end if;
					end if;
				end if;
			else 
				-- first time we get a value, so force them 
				if l_period_left = '-oo' then 
					select null into l_period_min;
				else
					select l_period_left::timestamp without time zone into l_period_min;
				end if;
				if l_period_right = '+oo' then 
					select null into l_period_max;
				else
					select l_period_right::timestamp without time zone into l_period_max;
				end if;
			end if; 
			-- we found a value
			l_activity_found  = true ;	
		end if;	
	end loop;

	-- finally, insert 
	select l_period_min is null and l_period_max is null into l_period_full;

	insert into sgraphs.periods(period_empty, period_full, period_min, period_max, period_value)
	select not l_activity_found, l_period_full, l_period_min, l_period_max, p_activity
	returning period_id into p_new_id;
end; $$;

alter procedure sgraphs.insert_period owner to upa;

-- sgraphs.upsert_metadata_for_graph
create or replace procedure sgraphs.upsert_metadata_for_graph(
	p_graph_id in text, p_key in text, p_values in text[]
) language plpgsql as $$
declare 
	l_key_id bigint;
begin 
	if not exists (select 1 from sgraphs.graphs where graph_id = p_graph_id) then 
		raise exception 'no graph with provided id';
	end if;

	select entry_id into l_key_id 
	from sgraphs.graph_entries
	where graph_id = p_graph 
	and entry_key = p_key;

	if l_key_id is null then 
		insert into sgraphs.graph_entries(graph_id, entry_key, entry_values) values (p_graph_id, p_key, p_values);
	else
		update sgraphs.graph_entries set entry_values = p_values;
	end if;
end;$$;

alter procedure sgraphs.upsert_metadata_for_graph owner to upa;

-- sgraphs.upsert_element_in_graph adds an element in a graph
create or replace procedure sgraphs.upsert_element_in_graph(
	p_graph_id in text, 
	p_element_id in text, 
	p_element_type int, 
	p_activity in text,
	p_traits in text[]
) language plpgsql as $$
declare 
	-- id of previous activity if any
	l_old_period bigint;
	-- new id of current activity 
	l_new_period bigint;
	-- current trait in trait loop 
	l_trait text;
	-- current id of trait 
	l_trait_id text;
begin
	-- insert period, useful no matter what
	call sgraphs.insert_period(p_activity, l_new_period);
	
	-- cannot delete existing value due to links with attributes or roles 
	if exists (
		select 1 
		from sgraphs.elements 
		where element_id = p_element_id  
	) then 
		-- update all significant parts of the element
		-- force element in the graph 
		update sgraphs.elements 
		set graph_id = p_graph_id
		where element_id = p_element_id;
		-- clean period
		select element_period into l_old_period
		from sgraphs.elements 
		where element_id = p_element_id  
		and graph_id = p_graph_id;
		update sgraphs.elements
		set element_period = l_new_period
		where element_id = p_element_id;		
		delete from sgraphs.periods 
		where period_id = l_old_period;

		-- clean traits 
		delete from sgraphs.element_trait where element_id = p_element_id;
		-- upsert type to be sure 
		update sgraphs.elements
		set element_type = p_element_type 
		where element_id = p_element_id;
	else
		insert into sgraphs.elements(element_id, graph_id, element_type, element_period) 
		values (p_element_id, p_graph_id, p_element_type, l_new_period);
	end if;

	foreach l_trait in array p_traits loop 
		-- test if trait already exists
		select trait_id into l_trait_id 
		from sgraphs.traits 
		where graph_id = p_graph_id 
		and trait_type in (10, p_element_type) 
		and trait = l_trait;
		-- if not, insert it 
		if l_trait_id is null then 
			insert into sgraphs.traits(trait_id, graph_id, trait_type, trait)
			values (gen_random_uuid()::text, p_graph_id, p_element_type, l_trait)
			returning trait_id into l_trait_id; 
		end if;
		-- then, link trait to element
		insert into sgraphs.element_trait(element_id, trait_id)
		values (p_element_id, l_trait_id);
	end loop;
end; $$;

-- sgraphs.upsert_attributes adds one attribute and all its values (and periods)
create or replace procedure upsert_attributes(p_id text, p_name text, p_values text[], p_periods text[])
language plpgsql as $$
declare 
	l_index int;
	l_attribute_id text;
	l_value text;
	l_period_id bigint;
	l_period text;
	l_type int;
begin 
	if lenght(p_values) <> lenght(p_periods) then 
		raise exception 'different sizes for periods and values';
	end if;

	if not exists (select 1 from sgraphs.elements where element_id) then 
		raise exception 'no match for entity %', p_id;
	end if;

	delete from sgraphs.entity_attributes 
	where entity_id = p_id;

	select element_type into l_type
	from sgraphs.elements 
	where element_id = p_id;

	if l_type is not null and element_type = 2 then 
		update sgraphs.elements 
		set element_type = 10 
		where element_id = p_id;
	end if;

	l_index = 1; 
	foreach l_value in array p_values loop 
		l_period = p_periods[l_index];
		call sgraphs.insert_period(l_period, l_period_id);

		insert into sgraphs.entity_attributes(entity_id, attribute_name, attribute_value, period_id)
		select p_id, p_name, l_value, l_period_id;

		l_index = l_index + 1;
	end loop;
end; $$;

alter procedure upsert_attributes owner to upa;

-- sgraphs.upsert_links adds a role and its values to a relation
create or replace procedure sgraphs.upsert_links(p_id text, p_role text, p_values text[])
language plpgsql as $$
declare 
	l_element text;
	l_type int;
begin 

	if not exists (select 1 from sgraphs.relation_role where element_id) then 
		raise exception 'no match for relaton %', p_id;
	end if;

	foreach l_element in array p_values loop 
		if not exists (select 1 from sgraphs.elements where element_id = p_id) then 
			raise exception 'invalid argument in link: %', l_element;
		end if;
	end loop;

	select element_type into l_type
	from sgraphs.elements 
	where element_id = p_id;

	if l_type is not null and element_type = 1 then 
		update sgraphs.elements 
		set element_type = 10 
		where element_id = p_id;
	end if;

	delete from sgraphs.relation_role 
	where relation_id = p_id and role_in_relation = p_role;

	insert into sgraphs.relation_role(relation_id, role_in_relation, role_values)
	select p_id, p_role, p_values;

end; $$;

alter procedure sgraphs.upsert_links owner to upa;

-- sgraphs.load_graph_metadata returns the name, description and associated map of a graph
create or replace function sgraphs.load_graph_metadata(p_id text)
returns table (graph_name text, graph_description name, entry_key text, entry_values text[])
language plpgsql as $$
declare 
begin 
	return query 
	select 
	GRA.graph_name, GRA.graph_description, 
	GEN.entry_key, GEN.entry_values
	from sgraphs.graphs GRA
	left outer join sgraphs.graph_entries GEN on GEN.graph_id = GRA.graph_id
	where GRA.graph_id = p_id;
end; $$;

alter function sgraphs.load_graph_metadata owner to upa;


-- sgraphs.serialize_period returns the value as a text
create or replace function sgraphs.serialize_period(p_full bool, p_empty bool, p_value text) returns text 
language plpgsql as $$
declare 
begin 
	if p_full then 
		return ']-oo;+oo[';
	elsif p_empty then 
		return '];[';
	else 
		return p_value;
	end if;
end; $$;

alter function sgraphs.serialize_period owner to upa;