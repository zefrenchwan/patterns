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
    resource_type int not null references susers.classes(class_id) on delete cascade,
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
-- When auth_inclusion is true, then it is granted, otherwise revoked. 
-- When auth_all_resources is true, then access policy is to exclude other resources. 
-- For instance, when auth_inclusion and auth_all_resources, then excluded elements for same role and classes are the only refused resources. 
create table susers.authorizations (
    auth_id bigserial primary key,
	auth_all_resources bool not null default false,
    auth_inclusion bool not null default true,
    auth_user_id text not null references susers.users(user_id) on delete cascade,
    auth_role_id int not null references susers.roles(role_id) on delete cascade,
    auth_class_id int not null references susers.classes(class_id) on delete cascade
);

alter table susers.authorizations owner to upa;


-- if not all resources are accessible, then detail authorized resources
create table susers.resources_authorizations (
	auth_id bigint not null references susers.authorizations (auth_id) on delete cascade, 
	resource text references susers.resources(resource_id) on delete cascade
);

alter table susers.resources_authorizations owner to upa;

-- susers.all_users_authorizations agregates data from all auth tables
-- to make an usable data. 
create or replace view susers.all_users_authorizations 
(user_id , user_login , user_active ,
class_name , role_name , 
auth_all_resources , auth_inclusion , resource 
) as 
select 
USR.user_id, 
USR.user_login,
USR.user_active,
CLA.class_name,
ROL.role_name,
AUT.auth_all_resources, 
AUT.auth_inclusion, 
RAU.resource
from susers.authorizations AUT 
join susers.users USR on USR.user_id = AUT.auth_user_id
join susers.roles ROL on AUT.auth_role_id = ROL.role_id
join susers.classes CLA on CLA.class_id = AUT.auth_class_id
left outer join susers.resources_authorizations RAU on RAU.auth_id = AUT.auth_id;

alter view susers.all_users_authorizations owner to upa;



grant all privileges on all tables in schema susers to upa;
