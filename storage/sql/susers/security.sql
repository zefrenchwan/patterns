drop schema if exists susers cascade; 
create schema susers;
alter schema susers owner to upa;


-- defines roles in general, for users and api users
create table susers.roles (
	role_id serial primary key, 
	role_name text unique not null, 
	role_description text not null
);

alter table susers.roles owner to upa;

-- insert key roles 
insert into susers.roles(role_name, role_description) 
values ('manager', 'creates or deletes resources');
insert into susers.roles(role_name, role_description) 
values ('modifier', 'modifies resources');
insert into susers.roles(role_name, role_description) 
values ('observer', 'sees resources');
insert into susers.roles(role_name, role_description) 
values ('granter', 'allows other users to have same authorizations on objects');


-- susers.classes define objects that embed security to check
create table susers.classes (
    class_id serial primary key,
    class_name text not null unique, 
    class_description text not null
);

alter table susers.classes owner to upa;

insert into susers.classes(class_name, class_description) values('user', 'users of the api');
insert into susers.classes(class_name, class_description) values('graph', 'graph defines links between entities');

-- susers.resources define concrete resources. 
-- Each time a graph or an user is inserted or deleted, it is changed in here too. 
-- It is not magnificent, but a graph or an user is inserted or deleted in here way less often an access is checked, so...
-- We maintain this table to speed up instead of a view (generated for each access) or a resources table in graphs (not model linked). 
-- Line is deleted once underlying resource is deleted. 
-- But if an user is deleted, resource should stay here. 
create table susers.resources (
    resource_id text primary key,
    resource_type int not null references susers.classes(class_id),
    resource_creation_date timestamp with time zone default now(), 
    resource_creator_login text not null
);

alter table susers.resources owner to upa;

-- susers.users define api users
create table susers.users (
    user_id text primary key, 
    user_active bool not null default true, 
    user_login text unique, 
    user_salt text not null, 
    user_secret text not null,
    user_hash text not null
);

alter table susers.users owner to upa;

-- susers.authorizations define concrete access rights to ressources. 
-- null in resource means that ALL resources are concerned. 
create table susers.authorizations (
    auth_id bigserial primary key,
    auth_active bool default true,
    auth_user_id text not null references susers.users(user_id),
    auth_role_id int not null references susers.roles(role_id),
    auth_class_id int not null references susers.classes(class_id),
    auth_resource text references susers.resources(resource_id)
);

alter table susers.authorizations owner to upa;

-- susers.all_authorizations list all authorizations for all users
create or replace view susers.all_authorizations(
	user_id, user_login, user_active, role_name, class_name,
	all_resources, authorized_resources, unauthorized_resources
) as 
	with all_auth as (
		select USR.user_id, USR.user_login, USR.user_active,
		ROL.role_name, CLA.class_name, 
		AUT.auth_active as active_resource,
		case AUT.auth_resource when null then 0 else 1 end as flag_all_resources, 
		AUT.auth_resource as resource
		from susers.authorizations AUT
		join susers.roles ROL on ROL.role_id = AUT.auth_role_id
		join susers.classes CLA on CLA.class_id = AUT.auth_class_id
		join susers.users USR on USR.user_id = AUT.auth_user_id
	), mapped_auth as (
		select AAU.user_id, AAU.user_login, AAU.user_active,
		AAU.role_name, AAU.class_name, 
		AAU.flag_all_resources, 
		AAU.resource as original_value,
		case AAU.active_resource when true then AAU.resource else null end as auth_resource,
		case AAU.active_resource when false then AAU.resource else null end as unauth_resource
		from all_auth AAU
	), agg_non_null_auth as (
		select MAA.user_id, MAA.user_login, MAA.user_active,
		MAA.role_name, MAA.class_name, 
		array_agg(auth_resource) as authorized_resources,
		array_agg(unauth_resource) as unauthorized_resources
		from mapped_auth MAA
		where MAA.original_value is not null
		group by MAA.user_id, 
		MAA.user_login, MAA.user_active,
		MAA.role_name, MAA.class_name 
	), full_auth as (
		select ANA.user_id, ANA.user_login, ANA.user_active, ANA.class_name,ANA.role_name, 
		false as all_resources, ANA.authorized_resources, ANA.unauthorized_resources
		from agg_non_null_auth ANA
		union 
		select MAA.user_id, MAA.user_login, MAA.user_active,
		MAA.class_name, MAA.role_name, true as all_resources, 
		null as authorized_resources, 
		null as unauthorized_resources 
		from mapped_auth MAA
		where MAA.original_value is null
	)
	select * from full_auth FAU
	order by FAU.user_id, FAU.user_login, 
	FAU.user_active,
	FAU.class_name, FAU.role_name;

alter view susers.all_authorizations owner to upa;


grant all privileges on all tables in schema susers to upa;
