drop table if exists susers.api_users;
drop table if exists susers.roles;

-- defines roles in general, for users and api users
create table susers.roles (
	role_id serial primary key, 
	role_name text unique not null, 
	role_description text not null
);

alter table susers.roles owner to upa;

-- insert key roles 
insert into susers.roles(role_name, role_description) 
values ('creator', 'creates graphs and modifies them all');
insert into susers.roles(role_name, role_description) 
values ('modifier', 'modifies and see graphs, cannot create');
insert into susers.roles(role_name, role_description) 
values ('contributor', 'shares and sees a graph, cannot modify');
insert into susers.roles(role_name, role_description) 
values ('observer', 'sees a graph, cannot modify it, cannot share');


-- susers.api_users define api users
create table susers.api_users (
    apiuser_id serial primary key, 
    apiuser_active bool not null default true, 
    apiuser_login text unique, 
    apiuser_salt text not null, 
    apiuser_secret text not null,
    apiuser_hash text not null,
    apiuser_authorizations int[] not null
);

alter table susers.api_users owner to upa;