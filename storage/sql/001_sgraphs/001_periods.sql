create or replace function sgraphs.empty_intersection(
		min_a timestamp without time zone,
		min_in_a bool,
		max_a timestamp without time zone,
		max_in_a bool,
		min_b timestamp without time zone,
		min_in_b bool,
		max_b timestamp without time zone,
		max_in_b bool
	) returns bool language plpgsql as $$
declare
	res_min timestamp without time zone;
	res_max timestamp without time zone;
	res_min_in bool;
	res_max_in bool;
begin 
	-- test infinite left or right match
	if min_a is null and min_b is null then 
		return false;
	elsif max_a is null and max_b is null then 
		return false;
	end if;
	-- find min
	if min_a is null then 
		select min_b into res_min;
		select min_in_b into res_min_in;
	elsif min_b is null then 
		select min_a into res_min;
		select min_in_a into res_min_in;
	elsif min_a = min_b then 
		select min_a into res_min;
		select min_in_a and min_in_b into res_min_in;
	elsif min_a < min_b then 
		select min_b into res_min;
		select min_in_b into res_min_in;
	else
		select min_a into res_min;
		select min_in_a into res_min_in;
	end if;
	-- find max
	if max_a is null then 
		select max_b into res_max;
		select max_in_b into res_max_in;
	elsif max_b is null then 
		select max_a into res_max;
		select max_in_a into res_max_in;
	elsif max_a = max_b then 
		select max_b into res_max;
		select (max_in_a and max_in_b) into res_max_in;
	elsif max_a < max_b then 
		select max_a into res_max;
		select max_in_a into res_max_in;
	else
		select max_b into res_max;
		select max_in_b into res_max_in;
	end if;
	-- intersection interval is built
	-- then, decide whether said interval is empty or not 
	if res_min is null or res_max is null then 
		return false;
	elsif res_min = res_max then 
		return not (res_min_in and res_max_in);
	elsif res_min > res_max then 
		return true;
	else
		return false;
	end if;
end; $$;

create or replace function sgraphs.are_period_disjoin_with_interval(p_interval text, p_period text) returns bool language plpgsql as $$
declare
    -- split interval parameter
	l_interval_left text;
	l_interval_left_value timestamp without time zone;
	l_interval_right text; 
	l_interval_right_value timestamp without time zone;
	l_interval_split text[];
    l_interval_left_in bool;
    l_interval_right_in bool;
    -- split period
	l_period_split text[];
	l_period_left text;
	l_period_left_value timestamp without time zone;
	l_period_right text; 
	l_period_right_value timestamp without time zone;
    l_period_left_in bool;
    l_period_right_in bool;
	-- loop iteration 
    l_period_element text;
    -- local comparison 
    l_local_disjoin bool;
begin 
	if p_interval = '];[' then 
        return true;
    elsif p_interval = ']-oo;+oo[' then 
        return false;
    end if;
    -- interval value split 
    select string_to_array(p_interval,';') into l_interval_split;
    select replace(replace(l_interval_split[1],']',''), '[','') into l_interval_left;
    select replace(replace(l_interval_split[2],']',''), '[','') into l_interval_right;
    if l_interval_left = '-oo' then 
        select null into l_interval_left_value;
    else
        select l_interval_left::timestamp without time zone into l_interval_left_value;
    end if;
    if l_interval_right = '+oo' then 
        select null into l_interval_right_value;
    else
        select l_interval_right::timestamp without time zone into l_interval_right_value;
    end if;
    select (left(p_interval, 1) = '[') into l_interval_left_in;
    select (right(p_interval, 1) = ']') into l_interval_right_in;


	foreach l_period_element in array string_to_array(p_period,'U') loop 
		if l_period_element = '];[' then 
            continue;
        elsif l_period_element = ']-oo;+oo[' then 
            return false;
        else
            -- period value split 
            select string_to_array(l_period_element,';') into l_period_split;
            select replace(replace(l_period_split[1],']',''), '[','') into l_period_left;
            select replace(replace(l_period_split[2],']',''), '[','') into l_period_right;
            if l_period_left = '-oo' then 
                select null into l_period_left_value;
            else
                select l_period_left::timestamp without time zone into l_period_left_value;
            end if;
            if l_period_right = '+oo' then 
                select null into l_period_right_value;
            else
                select l_period_right::timestamp without time zone into l_period_right_value;
            end if;
            select (left(l_period_element, 1) = '[') into l_period_left_in;
            select (right(l_period_element, 1) = ']') into l_period_right_in;
            -----------------------------------------------
            -- then, we may test if values are separated --
            -----------------------------------------------
            select sgraphs.empty_intersection(
                l_interval_left_value, l_interval_left_in, 
                l_interval_right_value, l_interval_right_in,
                l_period_left_value, l_period_left_in, 
                l_period_right_value, l_period_right_in
            ) into l_local_disjoin;

            if not l_local_disjoin then 
                return false;
            end if; 
        end if;
    end loop;

    return true;
end; $$;

create or replace function sgraphs.are_periods_disjoin(p_period text, p_other_period text) returns bool language plpgsql as $$
declare
    -- split period
	l_period_split text[];
	l_period_left text;
	l_period_left_value timestamp without time zone;
	l_period_right text; 
	l_period_right_value timestamp without time zone;
    l_period_left_in bool;
    l_period_right_in bool;
	-- loop iteration 
    l_period_element text;
    -- local comparison 
    l_local_disjoin bool;
begin 
	foreach l_period_element in array string_to_array(p_period,'U') loop 
		if l_period_element = '];[' then 
            continue;
        elsif l_period_element = ']-oo;+oo[' then 
            return false;
        else
            select sgraphs.are_period_disjoin_with_interval(l_period_element, p_other_period) into l_local_disjoin;
            if not l_local_disjoin then 
                return false;
            end if;             
        end if;
    end loop;

    return true;
end; $$;