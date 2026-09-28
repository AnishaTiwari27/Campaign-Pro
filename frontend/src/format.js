// Shared formatting helpers — split out of CampaignTrackerDashboard.jsx so
// every tab (tabs/*.jsx) can import them without a prop-drilled bundle.

// Campaign dates arrive from the API as "YYYY-MM-DD" strings, not Date
// objects — parse defensively so this also tolerates a full ISO timestamp.
export function fmtDate(iso) {
  const d = new Date(iso.length === 10 ? `${iso}T00:00:00` : iso);
  return d.toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "2-digit" });
}

export function fmtReach(n) {
  return n >= 100000 ? (n / 100000).toFixed(1) + "L" : (n / 1000).toFixed(0) + "K";
}

export function fmtMoney(n) {
  return "₹" + (n >= 100000 ? (n / 100000).toFixed(1) + "L" : (n / 1000).toFixed(0) + "K");
}

// Two-letter initials for the avatar treatment CampaignsTab's tiles and
// OverviewTab's feed rows both use. Lives here, not in either tab file —
// both tabs are separately lazy-loaded routes (see AppRoutes.jsx); a
// static import from one tab's file into another's would merge their
// chunks and undo that split. format.js is already the shared,
// route-agnostic home for this kind of presentational helper.
export function initialsOf(name) {
  const words = name.trim().split(/\s+/);
  return words.length > 1 ? (words[0][0] + words[1][0]).toUpperCase() : name.slice(0, 2).toUpperCase();
}
