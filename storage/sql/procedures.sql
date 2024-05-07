--------------------------------------
-- STARTING ENTITIES RELATIONS CODE --
--------------------------------------

-- sgraphs.UpsertElement upserts an element (relation and entity): id, activity and traits.
-- If traits do not exist, they are inserted as entity traits.  
create or replace procedure sgraphs.UpsertElement(p_id text, p_type int, p_activity text, p_traits text[]) language plpgsql as $$
declare 
	l_trait text;
	l_traitid bigint;
	l_period bigint;
	l_previousperiod bigint;
begin
	if not exists (select 1 from sgraphs.reftypes where reftype_id = p_type) then 
		raise exception 'invalid p_type';
	end if;

	if not exists (select 1 from sgraphs.elements where element_id = p_id) then 
		call sgraphs.setperiod(p_activity, l_period);
		insert into sgraphs.elements(element_id, element_type, element_period) 
		values (p_id, p_type, l_period);
	else		
		select element_period into l_previousperiod
		from sgraphs.elements where element_id = p_id;

		call sgraphs.setperiod(p_activity, l_period);
		update sgraphs.elements set element_period = l_period;
		delete from sgraphs.periods where period_id = l_previousperiod;
	end if;
	
	delete from sgraphs.element_trait where element_id = p_id;

	if p_traits is null or array_length(p_traits, 1) = 0 then 
		return;
	end if;
	
	foreach l_trait slice 0 in array p_traits loop 
		select trait_id into l_traitid
		from sgraphs.traits 
		where trait = l_trait and trait_type in (p_type,10);
	
		if l_traitid is null then 
			insert into sgraphs.traits(trait_type, trait) values (1, l_trait) returning trait_id into l_traitid;
		end if;
		
		insert into sgraphs.element_trait(element_id, trait_id) values (p_id, l_traitid);
	end loop;
end $$;

alter procedure sgraphs.UpsertElement owner to upa;

create or replace procedure sgraphs.UpsertEntity(p_id text, p_activity text, p_traits text[]) language plpgsql as $$
declare 
begin 
	call sgraphs.UpsertElement(p_id, 1, p_activity, p_traits);
end; $$;

alter procedure sgraphs.UpsertEntity owner to upa;

create or replace procedure sgraphs.UpsertRelation(p_id text, p_activity text, p_traits text[]) language plpgsql as $$
declare 
begin 
	call sgraphs.UpsertElement(p_id, 2, p_activity, p_traits);
end; $$;

alter procedure sgraphs.UpsertRelation owner to upa;

-- sgraphs.ClearRolesForRelation clears roles for relation
create or replace procedure sgraphs.ClearRolesForRelation(p_id text) language plpgsql as $$
declare 
begin
	delete from sgraphs.relation_role where relation_id = p_id;
end $$;

alter procedure sgraphs.ClearRolesForRelation owner to upa;

-- sgraphs.UpsertRoleInRelation upserts role for an existing relation
create or replace procedure sgraphs.UpsertRoleInRelation(
	p_id text, p_role text, p_values text[]
) language plpgsql as $$
declare 
begin 

	if not exists (select 1 from sgraphs.elements where element_id = p_id) then 
		raise exception 'relation does not exist (cannot create due to missing period)';
	end if;
	
	delete from sgraphs.relation_role where relation_id = p_id and role_in_relation = p_role;
	insert into sgraphs.relation_role(relation_id, role_in_relation, role_values) values (p_id, p_role, p_values);
end; $$;

alter procedure sgraphs.UpsertRoleInRelation owner to upa;

-- sgraphs.LoadElement returns data to build an element per id
create or replace function sgraphs.LoadElement(p_id text) 
returns table(
	-- element part 
	element_id text, element_type int, element_traits text[], 
	period_full bool, period_value text, 
	-- role and values (null for entity)
	role_in_relation text, role_values text[],
	-- entity part (null for relation)
	attribute_name text, attribute_value text, 
	attribute_period_full bool, attribute_period_value text
) language plpgsql as $$
declare
	
begin
	return query 
	with 
	element_data as (
		select ELT.element_id, ELT.element_type, 
		PER.period_full, PER.period_value
		from sgraphs.elements ELT
		join sgraphs.periods PER on PER.period_id = ELT.element_period
		where ELT.element_id = p_id
	),
	element_traits as (
		select ELT.element_id, array_agg(TRA.trait) as traits
		from sgraphs.elements ELT
		join sgraphs.element_trait ETR on ETR.element_id = ELT.element_id
		join sgraphs.traits TRA on TRA.trait_id = ETR.trait_id
		where ELT.element_id = p_id
		group by ELT.element_id
	),
	entity_data as (
		select ETA.entity_id,
		ETA.attribute_name, ETA.attribute_value,
		PER.period_full as attribute_period_full, 
		PER.period_value as attribute_period_value
		from sgraphs.entity_attributes ETA 
		join sgraphs.periods PER on ETA.period_id = PER.period_id
		where ETA.entity_id = p_id
	),
	role_data as (
		select RRO.relation_id,
		RRO.role_in_relation, RRO.role_values
		from sgraphs.relation_role RRO 
		where RRO.relation_id = p_id 
	)
	select 
	ELD.element_id, ELD.element_type, 
	ETA.traits as element_traits, 
	ELD.period_full, ELD.period_value, 
	RDA.role_in_relation, RDA.role_values,
	EDA.attribute_name, EDA.attribute_value, 
	EDA.attribute_period_full, EDA.attribute_period_value
	from element_data ELD
	left outer join element_traits ETA on ETA.element_id = ELD.element_id
	left outer join entity_data EDA on EDA.entity_id = ELD.element_id
	left outer join role_data RDA on RDA.relation_id = ELD.element_id;
end; $$;

alter function sgraphs.LoadElement owner to upa;

-- sgraphs.ArePeriodsDisjoin returns true if periods are disjoin. 
-- Each period is represented as intervals separated by U
-- Each interval is represented as:
-- * border is either [ or ]
-- * value is either a YYYY-MM-DD HH24:MI:ss or -oo or +oo
create or replace function sgraphs.ArePeriodsDisjoin(p_a text, p_b text) returns bool 
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

alter function sgraphs.ArePeriodsDisjoin owner to upa;

-- sgraphs.SetPeriod sets a period and returns its id via out variable
create or replace procedure sgraphs.SetPeriod(p_period text, p_newid out bigint)
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
		insert into sgraphs.periods(period_empty, period_full, period_min, period_max, period_value) 
		values(true, false, null, null,null)
		returning period_id into l_resid;
		p_newid = l_resid;
	elsif p_period = ']-oo;+oo[' then 
		insert into sgraphs.periods(period_empty, period_full, period_min, period_max, period_value) 
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
		insert into sgraphs.periods(period_empty, period_full, period_min, period_max, period_value) 
		values(false, false, l_globalmin, l_globalmax, p_period)
		returning period_id into l_resid;
		p_newid = l_resid;
	end if;
end $$;

alter procedure sgraphs.SetPeriod owner to upa;

-- sgraphs.AddAttributeValuesInEntity adds attribute values to an exising entity
create or replace procedure sgraphs.AddAttributeValuesInEntity(
	p_id text, p_attribute text, p_values text[], p_periods text[]
) language plpgsql as $$
declare 
	l_attributeid bigint;
	l_value text;
	l_period text;
	l_periodid bigint;
begin 
	if not exists (
		select 1 from sgraphs.elements 
		where element_id = p_id and element_type in (1,10)
	) then 
		raise exception 'entity with id % does not exist', p_id;
	end if;
	
	if array_length(p_values,1) != array_length(p_periods,1) then 
		raise exception 'size of parameters do not match';
	end if; 

	delete from sgraphs.periods where period_id in (
		select period_id
		from sgraphs.entity_attributes
		where entity_id = p_id and attribute_name = p_attribute
	);
	
	delete from sgraphs.entity_attributes
	where entity_id = p_id and attribute_name = p_attribute; 
	
	for i in 1 .. array_length(p_values,1) loop 
		l_value = p_values[i];
		l_period = p_periods[i];
		
		call sgraphs.SetPeriod(l_period, l_periodid);
		insert into sgraphs.entity_attributes(entity_id,attribute_name, attribute_value,period_id)
		values(p_id, p_attribute, l_value, l_periodid);	
	end loop;
end $$;

alter procedure sgraphs.AddAttributeValuesInEntity owner to upa;

-- sgraphs.IsPeriodActiveAtTimestamp returns true if period contains the moment
create or replace function sgraphs.IsPeriodActiveAtTimestamp(
	p_period_empty bool, 
	p_period_full bool,
	p_period text, 
	p_moment timestamp without time zone
) returns bool language plpgsql as $$
declare 
	l_periods text[];
	l_period text; -- each period loop
	l_min timestamp without time zone; -- null means -oo
	l_max timestamp without time zone; -- null means +oo
	l_min_in bool; -- min included for period  
	l_max_in bool; -- max included for period 
	l_value text; -- temp value
begin
	if p_period_empty then 
		return false;
	elsif p_period_full then 
		return true;
	end if;
	
	l_periods = string_to_array(p_period,'U');
	foreach l_period slice 0 in array l_periods loop
		if left(l_period,4) = ']-oo' then
			l_min = null;
			l_min_in = false;
		else 
			select substr(split_part(l_period,';',1),2) into l_value;
			select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_min;
			if left(l_period,1) = '[' then 
				l_min_in = true;
			else
				l_min_in = false;
			end if;
		end if;
		-- parse right part of the interval
		if right(l_period, 4) = '+oo[' then 
			l_max = null;
			l_max_in = false;
		else
			select split_part(l_period,';',2) into l_value;
			select left(l_value, length(l_value)-1) into l_value;
			select TO_TIMESTAMP(l_value, 'YYYY-MM-DD HH24:MI:ss') into l_max;
			if right(l_period,1) = '[' then 
				l_max_in = false;
			else
				l_max_in = true;
			end if;
		end if;		
		
		-- then, test if date is in period
		if l_min = p_moment then
			return l_min_in;
		elsif l_max = p_moment then
			return l_max_in;
		elsif l_min < p_moment and p_moment < l_max then 
			return true;
		end if;
	end loop;

	return false;
end; $$; 

alter function sgraphs.IsPeriodActiveAtTimestamp owner to upa;

-- sgraphs.ActiveEntitiesValuesAt returns all active values at a given time
create or replace function sgraphs.ActiveEntitiesValuesAt(
	p_moment timestamp without time zone
) returns table (entity_id text, attribute_name text, attribute_value text, traits text[]) 
language plpgsql as $$
begin
return query 
	with 
	active_entities as (
		select ENT.element_id as entity_id
		from sgraphs.elements ENT 
		join sgraphs.periods PER on PER.period_id = ENT.element_period
		where ENT.element_type in (1,10)
		and sgraphs.isperiodactiveattimestamp(PER.period_empty, PER.period_full, PER.period_value, p_moment)
	), active_entities_traits as (
		select ENT.entity_id, array_agg(TRA.trait) as traits
		from active_entities ENT 
		join sgraphs.element_trait ETA on ETA.element_id = ENT.entity_id
		join sgraphs.traits TRA on TRA.trait_id = ETA.trait_id
		group by ENT.entity_id
	), active_attributes as (
		select ENT.entity_id, EAT.attribute_name, EAT.attribute_value
		from active_entities ENT 
		join sgraphs.entity_attributes EAT on EAT.entity_id = ENT.entity_id
		join sgraphs.periods EPER on EPER.period_id = EAT.period_id 
		where sgraphs.isperiodactiveattimestamp(EPER.period_empty, EPER.period_full, EPER.period_value, p_moment)
	)
	select ENT.entity_id, AAT.attribute_name, AAT.attribute_value, AET.traits
	from active_entities ENT
	left outer join active_attributes AAT on AAT.entity_id = ENT.entity_id 
	left outer join active_entities_traits AET on AET.entity_id = ENT.entity_id 
;
end; $$; 

alter function sgraphs.ActiveEntitiesValuesAt owner to upa;

-- sgraphs.ElementRelationsCountAtMoment returns the number of matches per trait, role and active status of an element id at a given time. 
-- For instance, let X be a person with N followers (A active, B inactive), result would be 
-- follower | subject | true  | A 
-- follower | subject | false | B
create or replace function sgraphs.ElementRelationsCountAtMoment(
	p_id text, p_moment timestamp without time zone 
) returns table(relation_trait text, relation_role text, active_relation bool, counter bigint)
language plpgsql as $$
begin
return query 
with 
current_neighborhoods as (
	select distinct 
	ELT.element_id as relation_id, 
	sgraphs.isperiodactiveattimestamp(PER.period_empty, PER.period_full, period_value, p_moment) as active_relation,
	RRO.role_in_relation
	from sgraphs.elements ELT
	join sgraphs.periods PER on PER.period_id = ELT.element_period
	join sgraphs.relation_role RRO on ELT.element_id = RRO.relation_id
	where not PER.period_empty 
	and p_id = ANY (RRO.role_values)
),
current_neighborhoods_traits as (
	select distinct 
	relation_id, TRA.trait, CUR.role_in_relation, CUR.active_relation
	from current_neighborhoods CUR
	left outer join sgraphs.element_trait ETA on CUR.relation_id = ETA.element_id
	left outer join sgraphs.traits TRA on TRA.trait_id = ETA.trait_id
)
select 
CNT.trait as relation_trait, 
CNT.role_in_relation as relation_role, 
CNT.active_relation as active_relation, 
count(distinct CNT.relation_id) as counter
from current_neighborhoods_traits CNT
group by CNT.trait, CNT.role_in_relation, CNT.active_relation;
end; $$;

alter function sgraphs.ElementRelationsCountAtMoment owner to upa;

-- sgraphs.ElementRelationsOperandsCountAtMoment details, for each relation with this id as a parameter, 
-- the traits, roles, activity and values (sorted to avoid permutations explosion) at a given time. 
-- It is basically  sgraphs.ElementRelationsCountAtMoment with sorted operands. 
create or replace function sgraphs.ElementRelationsOperandsCountAtMoment(
	p_id text, p_moment timestamp without time zone 
) returns table(relation_trait text, relation_role text, active_relation bool, relation_sorted_values text[], counter bigint)
language plpgsql as $$
begin
return query 
with 
current_neighborhoods as (
	select 
	ELT.element_id as relation_id, 
	sgraphs.isperiodactiveattimestamp(PER.period_empty, PER.period_full, period_value, p_moment) as active_relation,
	RRO.role_in_relation, 
	unnest(RRO.role_values) as role_operand
	from sgraphs.elements ELT
	join sgraphs.periods PER on PER.period_id = ELT.element_period
	join sgraphs.relation_role RRO on ELT.element_id = RRO.relation_id
	where not PER.period_empty 
	and p_id = ANY (RRO.role_values)
),
current_aggregated_neighbors as (
	select relation_id, CUR.role_in_relation, CUR.active_relation, 
	array_agg(CUR.role_operand order by CUR.relation_id, CUR.role_in_relation, CUR.active_relation, CUR.role_operand) as role_operands
	from current_neighborhoods CUR
	group by CUR.relation_id, CUR.role_in_relation, CUR.active_relation
),
current_neighborhoods_traits as (
	select relation_id, TRA.trait, CUR.role_in_relation, CUR.active_relation, role_operands
	from current_aggregated_neighbors CUR
	left outer join sgraphs.element_trait ETA on CUR.relation_id = ETA.element_id
	left outer join sgraphs.traits TRA on TRA.trait_id = ETA.trait_id
)
select 
CNT.trait as relation_trait, 
CNT.role_in_relation as relation_role, 
CNT.active_relation as active_relation, 
CNT.role_operands as relation_sorted_values, 
count(distinct CNT.relation_id) as counter
from current_neighborhoods_traits CNT
group by CNT.trait, CNT.role_in_relation, CNT.active_relation, CNT.role_operands;
end; $$;

alter function sgraphs.ElementRelationsOperandsCountAtMoment owner to upa;

-- sgraphs.ClearAll() deletes all the content 
create or replace procedure sgraphs.ClearAll() language plpgsql as $$
declare 
begin 
	delete from sgraphs.element_trait;
	delete from sgraphs.relation_role;
	delete from sgraphs.entity_attributes;
	delete from sgraphs.elements;
	delete from sgraphs.traits;
	delete from sgraphs.reftypes;
	delete from sgraphs.periods;
end; $$;

alter procedure sgraphs.ClearAll() owner to upa;

-- sgraphs.InitSchema cleans the schema and insert base data
create or replace procedure sgraphs.InitSchema() language plpgsql as $$
declare 
begin 
call sgraphs.ClearAll();

insert into sgraphs.reftypes(reftype_id, reftype_description) values(1,'entity only');
insert into sgraphs.reftypes(reftype_id, reftype_description) values(2,'relation only');
insert into sgraphs.reftypes(reftype_id, reftype_description) values(10, 'mixed');
end; $$;

alter procedure sgraphs.InitSchema() owner to upa;
