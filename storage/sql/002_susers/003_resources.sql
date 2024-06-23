------------------------------------------------------------
--       UNSECURED PART, TO ACT ON LOW LEVEL STORAGE      --
------------------------------------------------------------
------------------------------------------------------------
-- Do not use it directly, perform security checks before --
-------------------------------------------------------------

-- susers.insert_new_resource inserts a new resource 
create or replace procedure susers.insert_new_resource(p_user_login text, p_class_name text, p_new_id text) 
language plpgsql as $$
declare 
    l_class_id int;
begin 
    select class_id into l_class_id from susers.classes where class_name = p_class_name;
    if l_class_id is null then 
		raise exception 'unexpected class %', p_class_name using errcode = '42704';	
	end if;

    if exists (select 1 from susers.resources where resource_id = p_new_id and resource_type = l_class_id) then 
        raise exception 'resource % already inserted', p_new_id using errcode = '42710';
    end if;

    if not exists (select 1 from susers.users where user_login = p_user_login) then 
        raise exception 'no creator %', p_user_login using errcode = 'P0002';
    end if;

    if p_class_name = 'graph' and not exists (select 1 from sgraphs.graphs where graph_id = p_new_id) then 
        raise exception 'no graph %', p_new_id using errcode = 'P0002';
    elsif p_class_name = 'user' and not exists (select 1 from susers where user_id = p_new_id) then 
        raise exception 'no user %', p_new_id using errcode = 'P0002';
    end if;

    insert into susers.resources(resource_id, resource_type, resource_creator_login)
    select p_new_id, l_class_id, p_user_login;
end;$$;

-- susers.delete_resource deletes the reource in the resource table, not in the graph or user tables
create or replace procedure susers.delete_resource(p_resource_class text, p_resource_id text) 
language plpgsql as $$
declare 
 l_class_id int;
begin 
    select class_id into l_class_id from susers.classes where class_name = p_class_name;
    if l_class_id is null then 
		raise exception 'unexpected class %', p_class_name using errcode = '42704';	
	end if;

	delete from susers.resources where resource_id = p_resource and resource_type = l_class_id;
end; $$;

-- susers.all_graphs_authorized_for_user 
create or replace function susers.all_graphs_authorized_for_user(p_login text)
returns table (resource text, role_names text[]) language plpgsql as $$
declare
begin 
	return query 
	with all_graphs_current_auths as (
		select AUA.role_name, AUA.auth_all_resources, 
		AUA.resource_included, AUA.resource
		from susers.all_users_authorizations AUA
		where AUA.class_name = 'graph'
		and AUA.user_login = p_login
		and AUA.user_active = true
	), all_resources_unauth as (
		select AGC.resource, AGC.role_name
		from all_graphs_current_auths AGC 
		where AGC.resource_included = false
		and AGC.resource is not null
	), all_resources_auth as (
		select GRA.graph_id as resource, AGC.role_name
		from all_graphs_current_auths AGC
		cross join sgraphs.graphs GRA
		where AGC.resource is null
		and auth_all_resources = true
	), specific_resources_auth as (
		select AGC.resource, AGC.role_name
		from all_graphs_current_auths AGC
		where AGC.resource_included = true
		and AGC.resource is not null
	), all_auths as (
		select distinct SRA.resource, SRA.role_name 
		from specific_resources_auth SRA
		union 
		select distinct ARA.resource, ARA.role_name 
		from all_resources_auth ARA
	)
	select ALA.resource, array_agg(distinct ALA.role_name order by ALA.role_name) 
	from all_auths ALA
	left outer join all_resources_unauth ARU on ARU.resource = ALA.resource and ALA.role_name = ARU.role_name
	where ARU.resource is null and ARU.role_name is null
	group by ALA.resource ;
end; $$;

-- susers.change_access_to_user_for_resource is general core procedure 
-- to grant or revoke authorizations to user. 
-- NO interest for a direct call, use grant or revoke alias procedures. 
create or replace procedure susers.change_access_to_user_for_resource(
	p_user_id text, p_class_name text, p_role_name text, 
	p_grant_access bool, p_all_resources bool, p_resource text
) language plpgsql as $$
declare 
	l_class_id int;
	l_role_id int;
	l_auth_all bool;
	l_auth_id bigint;
begin 
	select role_id into l_role_id from susers.roles where role_name = p_role_name;
	if l_role_id is null then 
		raise exception 'unexpected role %', p_role_name using errcode = '42704';	
	end if;

	select class_id into l_class_id from susers.classes where class_name = p_class_name;
	if l_class_id is null then 
		raise exception 'unexpected class %', p_class_name using errcode = '42704';	
	end if;

	if not exists (select 1 from susers.users where user_id = p_user_id) then 
		raise exception 'no user %', p_user_id using errcode = 'P0002';
	end if;

	if p_grant_access is null then
		raise exception 'invalid grant / revoke parameter' using errcode = '42704';
	end if;

	if (p_all_resources = true and p_resource is not null) or (p_resource is null and p_all_resources = false) or p_all_resources is null then 
		raise exception 'invalid auth parameters: resource and all access do not match' using errcode = 'P0001';
	end if; 

	-- all values are valid, start processing 

	-- Exclude values from the opposite side:
	-- If we grant values, then remove revoked said values. 
	-- If we revoke values, then remove granted said values.
	-- Either specific resource, or all resources, 
	-- each delete affects one case and one case only. 
	delete from susers.resources_authorizations
	where resource_included = not p_grant_access 
	and p_all_resouces and p_resource is null
	and auth_role_id = l_role_id 
	and auth_class_id = l_class_id
	and auth_user_id = p_user_id;
	delete from susers.resources_authorizations
	where resource_included = not p_grant_access 
	and not p_all_resouces and resource = p_resource
	and auth_role_id = l_role_id 
	and auth_class_id = l_class_id
	and auth_user_id = p_user_id;
	-- if all resources are set, then clear specific cases
	-- and clear previous values to force the new one 
	if p_all_resources then 
		delete from susers.resources_authorizations
		where resource_included = p_grant_access
		and auth_role_id = l_role_id 
		and auth_class_id = l_class_id
		and auth_user_id = p_user_id;
		delete from susers.authorizations 
		where auth_role_id = l_role_id 
		and auth_class_id = l_class_id
		and auth_user_id = p_user_id;
		-- finally, insert values. 
		-- No need to insert to exclude all, just insert if it is to grant access. 
		insert into susers.authorizations(auth_all_resources, auth_user_id, auth_role_id, auth_class_id)
		select true, p_user_id, l_role_id, l_class_id
		where p_grant_access = true;
		-- and we are done for this special case
		return;
	end if;

	-- we exited with all resources access. 
	-- Then, in here, deal with specific resource 
	select auth_id, auth_all_resources into l_auth_id, l_auth_all
	from susers.authorizations
	where auth_role_id = l_role_id 
	and auth_class_id = l_class_id
	and auth_user_id = p_user_id;

	if l_auth_id is null then 
		-- no previous value 
		insert into susers.authorizations(auth_all_resources, auth_user_id, auth_role_id, auth_class_id)
		select false, p_user_id, l_role_id, l_class_id
		returning auth_id into l_auth_id;
	elsif l_auth_all = true then 
		-- previous value is all resources allowed, no need to insert this specific value 
		return;
	elsif not exists (
            select 1 from susers.resources_authorizations
            where auth_id = l_auth_id and resource = p_resource and resource_included = p_grant_access
        ) then  
            -- insert specific resource case because we don't have it yet
            insert into susers.resources_authorizations(auth_id, resource, resource_included)
            select l_auth_id, p_resource, p_grant_access where p_resource is not null;  
	end if;
end;$$;

alter procedure susers.change_access_to_user_for_resource owner to upa;

-- susers.grant_access_to_user_for_resource grants access to resource (not null) or all resources (if null)
create or replace procedure susers.grant_access_to_user_for_resource(
	p_user_id text, p_class_name text, p_role_name text, p_resource text
) language plpgsql as $$
declare 
    l_user_id text;
	l_all_resources bool;
begin 
    select USR.user_name into l_user_id from susers.users USR where user_login = p_user_login;
    if l_user_id is null then 
        raise exception 'no user %', p_user_login using errcode = 'P0002';
    end if;

	select (p_resource is null) into l_all_resources;

	call susers.change_access_to_user_for_resource(l_user_id, p_class_name, p_role_name, true, l_all_resources, p_resource);
end;$$;

alter procedure susers.grant_access_to_user_for_resource owner to upa;

-- susers.revoke_access_to_user_for_resource revokes access to resource (not null) or all resources (if null)
create or replace procedure susers.revoke_access_to_user_for_resource(
	p_user_login text, p_class_name text, p_role_name text, p_resource text
) language plpgsql as $$
declare 
    l_user_id text;
	l_all_resources bool;
begin 
    select USR.user_name into l_user_id from susers.users USR where user_login = p_user_login;
    if l_user_id is null then 
        raise exception 'no user %', p_user_login using errcode = 'P0002';
    end if;

	select (p_resource is null) into l_all_resources;

	call susers.change_access_to_user_for_resource(l_user_id,  p_class_name, p_role_name, false, l_all_resources, p_resource);
end;$$;

alter procedure susers.revoke_access_to_user_for_resource owner to upa;

-- susers.accept_user_access_to_resource_or_raise test if user has access to given resource
create or replace procedure susers.accept_user_access_to_resource_or_raise(p_user_login text, p_class text, p_role_names text[], p_all_roles bool, p_resource text) 
language plpgsql as $$
declare
	l_resource text;
	l_found bool;
	l_user_id text;
	l_role text;
	l_role_id int;
	l_class_id int;
    l_all_matches bool;
    l_one_match bool;
    l_current_match bool;
begin 
	select user_id into l_user_id
	from susers.users 
	where user_active = true 
	and user_login = p_user_login;

	if l_user_id is null then 
		raise exception 'auth failure: no active user %', p_user_login using errcode = '42501';
	end if;
	-- user exists and is active

	select class_id into l_class_id 
	from susers.classes 
	where class_name = p_class;

	if l_class_id is null then 
		raise exception '% is not a valid class', p_class using errcode = '42704';
	end if;
	-- class exists

	if p_resource is not null then 
        if not exists (select 1 from susers.resources where resource_id = p_resource and resource_type = l_class_id) then 
			raise exception 'resource not found: non existing resource %', p_resource using errcode = 'P0002';
		end if;
	
    	select p_resource into l_resource;
	else
		select null into l_resource;
	end if;
	-- resource is valid for that class

    -- no role, no action, it is accepted
	if array_length(p_role_names) > 0 then 
        return;
    end if;

    -- process each role
    select true into l_all_matches;
    select false into l_one_match;
    -- for each role in parameters
	foreach l_role in array p_role_names loop 

		select role_id into l_role_id
		from susers.roles 
		where role_name = l_role;

		if l_role_id is null then  
			raise exception '% is not a valid role',  p_auth using errcode = '42704';
		end if;
		-- role exists

        with user_access as (
            select AUT.auth_all_resources, RAU.resource, RAU.resource_included
            from susers.authorizations AUT 
            left outer join susers.resources_authorizations RAU on RAU.auth_id = AUT.auth_id
            where AUT.auth_role_id = 4
            and AUT.auth_class_id = 2
            and AUT.auth_user_id = 'kfdlkfdlsk'
        ), specific_access_accept as (
            select 1 as validation
            from user_access UAC
            where UAC.auth_all_resources = false 
            and UAC.resource = 'opop'
            and UAC.resource_included = true
        ), all_access_reject as (
            select -100 as validation
            from user_access UAC
            where UAC.auth_all_resources = true 
            and UAC.resource = 'opop'
            and UAC.resource_included = false
        ), all_access_accept as (
            select 1 as validation
            from user_access UAC
            where UAC.auth_all_resources = true 
            and UAC.resource is null 
            and UAC.resource_included is null
        ), decision_table as (
            select validation
            from specific_access_accept
            UNION ALL 
            select validation
            from all_access_reject
            UNION ALL 
            select validation
            from all_access_accept
            UNION ALL
            select 0 as validation
        )
        select (sum(validation) > 0) into l_current_match 
        from decision_table;

        if not l_current_match then 
            select false into l_all_matches;
        else
            select true into l_one_match;
        end if;
	end loop;

    -- THEN, decide. 
    -- all matches, no matter the rest, it is accepted
    if l_all_matches then 
        return;
    end if;
    -- no match means refused anyway 
    if not l_one_match then 
        raise exception 'auth failure: unauthorized' using errcode = '42501';
    end if;
    -- expecting one match and got it
    if not p_all_roles then 
        return;
    end if;
    -- expecting all matches and did not have it
    if not l_all_matches and p_all_roles then 
		raise exception 'auth failure: unauthorized' using errcode = '42501';
    end if;
end; $$;

alter procedure susers.accept_user_access_to_resource_or_raise owner to upa;