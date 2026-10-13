import { useState, type CSSProperties } from "react";
import "./SubjectImage.css";

export type SubjectKind = "brand" | "person";

// Deterministic so a subject always gets the same avatar, across reloads
// and across machines. djb2 — small, stable, good enough for hue picking.
function hash(s: string): number {
  let h = 5381;
  for (let i = 0; i < s.length; i++) h = ((h << 5) + h + s.charCodeAt(i)) >>> 0;
  return h;
}

// Generated avatars, not photographs. The people in this system are
// fictional, so a real face would attach invented spend figures to someone
// who didn't run the campaign. A real creator's own photo drops into the
// same slot by giving them an imageUrl.
// Flat-illustration portraits for people, drawn from the seed so the same
// person is the same face everywhere, on every machine, forever.
//
// Deliberately illustrated rather than photographic. These creators are
// fictional, so a real face beside an invented campaign, an invented spend
// figure and an approval trail would read as a record of an endorsement
// that never happened — and in this market personality rights over exactly
// that kind of implied association are enforced. A drawn face carries the
// same recognition value with nobody's likeness attached. A real creator's
// own photograph still wins over this, through imageUrl.
const SKIN = ["#F7D2AE", "#E8B183", "#D19A62", "#B07A45", "#8E5C33", "#6B4226"];
const HAIR = ["#141110", "#3A2418", "#6B3F1F", "#1F1A17", "#52331E", "#2A1C14"];

type HairStyle = 0 | 1 | 2 | 3 | 4 | 5;

/** Six silhouettes, enough that a roster of a dozen reads as a dozen
 *  people rather than a repeated icon. */
function hairPath(style: HairStyle): string {
  switch (style) {
    case 0: // cropped
      return "M15 28c0-11 8-18 17-18s17 7 17 18c0-8-7-10-17-10s-17 2-17 10z";
    case 1: // long, past the shoulders
      return "M14 30c0-12 8-20 18-20s18 8 18 20c0 9 3 14 3 20h-8c0-8-1-13-1-20 0-7-5-11-12-11s-12 4-12 11c0 7-1 12-1 20h-8c0-6 3-11 3-20z";
    case 2: // side part
      return "M15 29c0-12 8-19 17-19s17 7 17 19c0-9-8-9-14-11-4-1-6-2-9 0-4 4-8 4-11 11z";
    case 3: // bun
      return "M15 28c0-11 8-18 17-18s17 7 17 18c0-8-7-10-17-10s-17 2-17 10zM32 1a6 6 0 110 12 6 6 0 010-12z";
    case 4: // curls
      return "M14 30c0-12 8-20 18-20s18 8 18 20c-3-5-6-4-9-8-3 4-6 1-9-2-3 4-6 6-10 4-3 4-7 1-8 6z";
    default: // receding
      return "M18 26c1-9 7-15 14-15s13 5 14 13c-3-5-7-6-14-6-6 0-11 2-14 8z";
  }
}

function GeneratedAvatar({ seed, name, initials, kind, size, fill }: { seed: string; name: string; initials: string; kind: SubjectKind; size: number; fill: boolean }) {
  const isPerson = kind === "person";
  const h = hash(seed);
  const hue = h % 360;
  const hue2 = (hue + 48) % 360;
  const uid = h.toString(36);
  const gradientId = `grad-${uid}`;
  const clipId = `clip-${uid}`;
  // Filling a panel, the artwork is the panel, so it keeps no rounding of
  // its own. The drawn portrait does not come with it: it is legible at
  // avatar size and reads as an emoji at a hundred and seventy pixels, so
  // a subject with no photograph gets the lettered card at this size —
  // a title slate, which is a thing an ad can be.
  const radius = fill ? 0 : kind === "person" ? 32 : 14;
  const drawFace = isPerson && !fill;

  // A brand is a mark, not a face: it keeps the lettered chip.
  const skin = SKIN[(h >> 2) % SKIN.length];
  const hair = HAIR[(h >> 9) % HAIR.length];
  const style = ((h >> 17) % 6) as HairStyle;
  const shirt = `hsl(${(hue + 160) % 360} 42% 42%)`;

  return (
    <svg
      width={fill ? "100%" : size}
      height={fill ? "100%" : size}
      viewBox="0 0 64 64"
      preserveAspectRatio={fill ? "xMidYMid slice" : undefined}
      className={`subject-avatar subject-avatar-${kind}${fill ? " subject-avatar-fill" : ""}`}
      role="img"
      aria-label={name}
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor={`hsl(${hue} 62% 58%)`} />
          <stop offset="100%" stopColor={`hsl(${hue2} 58% 44%)`} />
        </linearGradient>
        {/* The decorative arcs below extend past the avatar's edge; without
            this they paint into the square corners of the viewBox and show
            as hard straight edges. */}
        <clipPath id={clipId}>
          <rect width="64" height="64" rx={radius} />
        </clipPath>
      </defs>
      <g clipPath={`url(#${clipId})`}>
        <rect width="64" height="64" rx={radius} fill={`url(#${gradientId})`} />
        {/* Offset arcs give the block some depth so it reads as artwork
            rather than a flat colour chip. */}
        <circle cx="12" cy="10" r="26" fill="#fff" opacity="0.1" />
        <circle cx="58" cy="56" r="20" fill="#000" opacity="0.08" />

        {drawFace && (
          // Order matters: shoulders, then neck, then head, then hair over
          // the hairline. Drawn inside the clip so the shoulders run off
          // the bottom edge the way a cropped portrait does.
          <g>
            <path d="M6 64c0-13 11-20 26-20s26 7 26 20z" fill={shirt} />
            <path d="M25 42h14v11c0 4-14 4-14 0z" fill={skin} />
            <ellipse cx="32" cy="29" rx="16" ry="18" fill={skin} />
            <path d={hairPath(style)} fill={hair} />
            <circle cx="26" cy="29" r="1.9" fill="#2B2320" />
            <circle cx="38" cy="29" r="1.9" fill="#2B2320" />
            <path d="M27 37c2.5 2 7.5 2 10 0" stroke="#2B2320" strokeWidth="1.6" strokeLinecap="round" fill="none" opacity="0.6" />
          </g>
        )}
      </g>

      {!drawFace && (
        // Lifted in fill mode: the headline's scrim owns the lower third.
        <text
          x="32"
          y={fill ? 26 : 33}
          textAnchor="middle"
          dominantBaseline="central"
          fill="#fff"
          fontSize={initials.length > 2 ? 21 : 25}
          fontWeight="800"
          fontFamily="Archivo, system-ui, sans-serif"
          letterSpacing="-0.5"
        >
          {initials}
        </text>
      )}
    </svg>
  );
}

export function SubjectImage({
  name,
  initials,
  kind,
  domain,
  imageUrl,
  seed,
  size = 40,
  className = "",
  fill = false,
}: {
  name: string;
  initials: string;
  kind: SubjectKind;
  /** Brand domain — resolves to a fetched logo in /logos, if one exists. */
  domain?: string;
  /** An explicit image (a real creator's own photo, say) wins over everything. */
  imageUrl?: string;
  /**
   * Stable identity key for the generated avatar. Defaults to `name`, but a
   * creator appears under several campaign names ("Kabir Sehgal × boAt",
   * "Kabir Sehgal × CRED") and must look the same in all of them, so callers
   * pass the creator id instead.
   */
  seed?: string;
  size?: number;
  className?: string;
  /**
   * Fill the parent instead of drawing a fixed-size chip. The parent must be
   * positioned and clipped; the campaign tiles use this so the art panel is
   * one photograph edge to edge rather than a thumbnail on a gradient.
   */
  fill?: boolean;
}) {
  const [failed, setFailed] = useState(false);

  const src = imageUrl ?? (domain ? `/logos/${domain}.png` : undefined);
  const style: CSSProperties | undefined = fill ? undefined : { width: size, height: size };
  const wrapper = `subject-image${fill ? " subject-image-fill" : ""} ${className}`;

  // Not every domain has a usable logo (fetch-logos.sh skips the ones that
  // come back too small), so a failed load is expected, not exceptional.
  if (!src || failed) {
    return (
      <span className={wrapper} style={style}>
        <GeneratedAvatar seed={seed ?? name} name={name} initials={initials} kind={kind} size={size} fill={fill} />
      </span>
    );
  }

  return (
    <span className={wrapper} style={style}>
      <img
        src={src}
        alt={name}
        width={fill ? undefined : size}
        height={fill ? undefined : size}
        loading="lazy"
        className={fill ? `subject-fill-media subject-fill-media-${kind}` : `subject-logo subject-logo-${kind}`}
        onError={() => setFailed(true)}
      />
    </span>
  );
}
