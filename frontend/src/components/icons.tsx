// Small hand-drawn icon set — no icon library per the spec's plain-CSS,
// no-component-library constraint. Each is a fixed 18x18 viewBox so they
// drop into nav rows and buttons without extra sizing.
import type { SVGProps } from "react";

function Base(props: SVGProps<SVGSVGElement>) {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" {...props} />
  );
}

export function IconOverview(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <rect x="2.5" y="2.5" width="5.5" height="5.5" rx="1.2" />
      <rect x="10" y="2.5" width="5.5" height="8.5" rx="1.2" />
      <rect x="2.5" y="10.5" width="5.5" height="5" rx="1.2" />
      <rect x="10" y="13" width="5.5" height="2.5" rx="1" />
    </Base>
  );
}

export function IconCampaigns(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <line x1="3" y1="5" x2="15" y2="5" />
      <line x1="3" y1="9" x2="15" y2="9" />
      <line x1="3" y1="13" x2="11" y2="13" />
    </Base>
  );
}

export function IconApprovals(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <rect x="3" y="2.5" width="12" height="13" rx="1.5" />
      <path d="M6 9l2 2 4-4.5" />
    </Base>
  );
}

export function IconRegions(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M9 16s5.5-5.2 5.5-9A5.5 5.5 0 003.5 7c0 3.8 5.5 9 5.5 9z" />
      <circle cx="9" cy="7" r="1.8" />
    </Base>
  );
}

/** A bookmarked page — a published record rather than a live metric. */
export function IconEndorsements(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M4.5 2.5h9a1 1 0 011 1v13l-5.5-3-5.5 3v-13a1 1 0 011-1z" />
    </Base>
  );
}

/** Concentric sweep — something scanning for what is new. */
export function IconRadar(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="10" cy="10" r="7" />
      <circle cx="10" cy="10" r="3" />
      <line x1="10" y1="10" x2="15" y2="5" />
    </Base>
  );
}

export function IconSignals(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <line x1="3" y1="10" x2="17" y2="10" />
      <line x1="10" y1="3" x2="10" y2="17" />
      <circle cx="6" cy="6" r="1.6" />
      <circle cx="14" cy="14" r="1.6" />
    </Base>
  );
}

export function IconBenchmarks(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <line x1="4" y1="15" x2="4" y2="8" />
      <line x1="9" y1="15" x2="9" y2="4" />
      <line x1="14" y1="15" x2="14" y2="11" />
    </Base>
  );
}

export function IconReports(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M5 2.5h5.5L13.5 5.5V15.5h-8.5z" />
      <path d="M10.3 2.5v3.3h3.2" />
      <line x1="6.7" y1="9.5" x2="11.3" y2="9.5" />
      <line x1="6.7" y1="12" x2="11.3" y2="12" />
    </Base>
  );
}

export function IconUsers(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="7.5" cy="7" r="2.6" />
      <path d="M3 16c0-2.5 2-4 4.5-4s4.5 1.5 4.5 4" />
      <path d="M13.5 5.2a2.6 2.6 0 010 4.6M14.5 12.4c1.7.5 2.5 1.9 2.5 3.6" />
    </Base>
  );
}

export function IconSettings(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="9" cy="9" r="2.4" />
      <path d="M9 2.7v1.6M9 13.7v1.6M15.3 9h-1.6M4.3 9H2.7M13.1 4.9l-1.1 1.1M6 12.1l-1.1 1.1M13.1 13.1l-1.1-1.1M6 5.9L4.9 4.9" />
    </Base>
  );
}

export function IconSearch(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="8" cy="8" r="4.8" />
      <line x1="12.2" y1="12.2" x2="16" y2="16" />
    </Base>
  );
}

export function IconChevronDown(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M4.5 7l4.5 4.5L13.5 7" />
    </Base>
  );
}

export function IconCreators(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="7" cy="6.5" r="2.6" />
      <path d="M2.6 15.2c0-2.4 2-4.1 4.4-4.1s4.4 1.7 4.4 4.1" />
      <path d="M12.4 4.4a2.6 2.6 0 010 4.6" />
      <path d="M13.6 11.4c1.2.6 1.9 1.8 1.9 3.3" />
    </Base>
  );
}

export function IconLogout(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M11 13.5v1.4a1.2 1.2 0 01-1.2 1.2H4.2A1.2 1.2 0 013 14.9V3.1a1.2 1.2 0 011.2-1.2h5.6A1.2 1.2 0 0111 3.1v1.4" />
      <path d="M14.2 9H6.8" />
      <path d="M12.1 6.6L14.5 9l-2.4 2.4" />
    </Base>
  );
}

/** Theme switch. Auto is the dial half-filled — the same circle as
 *  the other two, with the system deciding which half you get. */
export function IconAuto(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="9" cy="9" r="6.2" />
      <path d="M9 2.8a6.2 6.2 0 0 0 0 12.4z" fill="currentColor" stroke="none" />
    </Base>
  );
}

export function IconSun(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="9" cy="9" r="3.4" />
      <path d="M9 1.6v1.6M9 14.8v1.6M16.4 9h-1.6M3.2 9H1.6M14.2 3.8l-1.1 1.1M4.9 13.1l-1.1 1.1M14.2 14.2l-1.1-1.1M4.9 4.9 3.8 3.8" />
    </Base>
  );
}

export function IconMoon(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M15 10.4A6.4 6.4 0 0 1 7.6 3a6.6 6.6 0 1 0 7.4 7.4z" />
    </Base>
  );
}

/** Billing. A card, because that is what the page is about. */
export function IconBilling(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <rect x="1.8" y="4" width="14.4" height="10" rx="1.8" />
      <line x1="1.8" y1="7.6" x2="16.2" y2="7.6" />
      <line x1="4.6" y1="11.2" x2="7.4" y2="11.2" />
    </Base>
  );
}

export function IconBell(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M4.6 7.4a4.4 4.4 0 1 1 8.8 0c0 3.1 1.2 4.3 1.2 4.3H3.4s1.2-1.2 1.2-4.3z" />
      <path d="M7.3 14.2a1.9 1.9 0 0 0 3.4 0" />
    </Base>
  );
}

export function IconLifebuoy(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <circle cx="9" cy="9" r="6.6" />
      <circle cx="9" cy="9" r="2.7" />
      <path d="M4.3 4.3 7 7M11 11l2.7 2.7M13.7 4.3 11 7M7 11l-2.7 2.7" />
    </Base>
  );
}

export function IconSparkle(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M9 2.2 10.5 6.6 15 8.1l-4.5 1.5L9 14.1 7.5 9.6 3 8.1l4.5-1.5z" />
    </Base>
  );
}

export function IconHistory(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M3 9a6 6 0 1 0 1.8-4.3" />
      <path d="M2.6 2.9v3.4h3.4" />
      <path d="M9 5.9V9l2.2 1.5" />
    </Base>
  );
}

export function IconHandshake(props: SVGProps<SVGSVGElement>) {
  return (
    <Base {...props}>
      <path d="M2.2 7.4 5 5.2l2.6 1.9 2.8-1.9 2.4 2.2" />
      <path d="M5 5.2 1.6 8.6l2.2 2.2 1.6-1.3" />
      <path d="M13 7.4l3.4 1.2-2 2.6-1.8-1" />
      <path d="M5.4 9.5l2.4 2.3 1.4-1 1.6 1.4 1.5-1.3" />
    </Base>
  );
}
