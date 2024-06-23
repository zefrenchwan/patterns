-- susers.upsert_user changes password of an existing user, or creates user if possible. 
create or replace procedure susers.upsert_user(
	p_creator in text, p_login in text, p_new_password in text) 
language plpgsql as $$
declare 
	-- id of the user, if any, to upsert password for
	l_user text;
	-- salt of the user
	l_salt text; 
	-- hashed password
	l_text_hash text;
begin 

	-- find other user to act on
	select user_id into l_user 
	from susers.users 
	where user_login = p_login;

	if l_user is null and p_login <> p_creator then 
	    call susers.accept_user_access_to_resource_or_raise(p_creator, 'user', ARRAY['manager'], true, null); 
	else 
		-- an existing one, may be ourself when changing a password. Modifier is enough
	    call susers.accept_user_access_to_resource_or_raise(p_creator, 'user', ARRAY['modifier'], true, l_user); 
	end if;

	-- then, insert or update
	select susers.generate_random_string() into l_salt;
	
	select encode(sha256((p_new_password || l_salt)::bytea), 'base64') into l_text_hash;
    
	if l_user is null then
		select gen_random_uuid() into l_user;
		insert into susers.users(user_id, user_active, user_login, user_salt, user_secret, user_hash)
		select l_user, true, p_login, l_salt, susers.generate_random_string(), l_text_hash;
		call susers.insert_new_resource(p_creator, 'user', l_user);
		-- insert user as an observer and modifier for said user 
		call susers.grant_access_to_user_for_resource(p_login, 'user', 'observer', l_user);
		call susers.grant_access_to_user_for_resource(p_login, 'user', 'modifier', l_user);
	else 
		update susers.users 
		set user_salt = l_salt, 
		user_secret = susers.generate_random_string(), 
		user_hash = l_text_hash
		where user_id = l_user;
	end if;	
end; $$;

alter procedure susers.upsert_user owner to upa;

-- susers.list_user_data_and_supervised_user_data returns user data and visible user data
create or replace function susers.list_user_data_and_supervised_user_data(p_login text)
returns table (
		user_id text, user_login text, user_active bool, 
		role_name text, class_name text, all_resources bool, 
		authorized_resources text[], unauthorized_resources text[]
	)
language plpgsql as $$
declare
begin 
	return query 
	with users_for_all_auth as (
		select distinct
		ALR.user_id, ALR.user_login, ALR.user_active, 
		ALR.role_name, ALR.class_name, ALR.all_resources, 
		ALR.authorized_resources, ALR.unauthorized_resources
		from susers.all_authorizations ALA 
		join susers.all_authorizations ALR on ALR.user_id <> ALA.user_id 
		where ALA.user_active = true 
		and ALA.all_resources = true 
		and ALA.role_name = 'user'
		and ALA.user_login = p_login
		and ALA.class_name IN ('observer', 'modifier')
	), user_self as (
		select distinct
		ALA.user_id, ALA.user_login, ALA.user_active, 
		ALA.role_name, ALA.class_name, ALA.all_resources, 
		ALA.authorized_resources, 
		-- user should not see what user may not access
		null::text[] as unauthorized_resources
		from susers.all_authorizations ALA
		where ALA.user_active = true 
		and ALA.user_login = p_login
	), specific_visible_users as (
		select unnest(ALA.authorized_resources) as user_id 
		from susers.all_authorizations ALA
		where ALA.user_active = true 
		and ALA.role_name = 'user'
		and ALA.user_login = p_login
		and ALA.class_name in ('modifier', 'observer')
	), users_specific as (
		select distinct
		ALR.user_id, ALR.user_login, ALR.user_active, 
		ALR.role_name, ALR.class_name, ALR.all_resources, 
		ALR.authorized_resources, ALR.unauthorized_resources
		from susers.all_authorizations ALR
		join specific_visible_users SVU on ALR.user_id = SVU.user_id
	), users_union as (
		select *
		from users_for_all_auth
		UNION
		select *
		from user_self
		UNION  
		select *
		from users_specific 	
	) select distinct * 
	from users_union
	order by user_login, role_name, class_name;
end;$$;

alter function susers.list_user_data_and_supervised_user_data owner to upa;


-- susers.lock_user inactivates any access to user by id 
create or replace procedure susers.lock_user(p_actor text, p_user_id text)
language plpgsql as $$
declare 
begin
	if not exists (select 1 from susers.users where user_id = p_user_id) then 
		raise exception 'no user %', p_user_id using errcode = '22023';
	end if;

	-- locking means modifying the user
	call susers.accept_user_access_to_resource_or_raise(p_actor, 'user', ARRAY['modifier'], true, p_user_id);
    -- then, unlock
	update susers.users set user_active = false where user_id = p_user_id;
end;$$;

-- susers.unlock_user (re)activates any access to user by id
create or replace procedure susers.unlock_user(p_actor text, p_user_id text)
language plpgsql as $$
declare 
begin
	if not exists (select 1	from susers.users where user_id = p_user_id) then 
		call susers.accept_user_access_to_resource_or_raise(p_actor, 'user', ARRAY['modifier'], true, p_user_id);
	else 
		raise exception 'no user %', p_user_id using errcode = '22023';
	end if;
	
	update susers.users set user_active = true where user_id = p_user_id;
end;$$;

-- susers.delete_user deletes any information about this user by id
create or replace procedure susers.delete_user(p_actor text, p_user_id text)
language plpgsql as $$
declare 
begin
	if not exists (select 1 from susers.users where user_id = p_user_id) then 
		raise exception 'no user %', p_user_id using errcode = '22023';
	end if;
	
	-- only a manager may delete a resource 
	call susers.accept_user_access_to_resource_or_raise(p_actor, 'user', ARRAY['manager'], true, p_user_id);

	-- to avoid foreign key issues
	update susers.authorizations 
	set auth_resource = null, auth_active = false 
	where auth_user_id = p_user_id;

	delete from susers.authorizations where auth_user_id = p_user_id;
	delete from susers.users where user_id = p_user_id;
	call susers.delete_resource('user', p_user_id);
end;$$;


-- susers.grant_all_role_auth_to grant full role auth for a given class
create or replace procedure susers.grant_all_role_auth_to(p_granter text, p_login text, p_role text, p_class text) 
language plpgsql as $$
declare
	l_user_id text;
begin
	-- find user to grant access to 
	select user_id into l_user_id 
	from susers.users
	where user_login = p_login;

	if l_user_id is null then 
		raise exception 'no user %', p_login using errcode = '22023';
	end if;

	call susers.accept_user_access_to_resource_or_raise(p_granter, 'user', ARRAY['granter'], true, l_user_id);
	call susers.accept_user_access_to_resource_or_raise(p_granter, p_class, ARRAY[p_role], true, null);
	call susers.grant_access_to_user_for_resource(p_login, p_class, p_role, null);
end;$$;
