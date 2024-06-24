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
    elsif p_class_name = 'user' and not exists (select 1 from susers.users where user_id = p_new_id) then 
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
    select class_id into l_class_id from susers.classes where class_name = p_resource_class;
    if l_class_id is null then 
		raise exception 'unexpected class %', p_resource_class using errcode = '42704';	
	end if;

	delete from susers.resources where resource_id = p_resource_id and resource_type = l_class_id;
end; $$;


-- susers.authorizations_for_user returns authorized and unauthorized resources. 
-- It hides database details, and then should be used as the only data source. 
-- It returns the class, the resource (null for all auth), included or excluded, and aggregated roles
create or replace function susers.authorizations_for_user(p_user_login text)
returns table(class_name text, resource text, included bool, roles text[])
language plpgsql as $$
begin
	return query 
	with raw_auths as (
		-- raw authorizations, the full content
		select 
		CLA.class_name,
		ROL.role_name,
		AUT.auth_all_resources, 
		AUT.auth_inclusion, 
		RAU.resource
		from susers.authorizations AUT 
		join susers.users USR on USR.user_id = AUT.auth_user_id
		join susers.roles ROL on AUT.auth_role_id = ROL.role_id
		join susers.classes CLA on CLA.class_id = AUT.auth_class_id
		left outer join susers.resources_authorizations RAU on RAU.auth_id = AUT.auth_id
		where USR.user_active = true 
		and USR.user_login = p_user_login 
	), all_forbidden_resources as (
		-- get all unauthorized resources. Set is finite by construction 
		select RAA.class_name, RAA.role_name, RAA.resource, RAA.auth_inclusion  
		from raw_auths RAA
		where RAA.auth_all_resources = false
		and RAA.auth_inclusion = false 
		and RAA.resource is not null
	), specific_authorized_resources as (
		-- only the explicit set of authorized values with no better access (resource nul)
		select RAA.class_name, RAA.role_name, RAA.resource, RAA.auth_inclusion
		from raw_auths RAA
		where RAA.auth_all_resources = false
		and RAA.auth_inclusion = true
		and RAA.resource is not null
		and not exists (
			select 1 
			from raw_auths RAUT
			where 
			RAUT.auth_all_resources = true 
			and RAA.class_name = RAUT.class_name
			and RAA.role_name = RAUT.role_name
			and RAA.resource is null
			and RAUT.auth_inclusion = true
		)
	), specific_remaining_authorized_resources as (
		-- authorized - unauthorized
		select SAR.class_name, SAR.role_name, SAR.resource, SAR.auth_inclusion
		from specific_authorized_resources SAR
		left outer join all_forbidden_resources AFR on SAR.class_name = AFR.class_name
			and SAR.role_name = AFR.role_name and SAR.resource = AFR.resource
		where AFR.class_name is null and AFR.role_name is null
		and SAR.resource is not null 
	), remaining_unauthorized_resources as (
		select AFR.class_name, AFR.role_name, AFR.resource, AFR.auth_inclusion
		from all_forbidden_resources AFR
		left outer join specific_remaining_authorized_resources SRAS 
			on AFR.class_name = SRAS.class_name
			and AFR.role_name = SRAS.role_name
			and AFR.resource = SRAS.resource
		where SRAS.resource is null 
		and SRAS.class_name is null 
		and AFR.role_name is null
	), final_unaggregated_auths as (
		-- all global included auth
		select RAA.class_name, RAA.role_name, RAA.resource,  RAA.auth_inclusion
		from raw_auths RAA
		where RAA.auth_all_resources = true
		and RAA.auth_inclusion = true
		and RAA.resource is null
		UNION 
		-- all specfic auth for included
		select SRAR.class_name, SRAR.role_name, SRAR.resource,  SRAR.auth_inclusion
		from specific_remaining_authorized_resources  SRAR
		UNION 
		-- all specific unauthorized remaining values 
		select RUR.class_name, RUR.role_name, RUR.resource, RUR.auth_inclusion
		from remaining_unauthorized_resources RUR 
	)
	select 
	FUD.class_name, 
	FUD.resource,
	FUD.auth_inclusion as included, 
	array_agg(role_name)
	from final_unaggregated_auths FUD
	group by 
	FUD.class_name, 
	FUD.resource,
	FUD.auth_inclusion ;
end;$$;

alter function susers.authorizations_for_user owner to upa;

-- susers.authorizations_for_user_on_resource returns all roles and if they are granted for a given resource and a given login. 
-- Note that resource may be null.
-- This function does NOT raise error for invalid parameter.  
create or replace function susers.authorizations_for_user_on_resource(p_user_login text, p_class_name text, p_resource text)
returns table(role_name text, role_included bool) language plpgsql as $$
begin 
	if p_resource is not null and not exists (
		select 1 
		from susers.resources RES 
		join susers.classes CLA on CLA.class_id = RES.resource_type  
		where RES.resource_id = p_resource
		and CLA.class_name = p_class_name
	) then 
		return query select null, false where 1 != 1;
	end if;

	return query
	with all_auths_for_resource as (
		select AFU.resource, included, unnest(roles) as role_name
		from susers.authorizations_for_user(p_user_login) AFU
		where AFU.class_name = p_class_name 
		and (p_resource is null and AFU.resource is null) 
		or (p_resource is not null and AFU.resource = p_resource)
	), refused_resource_roles as (
		select AAFR.included, AAFR.role_name
		from all_auths_for_resource AAFR 
		where AAFR.included = false 
	), accepted_resource_roles as (
		select AAFR.included, AAFR.role_name
		from all_auths_for_resource AAFR 
		where AAFR.included = true 
	), remaining_resource_roles as (
		select distinct ARR.role_name  
		from accepted_resource_roles ARR 
		left outer join refused_resource_roles RRR on RRR.role_name = ARR.role_name 
		where RRR.role_name is null
		and ARR.role_name is not null  
	) 
	select RRR.role_name, true as role_included 
	from remaining_resource_roles RRR
	UNION 
	select RERO.role_name, false as role_included
	from refused_resource_roles RERO;
end;$$;

alter function susers.authorizations_for_user_on_resource owner to upa;


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
	if p_all_resources then 
		delete from susers.authorizations
		where auth_inclusion = not p_grant_access 
		and p_all_resources and p_resource is null
		and auth_role_id = l_role_id 
		and auth_class_id = l_class_id
		and auth_user_id = p_user_id;
	else 
		select auth_id into l_auth_id
		from susers.authorizations
		where auth_inclusion = not p_grant_access 
		and not p_all_resources 
		and auth_role_id = l_role_id 
		and auth_class_id = l_class_id
		and auth_user_id = p_user_id;

		delete from susers.resources_authorizations
		where auth_id = l_auth_id and resource = p_resource;

		if not exists (select 1 from susers.resources_authorizations where auth_id = l_auth_id) then 
			delete from susers.authorizations where auth_id = l_auth_id;
		end if;
	end if;
	
	-- if all resources are set, then clear specific cases
	-- and clear previous values to force the new one 
	if p_all_resources then 
		delete from susers.authorizations
		where auth_inclusion = p_grant_access
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
            where auth_id = l_auth_id and resource = p_resource and auth_inclusion = p_grant_access
        ) then  
            -- insert specific resource case because we don't have it yet
            insert into susers.resources_authorizations(auth_id, resource, auth_inclusion)
            select l_auth_id, p_resource, p_grant_access where p_resource is not null;  
	end if;
end;$$;

alter procedure susers.change_access_to_user_for_resource owner to upa;

-- susers.grant_access_to_user_for_resource grants access to resource (not null) or all resources (if null)
create or replace procedure susers.grant_access_to_user_for_resource(
	p_user_login text, p_class_name text, p_role_name text, p_resource text
) language plpgsql as $$
declare 
    l_user_id text;
	l_all_resources bool;
begin 
    select USR.user_id into l_user_id from susers.users USR where user_login = p_user_login;
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
	l_remaining_roles text[];
begin 

	with expected_roles as (
		select unnest(p_role_names) as role_name
	), granted_roles as (
		select AFU.role_name
		from susers.authorizations_for_user_on_resource(p_user_login, p_class, p_resource) AFU
		join expected_roles ERO on ERO.role_name = AFU.role_name 
		where role_included = true
	) 
	select array_agg(GRO.role_name) into l_remaining_roles
	from granted_roles GRO;

	if p_all_roles then 
		if not (p_role_names <@ l_remaining_roles) then
			raise exception 'no auth or no resource' using errcode = '42501';
		end if;
	elsif array_length(l_remaining_roles, 1) = 0  then 
		raise exception 'no auth or no resource' using errcode = '42501';
	end if;
end; $$;

alter procedure susers.accept_user_access_to_resource_or_raise owner to upa;

-- susers.all_resources_authorized_for_user returns all resources with all roles an user may use. 
-- Algorithm is to 
-- get all explicit auth resources (included and excluded)
-- get all authorized resources with null set (to say all but...) 
-- and perform the difference. 
create or replace function susers.all_resources_authorized_for_user(p_user_login text, p_class_name text) 
returns table(resource text, role_names text[]) language plpgsql as $$
declare 
    l_class_id int;
begin
    select class_id into l_class_id from susers.classes CLA where CLA.class_name = p_class_name;

    return query
    with all_auths as (
        select AFU.resource, AFU.included, unnest(AFU.roles) as role_name
        from susers.authorizations_for_user(p_user_login) AFU
        where class_name = p_class_name
    ), all_resources as (
		select RES.resource_id 
		from susers.resources RES
		where RES.resource_type = l_class_id
	), all_resource_auths as (
        select ALA.resource, ALA.included, ALA.role_name
        from all_auths ALA 
        join all_resources RES on RES.resource_id = ALA.resource
        where ALA.resource is not null 
        UNION 
        select RES.resource_id as resource, ALA.included, ALA.role_name
        from all_auths ALA 
        cross join all_resources RES 
        where ALA.resource is null 
        and ALA.included = true 
    ), reduced_auths as (
        select ARA.resource, ARA.role_name, 
		array_agg(distinct ARA.included) as role_inclusions 
        from all_resource_auths ARA 
        group by ARA.resource, ARA.role_name
    )
    select RAU.resource as resource, array_agg(RAU.role_name) as role_names 
    from reduced_auths RAU
    where true = ALL(RAU.role_inclusions)
	and array_length(RAU.role_inclusions, 1) > 0
	group by  RAU.resource;
end;$$;

alter function susers.all_resources_authorized_for_user owner to upa;

create or replace function susers.all_graphs_authorized_for_user(p_user_login text) 
returns table(resource text, role_names text[]) language plpgsql as $$
begin 
	return query select * from susers.all_resources_authorized_for_user(p_user_login, 'graph');
end;$$;

alter function susers.all_graphs_authorized_for_user owner to upa;

create or replace function susers.all_users_authorized_for_user(p_user_login text) 
returns table(resource text, role_names text[]) language plpgsql as $$
begin 
	return query select * from susers.all_resources_authorized_for_user(p_user_login, 'user');
end;$$;

alter function susers.all_users_authorized_for_user owner to upa;
