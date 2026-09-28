// Shared label/color mapping for a campaign's pacing (computed
// server-side — services/campaigns/internal/models's PacingOf; this is
// purely presentational, deciding how to label and color whatever the API
// already decided). Reused by the campaign drawer, the Campaigns table,
// and Overview's budget-alerts highlight so all three read it identically.
export const PACING_LABEL = { over: "Over pace", under: "Under pace", on: "On pace" };

// Reuses --alert/--muted/--status-live rather than adding a new color
// just for pacing — "over" is the same risk signal --alert already
// carries, "on" the same good signal --status-live carries for a Live
// campaign, "under" a neutral, non-alarming note.
export const PACING_COLOR = { over: "var(--alert)", under: "var(--muted)", on: "var(--status-live)" };
