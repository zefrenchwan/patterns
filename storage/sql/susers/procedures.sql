-- returns a new random string, one per call
create or replace function susers.generate_random_string() returns text language plpgsql as $$
declare
	l_result text;
begin
	select md5(random()::text) || md5(random()::text) || md5(random()::text) into l_result;
	return l_result;
end; $$;

alter function susers.generate_random_string owner to upa;

-- insert user (to boostrap the system)
create or replace procedure susers.insert_user(
    p_login text, 
    p_pass text
) language plpgsql as $$
declare 
	l_salt text; -- user specific salt
	l_text_hash text; -- value for password with salt 
begin
    -- exit if user already exists 
	if exists (select 1 from susers.users where user_login = p_login) then
		raise exception 'user already exists' using errcode = '42710';
		return;
	end if;
    -- generate random salt 
	select susers.generate_random_string() into l_salt;
	select encode(sha256((p_pass || l_salt)::bytea), 'base64') into l_text_hash;
    -- insert value
	insert into susers.users(user_id, user_active, user_login, user_salt, user_secret, user_hash)
	select gen_random_uuid(), true, p_login, l_salt, susers.generate_random_string(), l_text_hash;
end; $$;

alter procedure susers.insert_user owner to upa;

-- susers.insert_super_user_roles uses an existing user and grants it all access 
create or replace procedure susers.insert_super_user_roles(p_login text)
language plpgsql as $$
declare 
begin 
	delete from susers.authorizations
	where auth_user_id in (
		select USR.user_id
		from susers.users USR
		where user_login = p_login
	);

	insert into susers.authorizations (
		auth_active,
		auth_user_id,
		auth_role_id,
		auth_class_id,
		auth_resource)
	select true, USR.user_id, role_id, class_id, null
	from susers.users USR
	cross join susers.classes CLA 
	cross join susers.roles ROL
	where user_login = p_login;
end;$$;

alter procedure susers.insert_super_user_roles owner to upa;

-- susers.test_user_password tests user: 
-- login and password are provided, it returns true if auth is valid
create or replace function susers.test_user_password(p_login text, p_password text) returns bool 
language plpgsql as $$
declare 
	l_salt text;
	l_hash text;
	l_value_hash text;
begin
	select user_hash, user_salt into l_hash, l_salt 
	from susers.users 
	where user_login = p_login
	and user_active = true;

	if l_hash is null or length(l_hash) = 0 or length(p_password) = 0 then 
		return false;
	end if;

	select encode(sha256((p_password || l_salt)::bytea), 'base64') into l_value_hash;
	return l_value_hash is not distinct from l_hash;
end; $$;

alter function susers.test_user_password owner to upa;

-- susers.find_secret_for_user loads secret for a given active user, raises exception otherwise
create or replace function susers.find_secret_for_user(p_login text) returns text 
language plpgsql
as $$
declare
	l_found bool = false;
	l_secret text;
begin 

	select true, user_secret into l_found, l_secret
	from susers.users
	where user_login = p_login
	and user_active = true;
	
	if l_found is null or not l_found then 
		raise exception 'auth failure: no active user found for login %', p_login using errcode = '42501';
	end if;

	return l_secret;
end;$$;

alter function susers.find_secret_for_user owner to upa;

-- susers.user_class_authorizations returns the table of role, override and resouces
-- per login and class. 
-- Values are active. 
create or replace function susers.user_class_authorizations(
	p_login in text, p_class text
) returns table (role_name text, overridden bool, auth_resource text) 
language plpgsql as $$
declare 
begin 
	return query 
	with class_names as (
		select class_id
		from susers.classes CLA
		where CLA.class_name = p_class
	)
	select ROL.role_name, 
	case when CNA.class_id is null then false else true end as overridden,
	AUT.auth_resource
	from susers.authorizations AUT
	join susers.roles ROL on ROL.role_id = AUT.auth_role_id 
	join susers.users USR on USR.user_id = AUT.auth_user_id
	join class_names CNA on CNA.class_id = AUT.auth_class_id
	where AUT.auth_active
	and USR.user_active
	and USR.user_login = p_login;
end; $$;

alter function susers.user_class_authorizations owner to upa;

-- given a login, a class and a resource, get all, if any, 
create or replace function susers.roles_for_resource(p_login text, p_class text, p_resource text)
returns text[] language plpgsql as $$
declare 
	l_result text[];
begin 

with authorizations as (
	SELECT *
	from susers.user_class_authorizations(p_login,p_class)
), active_roles as (
	select role_name
	from authorizations 
	where overridden
	UNION ALL 
	select role_name
	from authorizations 
	where not overridden
	and auth_resource is not null 
	and auth_resource = p_resource
)
select array_agg(role_name) into l_result
from active_roles;

return l_result;
end; $$;

alter function susers.roles_for_resource owner to upa;

-- susers.upsert_user changes password of an existing user, or creates user if possible. 
create or replace procedure susers.upsert_user(
	p_creator in text, p_login in text, p_new_password in text) 
language plpgsql as $$
declare 
	-- is creator user active
	l_active bool;
	-- id of the user, if any, to upsert password for
	l_user text;
	-- all access rights, if any, for p_creator
	l_access_rights text[];
	-- true means authorized, false for no authorization for creator
	l_authorized bool;
	-- salt of the user
	l_salt text; 
	-- hashed password
	l_text_hash text;
begin 

	-- creator has to be active. Otherwise raise exception  
	select user_active into l_active
	from susers.users 
	where user_login = p_creator;

	if l_active is null or not l_active then 
		raise exception 'auth failure: no access for %', p_creator using errcode = '42501';
	end if;

	-- find other user to act on
	select user_id into l_user 
	from susers.users 
	where user_login = p_login;

	if l_user is null then 
		raise exception 'auth failure: no user %', p_login using errcode = '42501';
	end if;

	select susers.roles_for_resource(p_creator, 'user', l_user) into l_access_rights;

	select 'manager' = ANY(susers.roles_for_resource(p_creator,'user',l_user)) into l_authorized;

	if p_creator = p_login and not l_authorized then 
		select 'modifier' = ANY(susers.roles_for_resource(p_creator,'user',l_user)) into l_authorized;
	end if;

	if not l_authorized then 
		raise exception 'auth failure: % unauthorized', p_creator using errcode = '42501';
	end if;

	select susers.generate_random_string() into l_salt;
	
	select encode(sha256((p_new_password || l_salt)::bytea), 'base64') into l_text_hash;
    
	if l_user is null then
		insert into susers.users(user_id, user_active, user_login, user_salt, user_secret, user_hash)
		select gen_random_uuid(), true, p_login, l_salt, susers.generate_random_string(), l_text_hash;
	else 
		update susers.users 
		set user_salt = l_salt, 
		user_secret = susers.generate_random_string(), 
		user_hash = l_text_hash
		where user_id = l_user;
	end if;	
end; $$;

alter procedure susers.upsert_user owner to upa;

-- susers.test_if_resource_exists returns true if resource exists by id for a given class, false otherwise
create or replace function susers.test_if_resource_exists(p_class text, p_resource text) 
returns bool language plpgsql as $$
declare 
	l_found bool;
begin 
	if p_class = 'user' then 
		select true into l_found 
		from sgraphs.users
		where user_id = p_resource;
	elsif p_class = 'graph' then 
		select true into l_found 
		from sgraphs.graphs 
		where graph_id = p_resource;
	else 
		raise exception 'unexpected class %', p_class using errcode = '42704';
	end if;

	return l_found is not null and l_found = true;
end; $$;

alter function susers.test_if_resource_exists owner to upa;

-- susers.add_auth_for_user_on_resource grants an user authorization. 
-- There is NO parameter to define creator, has to be done before. 
-- Note that p_resource set to null is not allowed, would be too risky.  
create or replace procedure susers.add_auth_for_user_on_resource(
	p_user_login text, p_auth text, p_class text, p_resource text
) language plpgsql as $$
declare 
	l_found bool;
	l_user_id text;
	l_role_id int;
	l_class_id int;
begin 
	select user_id into l_user_id
	from susers.users 
	where user_active and user_login = p_user_login;

	if l_user_id is null then 
		raise exception 'auth failure: no active user %', p_user_login using errcode = '42501';
	end if;
	-- user exists and is active

	select role_id into l_role_id
	from susers.roles 
	where role_name = p_auth;

	if l_role_id is null then  
		raise exception '% is not a valid role',  p_auth using errcode = '42704';
	end if;
	-- role exists

	select class_id into l_class_id 
	from susers.classes 
	where class_name = p_class;

	if l_class_id is null then 
		raise exception '% is not a valid class', p_class using errcode = '42704';
	end if;
	-- class exists

	if p_resource is not null then 
		select susers.test_if_resource_exists(p_class, p_resource) into l_found;
		if not l_found then 
			raise exception 'resource not found: non existing resource %', p_resource using errcode = 'P0002';
		end if;
	else
		raise exception 'resource shoud not be null' using errcode = '39004';
	end if;
	-- resource is valid for that class

	-- THEN, test if previous authorizations were better than current one. 
	-- If not, insert. 
	if exists (
		select 1 from susers.authorizations 
		where auth_active and auth_user_id = l_user_id 
		and auth_role_id = l_role_id and auth_class_id = l_class_id
		and auth_resource is null 
	) then 
		return;
	end if;

	if not exists (
		select 1 from susers.authorizations 
		where auth_active and auth_user_id = l_user_id 
		and auth_role_id = l_role_id and auth_class_id = l_class_id
		and auth_resource = p_resource
	) then 
		insert into susers.authorizations (auth_active, auth_user_id, auth_role_id, auth_class_id, auth_resource)
		select true, l_user_id, l_role_id, l_class_id, p_resource;
	end if;
end; $$;

alter procedure susers.add_auth_for_user_on_resource owner to upa;

-- susers.accept_any_user_access_to_resource_or_raise tests all security access and existence. 
-- If all tests pass, it does nothing more. Otherwise, it raises an exception for the first failing test.
create or replace procedure susers.accept_any_user_access_to_resource_or_raise(p_user_login text, p_class text, p_role_names text[], p_resource text) 
language plpgsql as $$
declare
	l_resource text;
	l_found bool;
	l_user_id text;
	l_role text;
	l_role_id int;
	l_class_id int;
begin 
	select user_id into l_user_id
	from susers.users 
	where user_active 
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
		select susers.test_if_resource_exists(p_class, p_resource) into l_found;
		if not l_found then 
			raise exception 'resource not found: non existing resource %', p_resource using errcode = 'P0002';
		end if;
		select p_resource into l_resource;
	else
		select null into l_resource;
	end if;
	-- resource is valid for that class

	foreach l_role in array p_role_names loop 

		select role_id into l_role_id
		from susers.roles 
		where role_name = l_role;

		if l_role_id is null then  
			raise exception '% is not a valid role',  p_auth using errcode = '42704';
		end if;
		-- role exists

		if exists (
			select 1 from susers.authorizations 
			where auth_active and auth_user_id = l_user_id 
			and auth_role_id = l_role_id and auth_class_id = l_class_id
			and (
				auth_resource is null 
				or (l_resource is not null and auth_resource = l_resource))
		) then 
			return;
		end if;

		if l_resource is not null and exists (
			select 1 from susers.authorizations 
			where auth_active and auth_user_id = l_user_id 
			and auth_role_id = l_role_id and auth_class_id = l_class_id
			and auth_resource = l_resource
		) then 
			return;
		end if;
	end loop;

	if array_length(p_role_names) > 0 then 
		-- no match found and one was necessary
		raise exception 'auth failure: unauthorized' using errcode = '42501';
	end if;
end; $$;

alter procedure susers.accept_any_user_access_to_resource_or_raise owner to upa;

-- susers.list_authorized_graphs_for_any_roles finds graphs an user may access for given roles
create or replace function susers.list_authorized_graphs_for_any_roles(p_user_login text, p_roles text[]) 
returns table(graph_id text, role_names text[]) language plpgsql as $$
declare 
	l_user_id text;
	l_class_id int;
	l_matching_roles int[];
	l_role text;
begin 
	select USR.user_id into l_user_id
	from susers.users USR
	where USR.user_login = p_user_login
	and USR.user_active = true;

	select array_agg(ROL.role_id) into l_matching_roles
	from susers.roles ROL 
	where ROL.role_name = ANY(p_roles);

	select CLA.class_id into l_class_id 
	from susers.classes CLA 
	where CLA.class_name = 'graph';
	
	if l_class_id is null then 
		raise exception 'invalid class provided' using errcode = '42704';
	end if;

	return query 
	with roles_resources as (
		select AUT.auth_resource as graph_id, ROL.role_name
		from susers.authorizations AUT
		join susers.roles ROL on AUT.auth_role_id = ROL.role_id
		where l_user_id is not null 
		and AUT.auth_user_id = l_user_id
		and AUT.auth_active = true
		and AUT.auth_role_id = ANY(l_matching_roles)
		and AUT.auth_class_id = l_class_id
		and AUT.auth_resource is null
	), roles_graphs as (
		select GRA.graph_id, ROR.role_name
		from sgraphs.graphs GRA 
		join roles_resources ROR on ROR.graph_id = GRA.graph_id
		UNION 
		select GRA.graph_id, ROR.role_name
		from roles_resources ROR
		cross join sgraphs.graphs GRA
		where ROR.graph_id is null 
	) 
	select distinct ROG.graph_id, array_agg(ROG.role_name) as role_names
	from roles_graphs ROG
	group by ROG.graph_id;
end;$$;

alter function susers.list_authorized_graphs_for_any_roles owner to upa;