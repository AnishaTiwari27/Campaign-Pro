// Re-checks every entry in src/data/adFilms.ts against YouTube.
//
// Three things can rot in that file and none of them are visible in a
// diff: a video is deleted, its owner turns embedding off, or it is
// re-uploaded by someone who is not the brand. The first two leave a
// dead grey box on the dashboard; the third is worse, because it looks
// fine while attributing a stranger's upload to the advertiser.
//
// oEmbed answers all three at once: it 404s for a video that is gone,
// 401/403s for one that may not be embedded, and names the channel that
// published it. Run it with: node scripts/verify-ad-films.mjs
import { readFileSync } from "node:fs";

const source = readFileSync(new URL("../src/data/adFilms.ts", import.meta.url), "utf8");

// Parsed rather than imported, so this stays a plain node script with no
// TypeScript toolchain in the way.
const entries = [...source.matchAll(
  /"?([\w-]+)"?:\s*\{\s*id:\s*"([^"]+)",\s*channel:\s*"([^"]+)",\s*title:\s*\n?\s*"((?:[^"\\]|\\.)*)"/g,
)].map(([, campaign, id, channel, title]) => ({ campaign, id, channel, title }));

if (entries.length === 0) {
  console.error("No entries parsed from src/data/adFilms.ts — has its shape changed?");
  process.exit(1);
}

// The manifest is a TypeScript source file read as text, so a string
// written with \u escapes arrives here still escaped while YouTube sends
// the real character. Decoding here keeps that from reading as "the
// video was replaced".
function unescapeLiteral(s) {
  return s
    .replace(/\\u([0-9a-fA-F]{4})/g, (_, h) => String.fromCharCode(parseInt(h, 16)))
    .replace(/\\"/g, '"');
}

async function check({ campaign, id, channel, title }) {
  const url =
    "https://www.youtube.com/oembed?url=" +
    encodeURIComponent(`https://www.youtube.com/watch?v=${id}`) +
    "&format=json";
  const res = await fetch(url);
  if (!res.ok) {
    return {
      campaign,
      ok: false,
      why:
        res.status === 404
          ? "video no longer exists"
          : `not embeddable (HTTP ${res.status}) — it would render as a dead box`,
    };
  }
  const data = await res.json();
  if (data.author_name.trim() !== unescapeLiteral(channel)) {
    return {
      campaign,
      ok: false,
      why: `published by "${data.author_name.trim()}", not "${channel}" — this is not the advertiser's own upload`,
    };
  }
  if (data.title.trim() !== unescapeLiteral(title).trim()) {
    return { campaign, ok: false, why: `title is now "${data.title}" — the video was replaced` };
  }
  return { campaign, ok: true, why: `${data.author_name.trim()} — ${data.title}` };
}

const results = [];
for (const entry of entries) {
  results.push(await check(entry));
}

let failed = 0;
for (const r of results) {
  if (!r.ok) failed++;
  console.log(`${r.ok ? "PASS" : "FAIL"}  ${r.campaign.padEnd(24)} ${r.why}`);
}
console.log(`\n${results.length - failed}/${results.length} verified.`);
process.exit(failed === 0 ? 0 : 1);
