import { describe, expect, it } from "vitest";
import { AD_FILMS, adFilmFor } from "./adFilms";

describe("ad film manifest", () => {
  it("returns nothing for a campaign with no verified film", () => {
    // The important half of the contract: no film is a valid answer.
    // Anything that stood in for one would be inventing creative for a
    // campaign, which is the thing this whole design refuses to do.
    // Not a campaign that merely lacks one today — Blinkit used to sit
    // here and then a film was verified for it, which is the normal
    // direction of travel. An id that cannot ever gain one keeps this
    // testing the contract rather than the current coverage.
    expect(adFilmFor("no-such-campaign")).toBeUndefined();
    expect(adFilmFor("")).toBeUndefined();
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

  // Who appears in an ad is a claim about a real person, so it may
  // only come from what the film itself credits — never from
  // reporting about the campaign, and never from a guess.
  it("names a cast only where the film's own title credits one", () => {
    for (const [campaign, film] of Object.entries(AD_FILMS)) {
      if (!film.featuring) continue;
      expect(film.featuring.length, campaign).toBeGreaterThan(0);
      for (const person of film.featuring) {
        expect(person.trim(), campaign).not.toBe("");
      }
    }
    expect(AD_FILMS.cred?.featuring).toEqual(["Anil Kapoor"]);
    expect(AD_FILMS.tanishq?.featuring).toContain("Radhika Apte");
    // A brand film with nobody named must not acquire one.
    expect(AD_FILMS.swiggy?.featuring).toBeUndefined();
  });

  // Measured by scripts/measure-letterbox.mjs. A crop outside this
  // range is a measurement that went wrong, and it would either leave
  // bars or eat the picture.
  it("keeps every crop within a sane range", () => {
    for (const [campaign, film] of Object.entries(AD_FILMS)) {
      if (!film.crop) continue;
      const { top, bottom } = film.crop;
      expect(top, campaign).toBeGreaterThanOrEqual(0);
      expect(bottom, campaign).toBeGreaterThanOrEqual(0);
      expect(top + bottom, campaign).toBeLessThan(0.45);
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
