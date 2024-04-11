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


