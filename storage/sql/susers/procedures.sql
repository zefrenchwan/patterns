-- returns a new random string, one per call
create or replace function susers.generate_random_string() returns text language plpgsql as $$
declare
	l_result text;
begin
	select md5(random()::text) || md5(random()::text) || md5(random()::text) into l_result;
	return l_result;
end; $$;

alter function susers.generate_random_string owner to upa;

-- insert api user (not user, but api users)
create or replace procedure susers.insert_api_user(
    p_login text, 
    p_pass text,
    p_roles int[]
) language plpgsql as $$
declare 
	l_salt text; -- user specific salt
	l_text_hash text; -- value for password with salt 
begin
    -- exit if apiuser already exists 
	if exists (select 1 from susers.api_users where apiuser_login = p_login) then
		raise exception 'user already exists';
		return;
	end if;
    -- generate random salt 
	select susers.generate_random_string() into l_salt;
	select encode(sha256((p_pass || l_salt)::bytea), 'base64') into l_text_hash;
    -- insert value
	insert into susers.api_users(apiuser_active, apiuser_login, apiuser_salt, apiuser_secret, apiuser_hash, apiuser_authorizations)
	select true, p_login, l_salt, susers.generate_random_string(), l_text_hash, p_roles;
end; $$;

alter procedure susers.insert_api_user owner to upa;

-- test api user: login and password are provided, it returns true if auth is valid
create or replace function susers.test_api_user_password(p_login text, p_password text) returns bool 
language plpgsql as $$
declare 
	l_salt text;
	l_hash text;
	l_value_hash text;
begin
	select apiuser_hash, apiuser_salt into l_hash, l_salt 
	from susers.api_users 
	where apiuser_login = p_login
	and apiuser_active = true;

	if l_hash is null or length(l_hash) = 0 or length(p_password) = 0 then 
		return false;
	end if;

	select encode(sha256((p_password || l_salt)::bytea), 'base64') into l_value_hash;
	return l_value_hash is not distinct from l_hash;
end; $$;

alter function susers.test_api_user_password owner to upa;

-- susers.find_secret_for_api_user loads secret for a given active user, raises exception otherwise
create or replace function susers.find_secret_for_api_user(p_login text) returns text 
language plpgsql
as $$
declare
	l_found bool = false;
	l_secret text;
begin 

	select true, apiuser_secret into l_found, l_secret
	from susers.api_users
	where apiuser_login = p_login
	and apiuser_active = true;
	
	if l_found is null or not l_found then 
		raise exception 'no active user found for login %', p_login;
	end if;

	return l_secret;
end;$$;

alter function susers.find_secret_for_api_user owner to upa;