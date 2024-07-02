drop schema sgraphs cascade; 
create schema sgraphs;
alter schema sgraphs owner to upa;

-------------------------------
-- GRAPHS GENERAL DEFINITION --
-------------------------------

-- sgraphs.graphs are graphs that contain versions of nodes
create table sgraphs.graphs (
	graph_id text primary key,
	graph_name text not null, 
	graph_description text
);

alter table sgraphs.graphs owner to upa;

-- sgrapghs.graph_entry contains a meta data entry for that graph (such as observer, source, etc)
create table sgraphs.graph_entries (
	entry_id bigserial primary key,
	graph_id text references sgraphs.graphs(graph_id) on delete cascade,	
	entry_key text not null, 
	entry_values text[]
);

alter table sgraphs.graph_entries owner to upa;

------------------------------------------------------
-- ELEMENTS DEFINITION: TRAITS, ENTITIES, RELATIONS --
------------------------------------------------------

-- reftypes just defines if the value is an entity, a relation, or may be both
create table sgraphs.reftypes (
	reftype_id int primary key,
	reftype_description text
);

alter table sgraphs.reftypes owner to upa;

insert into sgraphs.reftypes(reftype_id, reftype_description) values(1,'entity only');
insert into sgraphs.reftypes(reftype_id, reftype_description) values(2,'relation only');
insert into sgraphs.reftypes(reftype_id, reftype_description) values(10, 'mixed');

-- sgraphs.periods may be an activity or a period for an attribute. 
-- Intervals are stored as an ordered array of serialized time intervals
create table sgraphs.periods (
	period_id bigserial primary key, 
	-- to optimize and not load the full table to find matches
	period_min timestamp without time zone,
	-- to optimize and not load the full table to find matches
	period_max timestamp without time zone,
	-- ];[ for empty, ]-oo;+oo[ for full, intervals joined with U otherwise
	period_value text
);

alter table sgraphs.periods owner to upa;

-- sgraphs.traits defines all possible traits
create table sgraphs.traits (
	trait_id text not null primary key,
	graph_id text not null references sgraphs.graphs(graph_id) on delete cascade,
	trait_type int references sgraphs.reftypes(reftype_id),
	trait text not null
);

alter table sgraphs.traits owner to upa;

-- sgraphs.elements store common part for relation and entity
create table sgraphs.elements (
	element_id text primary key,
	graph_id text not null references sgraphs.graphs(graph_id) on delete cascade,
	element_type int not null references sgraphs.reftypes(reftype_id),
	element_period bigint references sgraphs.periods(period_id) on delete cascade
);

alter table sgraphs.elements owner to upa;

-- sgraphs.element_trait links an element and a trait. 
create table sgraphs.element_trait (
	element_id text references sgraphs.elements(element_id) on delete cascade,
	trait_id text references sgraphs.traits(trait_id)  on delete cascade
);

alter table sgraphs.element_trait owner to upa;

-- given an entity, an entry for an attribute AND its value
create table sgraphs.entity_attributes (
	attribute_id bigserial primary key,
	entity_id text references sgraphs.elements(element_id) on delete cascade,
	attribute_name text not null, 
	attribute_value text not null, 
	period_id bigint references sgraphs.periods(period_id) on delete cascade
);

alter table sgraphs.entity_attributes owner to upa;

-- given a relation, a line in this table defines all the elements with a given role
create table sgraphs.relation_role (
	relation_role_id bigserial primary key, 
	relation_id text references sgraphs.elements(element_id) on delete cascade,
	role_in_relation text not null
);

alter table sgraphs.relation_role owner to upa;

create table sgraphs.relation_role_values (
	relation_role_id bigint not null references sgraphs.relation_role(relation_role_id) on delete cascade,
	relation_value text not null references sgraphs.elements(element_id) on delete cascade,
	relation_period_id bigint references sgraphs.periods(period_id) on delete cascade
); 

alter table sgraphs.relation_role_values owner to upa;

-------------------------------------
-- INCLUSIONS AND NODES DEFINITION --
-------------------------------------

-- sgraphs.inclusions represent included graphs 
create table sgraphs.inclusions (
	source_id text references sgraphs.graphs(graph_id) on delete cascade,
	child_id text references sgraphs.graphs(graph_id) on delete cascade
);

alter table sgraphs.inclusions owner to upa;

-- sgraphs.nodes define same nodes from a graph to another
create table sgraphs.nodes (
	source_element_id text not null references sgraphs.elements(element_id) on delete cascade,
	child_element_id text not null references sgraphs.elements(element_id) on delete cascade
);

alter table sgraphs.nodes owner to upa;


-------------------------------
-- AND FINALLY, GRANT ACCESS --
-------------------------------

grant all privileges on all tables in schema sgraphs to upa;
