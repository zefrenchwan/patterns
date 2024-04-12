-- spat.UpsertPattern:
-- adds a pattern named p_pattern and links it to its parents if it did not exist
-- sets parents no matter previous values if already inserted
-- Calling parents with array[]::text[] will just add this pattern, no parent. 
create or replace procedure spat.UpsertPattern(p_pattern text, p_parents text[]) 
language plpgsql
as $$
declare
	v_pattern text ;
	s_index bigint := -1 ; 
	v_index bigint := -1 ;
begin

select pattern_id into s_index
from spat.patterns
where pattern_name = p_pattern;

if s_index is null then 
	insert into spat.patterns(pattern_name) values (p_pattern) returning pattern_id into s_index;
end if;

delete from spat.pattern_links where subpattern_id = s_index;

foreach v_pattern slice 0 in array p_parents loop 	
	select pattern_id into v_index
	from spat.patterns
	where pattern_name = v_pattern;

	if v_index is null then 
		insert into spat.patterns(pattern_name) values (v_pattern) returning pattern_id into s_index;
	end if;
	
	if not exists (select 1 from spat.pattern_links where subpattern_id = s_index and superpattern_id = v_index) then 
		insert into spat.pattern_links(subpattern_id, superpattern_id) values (s_index, v_index);
	end if ; 
	
end loop; 

end; $$
;


-- spat.CreateTrait creates a trait if it does not exist. 
create or replace procedure spat.CreateTrait(p_trait text) language plpgsql as $$
declare
	l_traitid bigint;
begin

	select trait_id, trait_type, trait into l_traitid 
	from spat.traits 
	where trait = p_trait and trait_type in (1, 10);

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_trait);
	end if;
end; $$
;

-- spat.CreateRelationTrait creates a relation trait if it does not exist
create or replace procedure spat.CreateRelationTrait(p_trait text) language plpgsql as $$
declare
	l_traitid bigint;
begin

	select trait_id, trait_type, trait into l_traitid 
	from spat.traits 
	where trait = p_trait and trait_type in (2,10);

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_trait);
	end if;
end; $$
;

-- spat.LinkTraitsInPattern links a subtrait to a trait in a pattern. 
create or replace procedure spat.LinkTraitsInPattern(p_patternname text, p_subtrait text, p_trait text, p_reftype int) language plpgsql as $$
declare
	l_patternid bigint;
	l_traitid bigint;
	l_subtraitid bigint;
begin

	select trait_id  into l_traitid 
	from spat.traits 
	where trait = p_trait and trait_type in (p_reftype, 10) ;

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_trait)
		returning trait_id into l_traitid;
	end if;
	
	select trait_id  into l_subtraitid 
	from spat.traits 
	where trait = p_subtrait and trait_type in (p_reftype, 10);

	if l_subtraitid is null then 
		insert into spat.traits(trait_type, trait) values (p_reftype, p_subtrait)
		returning trait_id into l_subtraitid;
	end if;
	
	select pattern_id into l_patternid 
	from spat.patterns 
	where pattern_name = p_patternname;
	
	if l_patternid is null then 
		insert into spat.patterns(pattern_name) values (p_patternname)
		returning pattern_id into l_patternid;
	end if;
	
	if not exists (
		select 1 from spat.mdlinks
		where pattern_id = l_patternid and subtrait_id = l_subtraitid
		and supertrait_id = l_traitid and reftype in (10, p_reftype)
	) then 
		insert into spat.mdlinks values (l_patternid, p_reftype, l_subtraitid, l_traitid);
	end if ;
	
end; $$
;


-- spat.AddRoleToRelationInPattern sets a role with given EXISTING traits. 
-- pattern and relation may be created on the fly if necessary. 
create or replace procedure spat.AddRoleToRelationInPattern(p_patternname text, 
	p_relationtrait text, p_role text, p_roleTraits text[]) language plpgsql as $$
declare
	l_patternid bigint;
	l_traitid bigint;
	l_roletrait text;
	l_desttrait bigint;
begin
	select pattern_id into l_patternid
	from spat.patterns 
	where pattern_name = p_patternname;
	
	if l_patternid is null then 
		insert into spat.patterns(pattern_name) values (p_patternname);
	end if; 
	
	select trait_id into l_traitid 
	from spat.traits 
	where trait = p_relationtrait and trait_type in (2, 10);

	if l_traitid is null then 
		insert into spat.traits(trait_type, trait) values (2, p_relationtrait)
		returning trait_id into l_traitid;
	end if;
	
	delete from spat.mdroles where pattern_id = l_patternid 
	and relation_trait_id = l_traitid and role_in_relation = p_role;
	
	foreach l_roletrait slice 0 in array p_roleTraits loop 
		select trait_id into l_desttrait
		from spat.traits 
		where trait = l_roletrait;
		
		if l_desttrait is null then 
			raise exception 'non existing destination trait %', l_roletrait;
		end if ;
	
		-- insert for sure due to deletion
		insert into spat.mdroles values (l_patternid, l_traitid, p_role, l_desttrait);
		
	end loop;
	
end; $$
;
