import { describe, expect, it } from "vitest";
import { AD_FILMS, adFilmFor } from "./adFilms";

describe("ad film manifest", () => {
  it("returns nothing for a campaign with no verified film", () => {
    // The important half of the contract: no film is a valid answer.
    // Anything that stood in for one would be inventing creative for a
    // campaign, which is the thing this whole design refuses to do.
    expect(adFilmFor("blinkit")).toBeUndefined();
    expect(adFilmFor("no-such-campaign")).toBeUndefined();
  });

  it("returns the film for a campaign that has one", () => {
    expect(adFilmFor("dream11")?.channel).toBe("Dream11");
  });

  // Whether the video is live, embeddable and still published by this
  // channel is checked against YouTube by scripts/verify-ad-films.mjs,
  // which needs the network. These are the invariants that can be held
  // without it.
  it("records a plausible video id and a channel for every entry", () => {
    for (const [campaign, film] of Object.entries(AD_FILMS)) {
      expect(film.id, campaign).toMatch(/^[\w-]{11}$/);
      expect(film.channel.trim(), campaign).not.toBe("");
      expect(film.title.trim(), campaign).not.toBe("");
    }
  });

  // A creator campaign is "<creator>-<brand>". Attaching a film from a
  // different brand would caption someone else's ad with this
  // campaign's numbers.
  it("attaches each creator campaign a film from that campaign's brand", () => {
    const expected: Record<string, string> = {
      "bhuvan-bam-lenskart": "Lenskart",
      "ms-dhoni-cars24": "Cars24",
      "virat-kohli-puma": "PUMA",
      "salman-khan-thums-up": "ThumsUpOfficial",
      "salman-khan-pepsi": "pepsiindia",
      "alia-bhatt-l-or-al-paris": "L’Oréal Paris India",
    };
    for (const [campaign, channel] of Object.entries(expected)) {
      expect(AD_FILMS[campaign]?.channel, campaign).toBe(channel);
    }
  });
});
