// useFilters centralizes filter state in the URL (React Router's
// useSearchParams) instead of component state — the gap named in
// docs/ROADMAP.md's Phase A: a filtered view ("Region, filtered to
// Mumbai") had no URL to bookmark or share, and the browser back button
// didn't work. Every filter control anywhere in the app should read/write
// through this hook, not local state, so the URL stays the single source
// of truth.
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

// A filter at its default is omitted from the URL entirely (kept out of
// the query string) so a plain "/overview" is the common case, not
// "/overview?category=All&region=All&...".
const DEFAULTS = { category: "All", region: "All", adType: "All", subjectType: "All", q: "", days: "90" };

export function useFilters() {
  const [searchParams, setSearchParams] = useSearchParams();

  function get(key) {
    return searchParams.get(key) ?? DEFAULTS[key];
  }

  function set(key, value) {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (value === undefined || value === null || value === "" || value === DEFAULTS[key]) {
          next.delete(key);
        } else {
          next.set(key, value);
        }
        return next;
      },
      { replace: true } // a filter tweak replaces the current history entry rather than piling up back-button stops per keystroke/click
    );
  }

  const category = get("category");
  const region = get("region");
  const adType = get("adType");
  const subjectType = get("subjectType");
  const q = get("q");
  const days = Number(get("days")) || 90;
  const subject = searchParams.get("subject") || null; // no default — "focused on one subject" is either on or off, never a fallback value

  // The shape api/client.js's functions expect — "All"/""/undefined all
  // mean "no filter" to the backend (see api/client.js's toQuery), so this
  // just forwards the URL's current values into that shape.
  const filters = useMemo(
    () => ({
      category,
      region,
      adType,
      subjectType: subjectType !== "All" ? subjectType : undefined,
      subject: subject || undefined,
      q: q || undefined,
      days,
    }),
    [category, region, adType, subjectType, subject, q, days]
  );

  return {
    category,
    region,
    adType,
    subjectType,
    q,
    days,
    subject,
    filters,
    setCategory: (v) => set("category", v),
    setRegion: (v) => set("region", v),
    setAdType: (v) => set("adType", v),
    setSubjectType: (v) => set("subjectType", v),
    setQuery: (v) => set("q", v),
    setDays: (v) => set("days", String(v)),
    setSubject: (v) => set("subject", v),
    resetAll: () => setSearchParams(new URLSearchParams(), { replace: true }),
  };
}
