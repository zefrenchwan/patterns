-- to execute in patterns database (pdb)
-- In said database, schema sgraphs exists. 

drop table if exists sgraphs.element_trait;
drop table if exists sgraphs.relation_role;
drop table if exists sgraphs.entity_attributes;
drop table if exists sgraphs.elements;
drop table if exists sgraphs.traits;
drop table if exists sgraphs.reftypes;
drop table if exists sgraphs.periods;

-- reftypes just defines if the value is an entity, a relation, or may be both
create table sgraphs.reftypes (
	reftype_id int primary key,
	reftype_description text
);

alter table sgraphs.reftypes owner to upa;

insert into sgraphs.reftypes(reftype_id, reftype_description) values(1,'entity only');
insert into sgraphs.reftypes(reftype_id, reftype_description) values(2,'relation only');
insert into sgraphs.reftypes(reftype_id, reftype_description) values(10, 'mixed');

-- sgraphs.traits defines all possible traits
create table sgraphs.traits (
	trait_id bigserial primary key, 
	trait_type int references sgraphs.reftypes(reftype_id),
	trait text not null,
	last_update timestamp without time zone default now()::timestamp without time zone
);

alter table sgraphs.traits owner to upa;

-- sgraphs.periods may be an activity or a period for an attribute. 
-- Intervals are stored as an ordered array of serialized time intervals
create table sgraphs.periods (
	period_id bigserial primary key, 
	period_empty bool,
	period_full bool,
	-- to optimize and not load the full table to find matches
	period_min timestamp without time zone,
	-- to optimize and not load the full table to find matches
	period_max timestamp without time zone,
	period_value text
);

alter table sgraphs.periods owner to upa;

-----------------------------------------------------
-----------------------------------------------------
-----------------------------------------------------
-----------------------------------------------------
-----------------------------------------------------
-----------------------------------------------------


-- sgraphs.elements store common part for relation and entity
create table sgraphs.elements (
	element_id text primary key,
	element_type int not null references sgraphs.reftypes(reftype_id),
	element_period bigint references sgraphs.periods(period_id),
	last_update timestamp without time zone default now()::timestamp without time zone
);

alter table sgraphs.elements owner to upa;

-- sgraphs.element_trait links an element and a trait 
create table sgraphs.element_trait (
	element_id text references sgraphs.elements(element_id) on delete cascade,
	trait_id bigint references sgraphs.traits(trait_id),
	last_update timestamp without time zone default now()::timestamp without time zone
);

alter table sgraphs.element_trait owner to upa;

-- given an entity, an entry for an attribute AND its value
create table sgraphs.entity_attributes (
	attribute_id bigserial primary key,
	entity_id text references sgraphs.elements(element_id) on delete cascade,
	attribute_name text not null, 
	attribute_value text not null, 
	period_id bigint references sgraphs.periods(period_id) on delete cascade,
	last_update timestamp without time zone default now()::timestamp without time zone
);

alter table sgraphs.entity_attributes owner to upa;

-- given a relation, a line in this table defines all the elements with a given role
create table sgraphs.relation_role (
	relation_role_id bigserial primary key, 
	relation_id text references sgraphs.elements(element_id) on delete cascade,
	role_in_relation text not null, 
	role_values text[],
	last_update timestamp without time zone default now()::timestamp without time zone
);

alter table sgraphs.relation_role owner to upa;

