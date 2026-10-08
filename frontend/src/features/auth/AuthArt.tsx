// Artwork for the auth screen, drawn rather than photographed.
//
// Same reasoning as SubjectImage: the campaigns, brands and creators in
// this product are fictional, so a stock photo of a "marketing team" would
// dress invented spend figures as something real. These shapes are
// abstract on purpose, and they theme themselves from the design tokens
// instead of shipping two PNGs and picking one.

/** The stylised dashboard shown beside the form. Decorative: the real
 *  numbers live behind the sign-in, so this is hidden from screen
 *  readers rather than described to them. */
export function AuthArt() {
  return (
    <svg
      className="auth-art"
      viewBox="10 18 400 264"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <linearGradient id="auth-bar" x1="0" y1="1" x2="0" y2="0">
          <stop offset="0%" stopColor="var(--accent)" stopOpacity="0.35" />
          <stop offset="100%" stopColor="var(--accent)" stopOpacity="1" />
        </linearGradient>
        <linearGradient id="auth-area" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--accent)" stopOpacity="0.28" />
          <stop offset="100%" stopColor="var(--accent)" stopOpacity="0" />
        </linearGradient>
      </defs>

      {/* main panel */}
      <rect x="18" y="26" width="300" height="196" rx="16" className="auth-art-panel" />
      <rect x="38" y="48" width="86" height="8" rx="4" className="auth-art-line-strong" />
      <rect x="38" y="64" width="52" height="6" rx="3" className="auth-art-line" />

      {/* bars */}
      {[
        { x: 38, h: 44 },
        { x: 74, h: 66 },
        { x: 110, h: 38 },
        { x: 146, h: 82 },
        { x: 182, h: 58 },
        { x: 218, h: 96 },
        { x: 254, h: 74 },
      ].map((b) => (
        <rect
          key={b.x}
          x={b.x}
          y={190 - b.h}
          width="22"
          height={b.h}
          rx="5"
          fill="url(#auth-bar)"
        />
      ))}
      <rect x="38" y="196" width="238" height="2" rx="1" className="auth-art-line" />

      {/* trend card, overlapping */}
      <rect x="236" y="138" width="166" height="104" rx="14" className="auth-art-card" />
      <rect x="254" y="158" width="58" height="7" rx="3.5" className="auth-art-line-strong" />
      <path
        d="M254 214 L280 200 L304 206 L328 184 L356 192 L384 170 L384 224 L254 224 Z"
        fill="url(#auth-area)"
      />
      <path
        d="M254 214 L280 200 L304 206 L328 184 L356 192 L384 170"
        className="auth-art-trend"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="384" cy="170" r="4.5" className="auth-art-dot" />

      {/* approved pill */}
      <rect x="40" y="236" width="128" height="34" rx="17" className="auth-art-card" />
      <circle cx="60" cy="253" r="9" className="auth-art-good" />
      <path
        d="M56 253 l3 3 l5.5 -6"
        stroke="var(--surface)"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <rect x="76" y="249" width="72" height="7" rx="3.5" className="auth-art-line-strong" />
    </svg>
  );
}

const ICONS = {
  // A rising chart: live performance.
  chart: "M3 17l5-5 3 3 6-7M21 8v4h-4",
  // A shield with a tick: approvals that leave a trail.
  shield: "M12 3l7 3v5c0 4.2-2.9 7.7-7 9-4.1-1.3-7-4.8-7-9V6l7-3zm-3 8.5l2 2 4-4.5",
  // An eye: scoped, client-safe visibility.
  eye: "M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6-10-6-10-6zm10 2.5a2.5 2.5 0 100-5 2.5 2.5 0 000 5z",
} as const;

export type FeatureIconName = keyof typeof ICONS;

export function FeatureIcon({ name }: { name: FeatureIconName }) {
  return (
    <svg
      className="auth-feature-icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      <path d={ICONS[name]} />
    </svg>
  );
}
