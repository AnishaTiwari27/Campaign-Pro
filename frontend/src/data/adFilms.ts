import type React from "react";

/** The real ad film a campaign is running, where one can be verified.
 *
 *  Every entry is a video published by the brand on its *own* YouTube
 *  channel and left embeddable by that brand. That is the whole
 *  licensing argument: the player is YouTube's, the upload is the
 *  advertiser's, the attribution travels with it, and nothing is copied
 *  or re-hosted here. It is the same thing any publisher does when it
 *  embeds a brand's ad.
 *
 *  What is deliberately NOT here: creative lifted from an ad server, a
 *  still composited out of a celebrity photo and a logo, or a film from
 *  a fan channel that re-uploaded it. The first two fabricate an
 *  advertisement; the third attributes someone else's upload to the
 *  brand. `channel` is recorded for exactly that reason — it is the
 *  evidence that this is the advertiser's own post, and
 *  `scripts/verify-ad-films.mjs` re-checks it against YouTube rather
 *  than trusting this file.
 *
 *  A campaign with no verified film is not a gap to be filled with
 *  something approximate. It simply shows what it shows today.
 */
export type AdFilm = {
  /** YouTube video id. */
  id: string;
  /** The channel that published it — must be the advertiser's own. */
  channel: string;
  /** The title as YouTube reports it, so a swapped video is noticed. */
  title: string;
  /** Black bars baked into the film's own frame, as a fraction of its
   *  height. A cinematic ad is shot wider than 16:9 and uploaded
   *  inside a 16:9 video, so the bars are in the picture YouTube
   *  serves rather than anything the layout did. Measured by
   *  scripts/measure-letterbox.mjs; absent means none. */
  crop?: { top: number; bottom: number };
};

/** Keyed by campaign id. Verified 15 October 2026 via YouTube oEmbed. */
export const AD_FILMS: Record<string, AdFilm> = {
  dream11: {
    id: "voALGUnxdPQ",
    channel: "Dream11",
    title: "Dream11: Aamir ne liya panga 𝚂̶𝙸̶𝙽̶𝙶̶𝙷̶ Kapoor se. | #AapkiTeamMeinKaun",
    crop: { top: 0, bottom: 0.05 },
  },
  lenskart: {
    id: "kPMLVO45XaU",
    channel: "Lenskart",
    title:
      "#HalkaRakhYaar I Experience Halke Phulke Frames Ft. BB Ki Vines Bhuvan Bam I Lenskart.com",
  },
  swiggy: {
    id: "c6j711sEJdA",
    channel: "Swiggy",
    title:
      "Swiggy Instamart - Get groceries and more delivered in minutes. Aap kiske saath dekhoge?",
  },
  myntra: {
    id: "hisIcGdSiAg",
    channel: "Myntra",
    title: "Ready For India's Biggest Fashion Sale? | Myntra End of Reason Sale",
  },

  blinkit: {
    id: "v29TSTqRlP8",
    channel: "Blinkit",
    title: "Ghar Bulao Khushiyan | Delivering a Happy Diwali in 10 Minutes",
  },
  phonepe: {
    id: "gBQwlE6nNiY",
    channel: "PhonePe App",
    title: "PhonePe - Utilities Advertisement (Hindi)",
    crop: { top: 0.0556, bottom: 0.0556 },
  },
  zepto: {
    id: "Lmk9nmVXUFw",
    channel: "Zepto",
    title: "Zepto - 10 Minute Grocery Delivery | Fresh Fruits & Vegetables",
  },
  boat: {
    id: "WIhpzOJFm0w",
    channel: "boAt",
    title:
      "boAt X Netflix | Presenting The Stream Edition, Made For India To Keep Watching",
  },
  rapido: {
    id: "bgGQYrlpRHA",
    channel: "Rapido",
    title: "#GoRapido- Don’t be late, Save your date",
    crop: { top: 0.0542, bottom: 0.0556 },
  },

  // Creator campaigns get the film that features that person for that
  // brand — not merely a film by the brand. A Lenskart ad without Bhuvan
  // Bam in it would be the wrong evidence on a campaign that is about
  // him fronting it.
  "bhuvan-bam-lenskart": {
    id: "pkT0mqq6MGU",
    channel: "Lenskart",
    title: "Lenskart Air As Seen On Bhuvan Bam | BB Ki Vines | Lenskart",
  },
  "ms-dhoni-cars24": {
    id: "3zV6g01h-do",
    channel: "Cars24",
    title:
      "Buying A Car? CARS24 MRL Check Wali Hi Lena Yaar | Dhoni Visits Best Used Car Platform",
  },
  "virat-kohli-puma": {
    id: "j078Y9Au8-M",
    channel: "PUMA",
    title: "PUMA INDIA | One8 by PUMAxVirat Kohli #ComeOutAndPlay | Wanna skip this ad? That's cool!",
    crop: { top: 0.1653, bottom: 0.1667 },
  },
  "salman-khan-thums-up": {
    id: "KPj0DZWyrPY",
    channel: "ThumsUpOfficial",
    title: "Thums Up #AbAayegaToofan with Salman Khan",
    crop: { top: 0.1208, bottom: 0.0903 },
  },
  "salman-khan-pepsi": {
    id: "MN2VM6DXeTQ",
    channel: "pepsiindia",
    title: "Salman Khan ke #HarGhoontMeinSwag hai",
    crop: { top: 0.1403, bottom: 0.1403 },
  },
  "alia-bhatt-l-or-al-paris": {
    id: "xfrr3fk1_T0",
    channel: "L’Oréal Paris India",
    title:
      "L'Oréal Paris Mujh Mein Hai Diwali ft. Alia Bhatt, Akansha Ranjan Kapoor & Anushka Ranjan",
  },
};

export function adFilmFor(campaignId: string): AdFilm | undefined {
  return AD_FILMS[campaignId];
}

/** A still from the film, served by YouTube.
 *
 *  Size matters more than it looks. `hqdefault` is 480x360 — a 4:3
 *  frame with black bars baked in for a 16:9 video — so it is the
 *  wrong default despite being the one that always exists.
 *  `mqdefault` is 320x180, genuinely 16:9, and also always exists,
 *  which makes it right for a tile. `maxresdefault` is 1280x720 but
 *  is missing on plenty of older uploads, so anything using it needs
 *  a fallback.
 */
export function adFilmThumb(id: string, size: "tile" | "hero"): string {
  return `https://i.ytimg.com/vi/${id}/${size === "hero" ? "maxresdefault" : "mqdefault"}.jpg`;
}

/** Inline style that hides a film's baked-in bars.
 *
 *  "clip" trims them and leaves the gap to whatever is behind — right
 *  for a tile, which has a blurred copy of the frame filling the box
 *  anyway. "fill" trims them and then scales the remaining band back
 *  up so it covers the frame, which is what the spotlight wants: no
 *  black, edge to edge. The translate re-centres a film whose bars are
 *  uneven, as Salman Khan's Thums Up upload is.
 *
 *  Playback is untouched. The video YouTube serves still carries its
 *  own bars, and scaling the player to match would push its controls
 *  out of view — so this is the still only.
 */
export function filmCropStyle(
  film: AdFilm,
  mode: "clip" | "fill",
  /** Extra scale for the blurred backdrop layer, which oversizes
   *  itself so the blur has pixels to bleed from at the edges. */
  bleed = 1,
): React.CSSProperties | undefined {
  const crop = film.crop;
  if (!crop || (crop.top === 0 && crop.bottom === 0)) return undefined;
  const clipPath = `inset(${(crop.top * 100).toFixed(2)}% 0 ${(crop.bottom * 100).toFixed(2)}% 0)`;
  if (mode === "clip") return { clipPath };
  const scale = bleed / (1 - crop.top - crop.bottom);
  const shift = ((crop.bottom - crop.top) / 2) * 100;
  return { clipPath, transform: `translateY(${shift.toFixed(2)}%) scale(${scale.toFixed(4)})` };
}
