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
