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
