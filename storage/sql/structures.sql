-- to execute in patterns database (pdb)
-- In said database, schema spat exists. 
drop table if exists spat.pattern_links;
drop table if exists spat.entity_trait;
drop table if exists spat.relation_trait;
drop table if exists spat.relation_role;
drop table if exists spat.relations;
drop table if exists spat.entity_attributes;
drop table if exists spat.entities;
drop table if exists spat.mdlinks;
drop table if exists spat.mdroles;
drop table if exists spat.traits;
drop table if exists spat.reftypes;
drop table if exists spat.patterns;
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

create table spat.patterns (
    pattern_id bigserial primary key, 
    pattern_name text unique not null
);

-- links a pattern to one of its parents
create table spat.pattern_links (
    subpattern_id bigint references spat.patterns(pattern_id),
    superpattern_id bigint references spat.patterns(pattern_id)
);

-- in a pattern's dictionary, a mdlink defines an entry as the trait / supertrait, 
-- both for relation and entities (depending on the reftype). 
create table spat.mdlinks (
	pattern_id bigint references spat.patterns(pattern_id),
	reftype int references spat.reftypes(reftype_id),
	subtrait_id bigint references spat.traits(trait_id),
	supertrait_id bigint references spat.traits(trait_id)
);

-- in a pattern's dictionary, for a relation metadata, an entry defines all possible traits given a role
create table spat.mdroles (
	pattern_id bigint references spat.patterns(pattern_id),
	relation_trait_id bigint references spat.traits(trait_id),
	role_in_relation text not null,
	trait_id bigint references spat.traits(trait_id)
);


-- general table for entities, to be linked to its traits and attributes
create table spat.entities (
	entity_id bigserial primary key, 
	entity_period bigint references spat.periods(period_id)
);

-- links an entity to its traits
create table spat.entity_trait (
	entity_id bigint references spat.entities(entity_id),
	trait_id bigint references spat.traits(trait_id)
);

-- given an entity, an entry for an attribute AND its value
create table spat.entity_attributes (
	attribute_id bigserial primary key,
	entity_id bigint references spat.entities(entity_id),
	attribute_name text not null, 
	attribute_value text not null, 
	period_id bigint references spat.periods(period_id)
);

-- defines a relation
create table spat.relations (
	relation_id bigserial primary key, 
    -- activity defines when the relation is true
	relation_activity bigint references spat.periods(period_id)
);

-- links a relation to its traits
create table spat.relation_trait (
	relation_id bigint references spat.relations(relation_id), 
	trait_id bigint references spat.traits(trait_id)
);

-- given a relation, a line in this table defines all the elements with a given role
create table spat.relation_role (
	relation_role_id bigserial primary key, 
	relation_id bigint references spat.relations(relation_id),
	role_in_relation text not null, 
	role_values text[]
);