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


-- susers.users define api users
create table susers.users (
    user_id serial primary key, 
    user_active bool not null default true, 
    user_login text unique, 
    user_salt text not null, 
    user_secret text not null,
    user_hash text not null
);

alter table susers.users owner to upa;
