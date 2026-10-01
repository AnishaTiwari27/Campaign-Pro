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
function GeneratedAvatar({ seed, name, initials, kind, size }: { seed: string; name: string; initials: string; kind: SubjectKind; size: number }) {
  const h = hash(seed);
  const hue = h % 360;
  const hue2 = (hue + 48) % 360;
  const gradientId = `grad-${h.toString(36)}`;

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      className={`subject-avatar subject-avatar-${kind}`}
      role="img"
      aria-label={name}
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor={`hsl(${hue} 62% 58%)`} />
          <stop offset="100%" stopColor={`hsl(${hue2} 58% 44%)`} />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx={kind === "person" ? 32 : 14} fill={`url(#${gradientId})`} />
      {/* Offset arcs give the block some depth so it reads as artwork
          rather than a flat colour chip. */}
      <circle cx="12" cy="10" r="26" fill="#fff" opacity="0.1" />
      <circle cx="58" cy="56" r="20" fill="#000" opacity="0.08" />
      <text
        x="32"
        y="33"
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
}) {
  const [failed, setFailed] = useState(false);

  const src = imageUrl ?? (domain ? `/logos/${domain}.png` : undefined);
  const style: CSSProperties = { width: size, height: size };

  // Not every domain has a usable logo (fetch-logos.sh skips the ones that
  // come back too small), so a failed load is expected, not exceptional.
  if (!src || failed) {
    return (
      <span className={`subject-image ${className}`} style={style}>
        <GeneratedAvatar seed={seed ?? name} name={name} initials={initials} kind={kind} size={size} />
      </span>
    );
  }

  return (
    <span className={`subject-image ${className}`} style={style}>
      <img
        src={src}
        alt={name}
        width={size}
        height={size}
        loading="lazy"
        className={`subject-logo subject-logo-${kind}`}
        onError={() => setFailed(true)}
      />
    </span>
  );
}
