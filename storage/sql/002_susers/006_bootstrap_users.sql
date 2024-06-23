-- insert user (to boostrap the system). 
-- User is inserted as an user and a resource. 
-- NO access is provided, should be done separately. 
create or replace procedure susers.insert_user(
    p_login text, 
    p_pass text
) language plpgsql as $$
declare 
	l_user_id text; -- user id
	l_salt text; -- user specific salt
	l_text_hash text; -- value for password with salt 
	l_class_id int;
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
	select gen_random_uuid() into l_user_id;
	insert into susers.users(user_id, user_active, user_login, user_salt, user_secret, user_hash)
	select l_user_id, true, p_login, l_salt, susers.generate_random_string(), l_text_hash;
	-- insert user created with user
	select class_id into l_class_id from susers.classes where class_name = 'user';
	insert into susers.resources(resource_id, resource_type, resource_creator_login)
	select l_user_id, l_class_id, p_login;
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
		auth_user_id,
		auth_role_id,
		auth_class_id,
		auth_all_resources)
	select USR.user_id, 
	ROL.role_id, 
	CLA.class_id, 
	true
	from susers.users USR
	cross join susers.classes CLA 
	cross join susers.roles ROL
	where user_login = p_login;
end;$$;

alter procedure susers.insert_super_user_roles owner to upa;