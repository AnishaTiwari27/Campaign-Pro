// Indian-unit formatting, mirrored 1:1 from api/internal/domain/format.go.

export function formatReach(lakh: number): string {
  if (lakh >= 100) return `${(lakh / 100).toFixed(2)}Cr`;
  return `${lakh.toFixed(1)}L`;
}

const CRORE = 1_00_00_000;
const LAKH = 1_00_000;
const THOUSAND = 1_000;

export function formatMoney(rupees: number): string {
  if (rupees >= CRORE) return `₹${(rupees / CRORE).toFixed(2)}Cr`;
  if (rupees >= LAKH) return `₹${(rupees / LAKH).toFixed(1)}L`;
  if (rupees >= THOUSAND) return `₹${(rupees / THOUSAND).toFixed(0)}K`;
  return `₹${Math.round(rupees)}`;
}

export function formatIndex(index: number): string {
  return `${index.toFixed(1)}x`;
}

export function formatCPM(cpm: number): string {
  if (cpm <= 0) return "—";
  return `₹${Math.round(cpm)}`;
}

export function formatPct(pct: number): string {
  return `${Math.round(pct)}%`;
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-IN", { day: "numeric", month: "short", year: "numeric" });
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("en-IN", {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function timeAgo(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diffMs / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}
