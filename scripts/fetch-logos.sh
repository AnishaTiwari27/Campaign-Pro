#!/usr/bin/env bash
# Fetches brand logos once into web/public/logos/ so the app serves them
# itself rather than hotlinking a third party on every render — that keeps
# the UI working offline and avoids hammering someone else's endpoint.
#
# Source is Google's favicon service, which is the only domain-to-logo
# endpoint still usable without an API token (Clearbit shut down in Dec
# 2025). It returns the real brand mark for most domains, but resolution
# varies: anything that comes back too small to look right is skipped, and
# the UI falls back to a generated avatar for those brands.
set -uo pipefail

OUT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/frontend/public/logos"
MIN_BYTES=${MIN_BYTES:-1200}
mkdir -p "$OUT"

DOMAINS=(
  dream11.com phonepe.com cred.club amul.com zeptonow.com myntra.com
  boat-lifestyle.com swiggy.com tanishq.co.in licious.in blinkit.com
  lenskart.com mamaearth.in nykaa.com rapido.bike urbancompany.com
  ethoswatches.com lakmeindia.com
)

ok=0; skipped=0
for d in "${DOMAINS[@]}"; do
  tmp="$(mktemp)"
  code=$(curl -s -L --max-time 10 -o "$tmp" -w "%{http_code}" \
    "https://www.google.com/s2/favicons?domain=${d}&sz=128")
  size=$(wc -c < "$tmp" | tr -d ' ')
  mime=$(file -b --mime-type "$tmp")

  if [[ "$code" == "200" && "$mime" == image/* && "$size" -ge "$MIN_BYTES" ]]; then
    mv "$tmp" "$OUT/${d}.png"
    printf '  %-22s %sB\n' "$d" "$size"
    ok=$((ok + 1))
  else
    rm -f "$tmp"
    printf '  %-22s skipped (%s, %s, %sB) — avatar fallback\n' "$d" "$code" "$mime" "$size"
    skipped=$((skipped + 1))
  fi
done

echo "fetched $ok, skipped $skipped -> $OUT"
