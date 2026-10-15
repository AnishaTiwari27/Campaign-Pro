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
};

/** Keyed by campaign id. Verified 15 October 2026 via YouTube oEmbed. */
export const AD_FILMS: Record<string, AdFilm> = {
  dream11: {
    id: "voALGUnxdPQ",
    channel: "Dream11",
    title: "Dream11: Aamir ne liya panga 𝚂̶𝙸̶𝙽̶𝙶̶𝙷̶ Kapoor se. | #AapkiTeamMeinKaun",
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
  },
  "salman-khan-thums-up": {
    id: "KPj0DZWyrPY",
    channel: "ThumsUpOfficial",
    title: "Thums Up #AbAayegaToofan with Salman Khan",
  },
  "salman-khan-pepsi": {
    id: "MN2VM6DXeTQ",
    channel: "pepsiindia",
    title: "Salman Khan ke #HarGhoontMeinSwag hai",
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
