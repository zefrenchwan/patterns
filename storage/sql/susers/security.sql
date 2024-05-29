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
values ('creator', 'creates objects');
insert into susers.roles(role_name, role_description) 
values ('modifier', 'modifies objects');
insert into susers.roles(role_name, role_description) 
values ('observer', 'sees objects');
insert into susers.roles(role_name, role_description) 
values ('granter', 'allows other users to have same authorizations on objects');
insert into susers.roles(role_name, role_description) 
values ('supervisor', 'allows to manage users');


-- susers.classes define objects that embed security to check
create table susers.classes (
    class_id serial primary key,
    class_name text not null unique, 
    class_description text not null
);

alter table susers.classes owner to upa;

insert into susers.classes(class_name, class_description) values('graph', 'graph defines links between entities');

-- susers.resources define concrete resources. 
-- Each resource has a class to remember where to find the matching id
create table susers.resources (
    resource_id text primary key,
    resource_type int not null references susers.classes(class_id),
    resource_creation_date timestamp with time zone default now() 
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

-- susers.authorizations define concrete access rights to ressources
create table susers.authorizations (
    auth_id bigserial primary key,
    auth_active bool default true,
    auth_class_override int references susers.classes(class_id),
    auth_resource text references susers.resources(resource_id)
);

alter table susers.authorizations owner to upa;



grant all privileges on all tables in schema susers to upa;
