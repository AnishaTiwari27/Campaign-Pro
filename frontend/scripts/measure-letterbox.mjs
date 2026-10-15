// Measures the black bars baked into each film's own frame.
//
// A cinematic ad is shot wider than 16:9 and uploaded inside a 16:9
// video, so the bars are part of the picture YouTube serves — not
// something our layout introduced. Virat Kohli's Puma film is a third
// black by height; Salman Khan's Pepsi film 28%. Left alone they read
// as a broken player.
//
// This prints the fraction to crop off each edge, which goes into the
// `crop` field in src/data/adFilms.ts. Re-run it when a film is added
// or replaced:  node scripts/measure-letterbox.mjs
//
// It needs a browser because node cannot decode a JPEG on its own, and
// Playwright is already here for the e2e suite.
import { chromium } from "@playwright/test";
import { readFileSync } from "node:fs";
import os from "node:os";

const src = readFileSync("src/data/adFilms.ts", "utf8");
const films = [...src.matchAll(/"?([\w-]+)"?:\s*\{\s*id:\s*"([^"]+)"/g)].map(([, c, id]) => ({ c, id }));

const exe = `${os.homedir()}/Library/Caches/ms-playwright/chromium-1234/chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing`;
const browser = await chromium.launch({ executablePath: exe });
const page = await browser.newPage();
await page.goto("about:blank");

for (const f of films) {
  const r = await page.evaluate(async (id) => {
    async function load(url) {
      return new Promise((res) => {
        const i = new Image();
        i.crossOrigin = "anonymous";
        i.onload = () => res(i);
        i.onerror = () => res(null);
        i.src = url;
      });
    }
    let img = await load(`https://i.ytimg.com/vi/${id}/maxresdefault.jpg`);
    let which = "maxres";
    if (!img || img.naturalWidth < 2) { img = await load(`https://i.ytimg.com/vi/${id}/hq720.jpg`); which = "hq720"; }
    if (!img) return null;
    const w = img.naturalWidth, h = img.naturalHeight;
    const cv = document.createElement("canvas");
    cv.width = w; cv.height = h;
    const ctx = cv.getContext("2d", { willReadFrequently: true });
    ctx.drawImage(img, 0, 0);
    const d = ctx.getImageData(0, 0, w, h).data;
    const rowDark = (y) => {
      let sum = 0, n = 0;
      for (let x = 0; x < w; x += 7) { const i = (y * w + x) * 4; sum += d[i] + d[i + 1] + d[i + 2]; n += 3; }
      return sum / n;
    };
    let top = 0; while (top < h / 3 && rowDark(top) < 18) top++;
    let bot = 0; while (bot < h / 3 && rowDark(h - 1 - bot) < 18) bot++;
    return { w, h, which, top, bot, frac: +((top + bot) / h).toFixed(4) };
  }, f.id);
  if (!r) {
    console.log(`${f.c.padEnd(24)} FAILED to load a still`);
    continue;
  }
  const top = +(r.top / r.h).toFixed(4);
  const bottom = +(r.bot / r.h).toFixed(4);
  const suggestion = top || bottom ? `crop: { top: ${top}, bottom: ${bottom} },` : "(no bars)";
  console.log(`${f.c.padEnd(24)} ${r.which} ${r.w}x${r.h}  ${suggestion}`);
}
await browser.close();
