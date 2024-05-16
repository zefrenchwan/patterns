drop schema if exists sknowledge cascade; 
create schema sknowledge;
alter schema sknowledge owner to upa;

-- sknowledge.topics define topics, that is knowledge about something
create table sknowledge.topics (
    topic_id bigserial primary key, 
    topic_name text unique not null, 
    topic_description text
);

-- sknowledge.traits define traits to qualify relations or entities
create table sknowledge.traits ( 
    trait_id bigserial primary key, 
    topic_id bigint references sknowledge.topics(topic_id),
    trait_name text not null, 
    trait_description text ,
    UNIQUE(trait_id, topic_id)
);

-- sknowledge.traits_inheritance define inheritance trees for traits
create table sknowledge.traits_inheritance (
    child_id bigint references sknowledge.traits(trait_id),
    parent_id bigint references sknowledge.traits(trait_id)
);

-- sknowledge.topics_inheritance define inheritance on topics (to include already defined notions)
create table sknowledge.topics_inheritance (
    child_id bigint references sknowledge.traits(trait_id),
    parent_id bigint references sknowledge.traits(trait_id)
);