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
		raise exception 'user already exists';
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
		raise exception 'no active user found for login %', p_login;
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
-- Result is out parameters: success and, if failure, if authorized.
create or replace procedure susers.upsert_user(
	p_creator in text, p_login in text, p_new_password in text, 
	p_success out bool, p_authorized out bool) 
language plpgsql as $$
declare 
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

	select user_id into l_user 
	from susers.users 
	where user_login = p_login;

	select susers.roles_for_resource(p_creator, 'user', l_user) into l_access_rights;

	select 'supervisor' = ANY(susers.roles_for_resource(p_creator,'user',l_user)) into l_authorized;

	if p_creator = p_login and not l_authorized then 
		select 'modifier' = ANY(susers.roles_for_resource(p_creator,'user',l_user)) into l_authorized;
	end if;

	if not l_authorized then 
		p_success = false;
		p_authorized = false;
		return;
	else
		p_authorized = true;
	end if;

	p_success = false;

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

	p_success = true;

end; $$;

alter procedure susers.upsert_user owner to upa;
