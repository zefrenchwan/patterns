-- returns a new random string, one per call
create or replace function susers.generate_random_string() returns text language plpgsql as $$
declare
	l_result text;
begin
	select md5(random()::text) || md5(random()::text) || md5(random()::text) into l_result;
	return l_result;
end; $$;

alter function susers.generate_random_string owner to upa;
