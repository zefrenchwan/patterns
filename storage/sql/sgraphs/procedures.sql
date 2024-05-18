-- sgraphs.create_graph adds a graph. 
create or replace procedure sgraphs.create_graph(p_id in text, p_name in text, p_description in text)
language plpgsql as $$
declare 
begin 
	insert into sgraphs.graphs(graph_id, graph_name, graph_description) values(p_id, p_name, p_description);
end; $$;

alter procedure sgraphs.create_graph owner to upa;

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
	l_trait_id bigint;
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
			insert into sgraphs.traits(graph_id, trait_type, trait)
			values (p_graph_id, p_element_type, l_trait)
			returning trait_id into l_trait_id; 
		end if;
		-- then, link trait to element
		insert into sgraphs.element_trait(element_id, trait_id)
		values (p_element_id, l_trait_id);
	end loop;
end; $$;