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
	    call susers.accept_any_user_access_to_resource_or_raise(p_creator, 'user', ARRAY['manager'], null); 
	else 
		-- an existing one, may be ourself when changing a password. Modifier is enough
	    call susers.accept_any_user_access_to_resource_or_raise(p_creator, 'user', ARRAY['modifier'], l_user); 
	end if;

	-- then, insert or update
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

-- susers.lock_user inactivates any access to user by login 
create or replace procedure susers.lock_user(p_actor text, p_login text)
language plpgsql as $$
declare 
	l_user_id text;
begin
	select user_id into l_user_id 
	from susers.users 
	where user_login = p_login;

	if l_user_id is null then 
		return;
	end if;

	-- locking means modifying the user
	call susers.accept_any_user_access_to_resource_or_raise(p_actor, 'user', ARRAY['modifier'], l_user_id);
    
	update susers.users set user_active = false
	where user_id = l_user_id;
end;$$;

-- susers.unlock_user (re)activates any access to user by login 
create or replace procedure susers.unlock_user(p_actor text, p_login text)
language plpgsql as $$
declare 
	l_user_id text;
begin
	select user_id into l_user_id 
	from susers.users 
	where user_login = p_login;

	if l_user_id is not null then 
		call susers.accept_any_user_access_to_resource_or_raise(p_actor, 'user', ARRAY['modifier'], l_user_id);
	else 
		-- raising would allow actor to know someone is an user, so ... 
		return;
	end if;
	
	update susers.users set user_active = true
	where user_id = l_user_id;
end;$$;

-- susers.delete_user deletes any information about this user by login 
create or replace procedure susers.delete_user(p_actor text, p_login text)
language plpgsql as $$
declare 
	l_user_id text;
begin
	select user_id into l_user_id 
	from susers.users 
	where user_login = p_login;

	if l_user_id is null then 
		return;
	end if;
	
	-- only a manager may delete a resource 
	call susers.accept_any_user_access_to_resource_or_raise(p_actor, 'user', ARRAY['manager'], l_user_id);
    

	
	update susers.authorizations 
	set auth_resource = null 
	where auth_user_id = l_user_id;

	delete from susers.authorizations 
	where auth_user_id = l_user_id;

	delete from susers.users 
	where user_id = l_user_id;

end;$$;
