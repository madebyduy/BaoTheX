-- BaoTheX inherited its entity taxonomy from RepWire, a strength-training
-- research aggregator. entity_kind therefore has 'researcher' and 'publication'
-- but no way to name a football club or a national team — which is why a sports
-- newspaper's entity table held seven rows, all of them gyms and journals.
--
-- These two values must land in their own migration: Postgres forbids using an
-- enum value in the same transaction that added it, and 0023 inserts hundreds
-- of rows that use them.
ALTER TYPE entity_kind ADD VALUE IF NOT EXISTS 'club';
ALTER TYPE entity_kind ADD VALUE IF NOT EXISTS 'national_team';
