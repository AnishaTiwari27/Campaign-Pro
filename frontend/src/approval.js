// Shared label/color mapping for a campaign's approval status (computed
// server-side — services/campaigns/internal/models's ApprovalStatusValid;
// this is purely presentational). Same two-flat-object-export shape as
// pacing.js, reused by the campaign drawer, the Campaigns table, and
// Overview's alert banner so all three read it identically.
export const APPROVAL_LABEL = { pending: "Pending review", approved: "Approved", rejected: "Rejected" };

// Reuses existing CSS vars rather than adding new ones, same reasoning
// pacing.js's palette gives: "pending" needs attention (--alert, the same
// risk signal pacing's "over" already carries), "approved" is the good
// signal (--status-live), "rejected" a neutral, non-alarming note (--muted).
export const APPROVAL_COLOR = { pending: "var(--alert)", approved: "var(--status-live)", rejected: "var(--muted)" };
