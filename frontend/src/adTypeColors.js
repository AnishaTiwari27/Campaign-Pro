// Six-series categorical palette for ad-type identity — validated with the
// dataviz skill's script (scripts/validate_palette.js), not picked by eye:
// worst adjacent pair ΔE 9.6 (CVD, deutan) / 21.6 (normal vision) against
// this app's --base surface, every color ≥3:1 contrast. Deliberately none
// of these reuse --accent (the brand accent) or --secondary (already doing
// double duty as the category-tag color) as a data series — a UI accent
// color moonlighting as one of several chart series is exactly the
// collision the validator exists to catch. Defined as CSS custom
// properties in theme.css (the single source of truth for the palette);
// referenced by var() here instead of repeating the hex values.
//
// Lives in its own module, not inside OverviewTab.jsx (where it
// originated, backing the trend chart) — CampaignsTab's ribbons/tiles
// need the same palette, and OverviewTab/CampaignsTab are separately
// lazy-loaded routes (see AppRoutes.jsx): a static import from one tab's
// file into another's merges their chunks and undoes that split (hit
// exactly this the other direction while wiring Overview's feed rows to
// CampaignsTab's initials-avatar helper — see format.js's initialsOf for
// the twin fix). A shared, route-agnostic module is the only version of
// this that scales past one direction of borrowing.
export const AD_TYPE_COLOR = {
  "Social Media": "var(--ad-social)",
  Influencer: "var(--ad-influencer)",
  "Google Ads": "var(--ad-google)",
  Display: "var(--ad-display)",
  Video: "var(--ad-video)",
  Performance: "var(--ad-performance)",
};
