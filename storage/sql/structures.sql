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

alter table spat.reftypes owner to upa;

insert into spat.reftypes(reftype_id, reftype_description) values(1,'entity only');
insert into spat.reftypes(reftype_id, reftype_description) values(2,'relation only');
insert into spat.reftypes(reftype_id, reftype_description) values(10, 'mixed');

-- spat.traits defines all possible traits
create table spat.traits (
	trait_id bigserial primary key, 
	trait_type int references spat.reftypes(reftype_id),
	trait text not null
);

alter table spat.traits owner to upa;

-- spat.periods may be an activity or a period for an attribute. 
-- Intervals are stored as an ordered array of serialized time intervals
create table spat.periods (
	period_id bigserial primary key, 
	period_empty bool,
	period_full bool,
	period_value text[]
);

alter table spat.periods owner to upa;

create table spat.patterns (
    pattern_id bigserial primary key, 
    pattern_name text unique not null
);

alter table spat.patterns owner to upa;

-- links a pattern to one of its parents
create table spat.pattern_links (
    subpattern_id bigint references spat.patterns(pattern_id),
    superpattern_id bigint references spat.patterns(pattern_id)
);

alter table spat.pattern_links owner to upa;

-- in a pattern's dictionary, a mdlink defines an entry as the trait / supertrait, 
-- both for relation and entities (depending on the reftype). 
create table spat.mdlinks (
	pattern_id bigint references spat.patterns(pattern_id),
	reftype int references spat.reftypes(reftype_id),
	subtrait_id bigint references spat.traits(trait_id),
	supertrait_id bigint references spat.traits(trait_id)
);

alter table spat.mdlinks owner to upa;

-- in a pattern's dictionary, for a relation metadata, an entry defines all possible traits given a role
create table spat.mdroles (
	pattern_id bigint references spat.patterns(pattern_id),
	relation_trait_id bigint references spat.traits(trait_id),
	role_in_relation text not null,
	trait_id bigint references spat.traits(trait_id)
);

alter table spat.mdroles owner to upa;

-- general table for entities, to be linked to its traits and attributes
create table spat.entities (
	entity_id bigserial primary key, 
	entity_period bigint references spat.periods(period_id)
);

alter table spat.entities owner to upa;

-- links an entity to its traits
create table spat.entity_trait (
	entity_id bigint references spat.entities(entity_id),
	trait_id bigint references spat.traits(trait_id)
);

alter table spat.entity_trait owner to upa;

-- given an entity, an entry for an attribute AND its value
create table spat.entity_attributes (
	attribute_id bigserial primary key,
	entity_id bigint references spat.entities(entity_id),
	attribute_name text not null, 
	attribute_value text not null, 
	period_id bigint references spat.periods(period_id)
);

alter table spat.entity_attributes owner to upa;

-- defines a relation
create table spat.relations (
	relation_id bigserial primary key, 
    -- activity defines when the relation is true
	relation_activity bigint references spat.periods(period_id)
);

alter table spat.relations owner to upa;

-- links a relation to its traits
create table spat.relation_trait (
	relation_id bigint references spat.relations(relation_id), 
	trait_id bigint references spat.traits(trait_id)
);

alter table spat.relation_trait owner to upa;

-- given a relation, a line in this table defines all the elements with a given role
create table spat.relation_role (
	relation_role_id bigserial primary key, 
	relation_id bigint references spat.relations(relation_id),
	role_in_relation text not null, 
	role_values text[]
);

alter table spat.relation_role owner to upa;


--------------------------------
--------------------------------
--------------------------------


drop view spat.v_traitslinks;

create or replace view spat.v_traitslinks as 
with all_linked_elements as (
	select TPAT.pattern_name, TPL.reftype, TTSUB.trait as subtrait, 
	array_agg(TTSUP.trait order by TTSUP.trait)  as supertraits 
	from spat.patterns TPAT
	inner join spat.mdlinks TPL on TPL.pattern_id = TPAT.pattern_id 
	inner join spat.traits TTSUB on TTSUB.trait_id = TPL.subtrait_id
	inner join spat.traits TTSUP on TTSUP.trait_id = TPL.supertrait_id
	group by TPAT.pattern_name, TPL.reftype, TTSUB.trait
), all_unlinked_elements as (
	select  TPAT.pattern_name, TPL.reftype, TTSUP.trait as supertrait, array[]::text[] 
	from spat.patterns TPAT
	inner join spat.mdlinks TPL on TPL.pattern_id = TPAT.pattern_id 
	inner join spat.traits TTSUP on TTSUP.trait_id = TPL.supertrait_id
	where not exists (
		select 1 from spat.mdlinks MDL where MDL.subtrait_id = TTSUP.trait_id
	)
)
select * from all_linked_elements
UNION 
select * from all_unlinked_elements;

alter view spat.v_traitslinks owner to upa;