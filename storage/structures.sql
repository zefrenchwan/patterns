-- to execute in patterns database (pdb)
-- In said database, schema spat exists. 
drop table if exists spat.entity_trait;
drop table if exists spat.relation_trait;
drop table if exists spat.relation_role;
drop table if exists spat.relations; 
drop table if exists spat.entities;
drop table if exists spat.mdlinks;
drop table if exists spat.mdroles;
drop table if exists spat.traits;
drop table if exists spat.reftypes;
drop table if exists spat.dictionaries;
drop table if exists spat.periods;

-- reftypes just defines if the value is an entity, a relation, or may be both
create table spat.reftypes (
	reftype_id int primary key,
	reftype_description text
);

insert into spat.reftypes(reftype_id, reftype_description) values(1,'entity only');
insert into spat.reftypes(reftype_id, reftype_description) values(2,'relation only');
insert into spat.reftypes(reftype_id, reftype_description) values(10, 'mixed');

-- spat.traits defines all possible traits
create table spat.traits (
	trait_id bigserial primary key, 
	trait_type int references spat.reftypes(reftype_id),
	trait text not null
);

-- spat.periods may be an activity or a period for an attribute. 
-- Intervals are stored as an ordered array of serialized time intervals
create table spat.periods (
	period_id bigserial primary key, 
	period_empty bool,
	period_full bool,
	period_value text[]
);

create table spat.dictionaries (
	dictionary_id bigserial primary key
);

create table spat.mdlinks (
	dictionary_id bigint references spat.dictionaries(dictionary_id),
	reftype int references spat.reftypes(reftype_id),
	subtrait_id bigint references spat.traits(trait_id),
	supertrait_id bigint references spat.traits(trait_id)
);

create table spat.mdroles (
	dictionary_id bigint references spat.dictionaries(dictionary_id),
	relation_trait_id bigint references spat.traits(trait_id),
	role_in_relation text not null,
	trait_id bigint references spat.traits(trait_id)
);


create table spat.entities (
	entity_id bigserial primary key, 
	entity_period bigint references spat.periods(period_id)
);

create table spat.entity_trait (
	entity_id bigint references spat.entities(entity_id),
	trait_id bigint references spat.traits(trait_id)
);

create table spat.relations (
	relation_id bigserial primary key, 
	realtion_activity bigint references spat.periods(period_id)
);

create table spat.relation_trait (
	relation_id bigint references spat.relations(relation_id), 
	trait_id bigint references spat.traits(trait_id)
);

create table spat.relation_role (
	relation_role_id bigserial primary key, 
	relation_id bigint references spat.relations(relation_id),
	role_in_relation text not null, 
	role_values text[]
);