-- Planned flight length, so "pace" can mean pace against plan.
--
-- days_running is elapsed time. Without the planned span there is no way to
-- tell a campaign that is halfway through its flight from one that is
-- underspending: both sit at 50% of budget. That ambiguity is why the
-- Signals quadrant measures budget consumed rather than pace against plan.
--
-- Days rather than an end date, deliberately: the whole model counts days
-- (days_running, the 8-point curve interpolated across them) and never
-- stores a calendar date for a campaign. An ends_at would need a starts_at
-- beside it and a timezone decision, to express the same number.
--
-- Nullable, because a campaign whose planned length nobody recorded is a
-- real state and must not be guessed at. Code reads an absent value as
-- "cannot compute pace against plan" rather than substituting days_running,
-- which would silently report every such campaign as exactly on plan.
ALTER TABLE campaigns ADD COLUMN flight_days INT;

-- A flight cannot be shorter than what has already elapsed, and zero-day
-- flights make the ratio undefined.
ALTER TABLE campaigns ADD CONSTRAINT campaigns_flight_days_check
    CHECK (flight_days IS NULL OR flight_days >= GREATEST(days_running, 1));
