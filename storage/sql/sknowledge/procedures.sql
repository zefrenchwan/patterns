-- sknowledge.upsert_topic upsers a topic
create or replace procedure sknowledge.upsert_topic(
    p_topic text, p_description text
) language plpgsql as $$
declare 
begin 
    if exists (select 1 from sknowledge.topics where topic_name = p_topic) then 
        update sknowledge.topics 
        set topic_description = p_description 
        where topic_name = p_topic;
    else
        insert into sknowledge.topics(topic_name, topic_description) 
        values (p_topic, p_description);
    end if;
end; $$;

-- sknowledge.upsert_trait upserts a trait, 
-- and if p_topic is not null, 
-- upserts the topic before adding the link 
create or replace procedure sknowledge.upsert_trait(
    p_trait text, p_description text, p_topic text
) language plpgsql as $$
declare 
    l_topic_id bigint;
    l_trait_id bigint;
begin 
    -- p_topic not null means upserting it
    if p_topic is not null then 
        select topic_id into l_topic_id
        from sknowledge.topics 
        where topic_name = p_topic;

        if l_topic_id is null then 
            insert into sknowledge.topics(topic_name) values (p_topic)
            returning topic_id into l_topic_id;
        end if;
    end if;

    -- then, insert trait if not already here 
    select trait_id into l_trait_id
    from sknowledge.traits 
    where trait_name = p_trait
    and topic_id = l_topic_id;

    if l_trait_id is null then
        insert into sknowledge.traits(trait_name, trait_description, topic_id)
        values (p_trait, p_description, l_topic_id);
    else
        update sknowledge.traits 
        set trait_description = p_description
        where topic_id = l_topic_id and trait_name = p_trait;
    end if;
end; $$;

-- sknowledge.upsert_link_topics create topics if necessary, 
-- and adds the link from child to parent
create or replace procedure sknowledge.upsert_link_topics(p_child text, p_parent text) 
language plpgsql as $$
declare 
    l_child_topic_id bigint;
    l_parent_topic_id bigint; 
begin 
    select topic_id into l_child_topic_id 
    from sknowledge.topics 
    where topic_name = p_child; 

    select topic_id into l_parent_topic_id 
    from sknowledge.topics 
    where topic_name = p_parent; 

    if l_parent_topic_id is null then 
        insert into sknowledge.topics(topic_name) values (p_parent)
        returning topic_id into l_parent_topic_id;
    end if;
    if l_child_topic_id is null then 
        insert into sknowledge.topics(topic_name) values (p_child)
        returning topic_id into l_child_topic_id;
    end if;

    insert into sknowledge.topics_inheritance (child_id,parent_id) 
    values (l_child_topic_id, l_parent_topic_id);
end; $$;