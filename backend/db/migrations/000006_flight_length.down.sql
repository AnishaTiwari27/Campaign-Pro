ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_flight_days_check;
ALTER TABLE campaigns DROP COLUMN IF EXISTS flight_days;
