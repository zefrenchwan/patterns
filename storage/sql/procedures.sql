-- spat.UpsertPattern:
-- adds a pattern named p_pattern and links it to its parents if it did not exist
-- sets parents no matter previous values if already inserted
-- Calling parents with array[]::text[] will just add this pattern, no parent. 
create or replace procedure spat.UpsertPattern(p_pattern text, p_parents text[]) 
language plpgsql
as $$
declare
	v_pattern text ;
	s_index bigint := -1 ; 
	v_index bigint := -1 ;
begin

select pattern_id into s_index
from spat.patterns
where pattern_name = p_pattern;

if s_index is null then 
	insert into spat.patterns(pattern_name) values (p_pattern) returning pattern_id into s_index;
end if;

delete from spat.pattern_links where subpattern_id = s_index;

foreach v_pattern slice 0 in array p_parents loop 	
	select pattern_id into v_index
	from spat.patterns
	where pattern_name = v_pattern;

	if v_index is null then 
		insert into spat.patterns(pattern_name) values (v_pattern) returning pattern_id into s_index;
	end if;
	
	if not exists (select 1 from spat.pattern_links where subpattern_id = s_index and superpattern_id = v_index) then 
		insert into spat.pattern_links(subpattern_id, superpattern_id) values (s_index, v_index);
	end if ; 
	
end loop; 

end; $$
;


-- spat.CreateTrait creates a trait if it does not exist. 
create or replace procedure spat.CreateTrait(p_trait text) language plpgsql as $$
declare
	l_traitid bigint;
begin

	select trait_id, trait_type, trait into l_traitid 
	from spat.traits 
	where trait = p_trait and trait_type in (1, 10);

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_trait);
	end if;
end; $$
;

-- spat.CreateRelationTrait creates a relation trait if it does not exist
create or replace procedure spat.CreateRelationTrait(p_trait text) language plpgsql as $$
declare
	l_traitid bigint;
begin

	select trait_id, trait_type, trait into l_traitid 
	from spat.traits 
	where trait = p_trait and trait_type in (2,10);

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_trait);
	end if;
end; $$
;

-- spat.LinkTraitsInPattern links a subtrait to a trait in a pattern. 
create or replace procedure spat.LinkTraitsInPattern(p_patternname text, p_subtrait text, p_trait text, p_reftype int) language plpgsql as $$
declare
	l_patternid bigint;
	l_traitid bigint;
	l_subtraitid bigint;
begin

	select trait_id  into l_traitid 
	from spat.traits 
	where trait = p_trait and trait_type in (p_reftype, 10) ;

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_trait)
		returning trait_id into l_traitid;
	end if;
	
	select trait_id  into l_subtraitid 
	from spat.traits 
	where trait = p_subtrait and trait_type in (p_reftype, 10);

	if l_subtraitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_subtrait)
		returning trait_id into l_subtraitid;
	end if;
	
	select pattern_id into l_patternid 
	from spat.patterns 
	where pattern_name = p_patternname;
	
	if l_patternid is null then 
		insert into spat.patterns(pattern_name) values (p_patternname)
		returning pattern_id into l_patternid;
	end if;
	
	if not exists (
		select 1 from spat.mdlinks
		where pattern_id = l_patternid and subtrait_id = l_subtraitid
		and supertrait_id = l_traitid and reftype in (10, p_reftype)
	) then 
		insert into spat.mdlinks values (l_patternid, p_reftype, l_subtraitid, l_traitid);
	end if ;
	
end; $$
;


-- spat.AddRoleToRelationInPattern sets a role with given EXISTING traits. 
-- pattern and relation may be created on the fly if necessary. 
create or replace procedure spat.AddRoleToRelationInPattern(p_patternname text, 
	p_relationtrait text, p_role text, p_roleTraits text[]) language plpgsql as $$
declare
	l_patternid bigint;
	l_traitid bigint;
	l_roletrait text;
	l_desttrait bigint;
begin
	select pattern_id into l_patternid
	from spat.patterns 
	where pattern_name = p_patternname;
	
	if l_patternid is null then 
		insert into spat.patterns(pattern_name) values (p_patternname);
	end if; 
	
	select trait_id into l_traitid 
	from spat.traits 
	where trait = p_relationtrait and trait_type in (2, 10);

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (2, p_relationtrait)
		returning trait_id into l_traitid;
	end if;
	
	delete from spat.mdroles where pattern_id = l_patternid 
	and relation_trait_id = l_traitid and role_in_relation = p_role;
	
	foreach l_roletrait slice 0 in array p_roleTraits loop 
		select trait_id into l_desttrait
		from spat.traits 
		where trait = l_roletrait;
		
		if l_desttrait is null then 
			raise exception 'non existing destination trait %', l_roletrait;
		end if ;
	
		-- insert for sure due to deletion
		insert into spat.mdroles values (l_patternid, l_traitid, p_role, l_desttrait);
		
	end loop;
	
end; $$
;

--------------------------------------
-- STARTING ENTITIES RELATIONS CODE --
--------------------------------------

-- spat.UpsertEntity upserts an entity: activity and traits)
-- If traits do not exist, they are inserted as entity traits.  
create or replace procedure spat.UpsertEntity(p_id text, p_activity text, p_traits text[]) language plpgsql as $$
declare 
	l_trait text;
	l_traitid bigint;
	l_period bigint;
	l_previousperiod bigint;
begin
	if not exists (select 1 from spat.entities where entity_id = p_id) then 
		call spat.setperiod(p_activity, l_period);
		insert into spat.entities values (p_id, l_period);
	else		
		select entity_period into l_previousperiod
		from spat.entities where entity_id = p_id;

		call spat.setperiod(p_activity, l_period);
		update spat.entities set entity_period = l_period;
		delete from spat.periods where period_id = l_previousperiod;
	end if;
	
	delete from spat.entity_trait where entity_id = p_id;

	if p_traits is null or array_length(p_traits, 1) = 0 then 
		return;
	end if;
	
	foreach l_trait slice 0 in array p_traits loop 
		select trait_id into l_traitid
		from spat.traits 
		where trait = l_trait and trait_type in (1,10);
	
		if l_traitid is null then 
			insert into spat.traits values (1, l_trait);
		end if;
		
		insert into spat.entity_trait values (p_id, l_traitid);
	end loop;
	
end $$;

-- spat.UpsertRelation upsers a relation with its activity and traits. 
-- Relational traits not already stored are inserted as relational traits (not mixed)
create or replace procedure spat.UpsertRelation(p_id text, p_activity text, p_traits text[]) language plpgsql as $$
declare 
	l_trait text;
	l_traitid bigint;
	l_period bigint;
	l_previousperiod bigint;
begin
	if not exists (select 1 from spat.relations where relation_id = p_id) then 
		call spat.setperiod(p_activity, l_period);
		insert into spat.relations values (p_id, l_period);
	else		
		select relation_activity into l_previousperiod
		from spat.relations where relation_id = p_id;

		call spat.setperiod(p_activity, l_period);
		update spat.relations set relation_activity = l_period;
		delete from spat.periods where period_id = l_previousperiod;
	end if;
	
	delete from spat.relation_trait where relation_id = p_id;

	if p_traits is null or array_length(p_traits, 1) = 0 then 
		return;
	end if;
	
	foreach l_trait slice 0 in array p_traits loop 
		select trait_id into l_traitid
		from spat.traits 
		where trait = l_trait and trait_type in (2,10);
	
		if l_traitid is null then 
			insert into spat.traits values (2, l_trait);
		end if;
		
		insert into spat.relation_trait values (p_id, l_traitid);
	end loop;
	
end $$;

-- spat.UpsertRoleInRelation upserts role for an existing relation
create or replace procedure spat.UpsertRoleInRelation(
	p_id text, p_role text, p_values text[]
) language plpgsql as $$
declare 
begin 

	if not exists (select 1 from spat.relations where relation_id = p_id) then 
		raise exception 'relation does not exist (cannot create due to missing period)';
	end if;
	
	delete from spat.relation_role where relation_id = p_id and role_in_relation = p_role;
	insert into spat.relation_role values (p_id, p_role, p_values);
end; $$;

-- spat.ArePeriodsDisjoin returns true if periods are disjoin. 
-- Each period is represented as intervals separated by U
-- Each interval is represented as:
-- * border is either [ or ]
-- * value is either a YYYY-MM-DD HH24:MI:ss or -oo or +oo
create or replace function spat.ArePeriodsDisjoin(p_a text, p_b text) returns bool 
language plpgsql as $$
declare 
	l_periods_a text[] := string_to_array(p_a,'U');
	l_periods_b text[] := string_to_array(p_b,'U');
	l_period_a text; -- each period a loop
	l_period_b text; -- each period b loop
	l_min_a timestamp without time zone; -- null means -oo
	l_min_b timestamp without time zone; -- null means -oo
	l_max_a timestamp without time zone; -- null means +oo
	l_max_b timestamp without time zone; -- null means +oo
	l_min_in_a bool; -- min included for period a 
	l_min_in_b bool; -- min included for period b
	l_max_in_a bool; -- max included for period a
	l_max_in_b bool; -- max included for period b
	l_value text; -- temp value
begin
	foreach l_period_a slice 0 in array l_periods_a loop
		-- Parse interval to fill inner values for each interval in a.
		-- To do so, basic string parsing using plpgsql functions. 
		-- parse left part of the interval
		if left(l_period_a,4) = ']-oo' then
			l_min_a = null;
			l_min_in_a = false;
		else 
			select substr(split_part(l_period_a,';',1),2) into l_value;
			select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_min_a;
			if left(l_period_a,1) = '[' then 
				l_min_in_a = true;
			else
				l_min_in_a = false;
			end if;
		end if;
		-- parse right part of the interval
		if right(l_period_a, 4) = '+oo[' then 
			l_max_a = null;
			l_max_in_a = false;
		else
			select split_part(l_period_a,';',2) into l_value;
			select left(l_value, length(l_value)-1) into l_value;
			select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_max_a;
			if right(l_period_a,1) = '[' then 
				l_max_in_a = false;
			else
				l_max_in_a = true;
			end if;
		end if;		
		-- p_a is parsed, then move to p_b. 
		-- We parse it again, indeed
		foreach l_period_b slice 0 in array l_periods_b loop
			if left(l_period_b,4) = ']-oo' then
				l_min_b = null;
				l_min_in_b = false;
			else 
				select substr(split_part(l_period_b,';',1),2) into l_value;
				select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_min_b;
				if left(l_period_b,1) = '[' then 
					l_min_in_b = true;
				else
					l_min_in_b = false;
				end if;
			end if;
			if right(l_period_b, 4) = '+oo[' then 
				l_max_b = null;
				l_max_in_b = false;
			else
				select split_part(l_period_b,';',2) into l_value;
				select left(l_value, length(l_value)-1) into l_value;
				select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_max_b;
				if right(l_period_b,1) = '[' then 
					l_max_in_b = false;
				else
					l_max_in_b = true;
				end if;
			end if;		

			-- Splitting is done, we may compare current interval in a and current interval in b
			if l_min_a is null and l_min_b is null then 
				-- common -oo
				return false;
			elsif l_max_a is null and l_max_b is null then
				-- common +oo
				return false;
			elsif l_min_a is null and l_min_b < l_max_a then
				-- common point before end of a = ]-oo, end)
				return false;
			elsif l_min_a is null and l_min_b = l_max_a and l_min_in_b and l_max_in_a then
				-- common border point l_max_a = l_min_b with both values included 
				return false;
			elsif l_min_b is null and l_min_a < l_max_b then
				-- same reason, dual interval
				return false;
			elsif l_min_b is null and l_min_a = l_max_b and l_min_in_a and l_max_in_b then 
				-- same reason, dual interval
				return false;
			elsif l_max_a is null and l_max_b > l_min_a then
				-- max(b) is more than min(a) and a is (min(a), +oo[
				return false;
			elsif l_max_a is null and l_max_b = l_min_a and l_min_in_a and l_max_in_b then 
				-- a is [min(a),+oo[ and b is ..., min(a)]
				return false;
			elsif l_max_b is null and l_max_a > l_min_b then
				return false;
			elsif l_max_b is null and l_max_a = l_min_b and l_min_in_b and l_max_in_a then 
				return false;	
			end if;	
			
			-- both are finite
			if l_min_a > l_max_b then 
				continue;
			elsif l_min_b > l_max_a then 
				continue;
			elsif l_min_a = l_max_b and l_max_in_b and l_min_in_a then 
				return false;
			elsif l_min_a = l_max_b then 
				continue;
			elsif l_min_b = l_max_a and l_max_in_a and l_min_in_b then
				return false;
			elsif l_min_b = l_max_a then
				continue;
			else
				return false;
			end if;
		end loop;		
	end loop;

	return true;
end $$;

-- spat.SetPeriod sets a period and returns its id via out variable
create or replace procedure spat.SetPeriod(p_period text, p_newid out bigint)
language plpgsql as $$
declare 
	l_periods text[] := string_to_array(p_period,'U');
	l_period text; -- each period loop
	l_min timestamp without time zone; -- null means -oo
	l_max timestamp without time zone; -- null means +oo
	l_min_in bool; -- min included for period  
	l_max_in bool; -- max included for period 
	l_value text; -- temp value
	l_resid bigint; -- to store p_newid value
	l_globalmin timestamp without time zone;
	l_globalmax timestamp without time zone;
	l_counter int;
begin
	if p_period = '];[' or upper(p_period) = 'EMPTY' then 
		insert into spat.periods(period_empty, period_full, period_min, period_max, period_value) 
		values(true, false, null, null,null)
		returning period_id into l_resid;
		p_newid = l_resid;
	elsif p_period = ']-oo;+oo[' then 
		insert into spat.periods(period_empty, period_full, period_min, period_max, period_value) 
		values(false, true, null, null,null)
		returning period_id into l_resid;
		p_newid = l_resid;
	else 
		-- parse values to find min and max timestamps
		l_counter = 0;
		foreach l_period slice 0 in array l_periods loop
			-- parse left part of the interval
			if left(l_period,4) = ']-oo' then
				l_min = null;
			else 
				select substr(split_part(l_period,';',1),2) into l_value;
				select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_min;
			end if;
			-- parse right part of the interval
			if right(l_period, 4) = '+oo[' then 
				l_max = null;
			else
				select split_part(l_period,';',2) into l_value;
				select left(l_value, length(l_value)-1) into l_value;
				select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_max;
			end if;
			
			-- then, set global min, global max
			if l_counter = 0 then 
				l_globalmin = l_min;
				l_globalmax = l_max;
			else
				if l_min is null then 
					l_globalmin = l_min;
				elsif l_globalmin > l_min then 
					l_globalmin = l_min;
				end if;
				
				if l_max is null then
					l_globalmax = l_max;			
				elsif l_globalmax < l_max then
					l_globalmax = l_max;
				end if;
			end if;

			l_counter = l_counter + 1;
		end loop;
		-- insert period 
		insert into spat.periods(period_empty, period_full, period_min, period_max, period_value) 
		values(false, false, l_globalmin, l_globalmax, p_period)
		returning period_id into l_resid;
		p_newid = l_resid;
	end if;
end $$;

-- spat.AddAttributeValuesInEntity adds attribute values to an exising entity
create or replace procedure spat.AddAttributeValuesInEntity(
	p_id text, p_attribute text, p_values text[], p_periods text[]
) language plpgsql as $$
declare 
	l_attributeid bigint;
	l_value text;
	l_period text;
	l_periodid bigint;
begin 
	if not exists (select 1 from spat.entities where entity_id = p_id) then 
		raise exception 'entity with id % does not exist', p_id;
	end if;
	
	if array_length(p_values,1) != array_length(p_periods,1) then 
		raise exception 'size of parameters do not match';
	end if; 

	delete from spat.periods where period_id in (
		select period_id
		from spat.entity_attributes
		where entity_id = p_id and attribute_name = p_attribute
	);
	
	--delete from spat.entity_attributes
	--where entity_id = p_id and attribute_name = p_attribute; 
	
	for i in 1 .. array_length(p_values,1) loop 
		l_value = p_values[i];
		l_period = p_periods[i];
		
		call spat.SetPeriod(l_period, l_periodid);
		insert into spat.entity_attributes(entity_id,attribute_name, attribute_value,period_id)
		values(p_id, p_attribute, l_value, l_periodid);	
	end loop;
end $$;