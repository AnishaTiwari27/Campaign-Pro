import { useCallback, useEffect, useRef, useState } from "react";
import { filmCropStyle, type AdFilm as AdFilmData } from "../data/adFilms";
import "./AdFilm.css";

/** The campaign's actual ad film, playing in place.
 *
 *  It starts on its own and muted, which is the only kind of autoplay
 *  a browser allows without a gesture.
 *
 *  YouTube's own chrome is switched off — `controls=0` — because a
 *  scrub bar, a play button, a title and a "More videos" strip sat
 *  permanently over the ad and over this card's own header. None of
 *  it can be styled or hidden from out here: the player is a
 *  cross-origin iframe, so CSS cannot reach inside it. The only
 *  lever is to not ask for it.
 *
 *  What replaces it is a mute and a pause button of our own, shown on
 *  hover and driven through the iframe API. They are the two controls
 *  that matter for an ad playing itself on a dashboard.
 */
export function AdFilm({
  film,
  campaignName,
  hero = false,
}: {
  film: AdFilmData;
  campaignName: string;
  /** The spotlight's treatment: the film is the card, not an inset. */
  hero?: boolean;
}) {
  const frame = useRef<HTMLIFrameElement>(null);
  const [muted, setMuted] = useState(true);
  const [paused, setPaused] = useState(false);

  // A new film is a new player: it arrives muted and playing, so the
  // buttons must not keep claiming the last one's state.
  useEffect(() => {
    setMuted(true);
    setPaused(false);
  }, [film.id]);

  const command = useCallback((func: string) => {
    frame.current?.contentWindow?.postMessage(
      JSON.stringify({ event: "command", func, args: [] }),
      "https://www.youtube-nocookie.com",
    );
  }, []);

  const params = new URLSearchParams({
    autoplay: "1",
    // Required. An unmuted autoplay is refused and the film would sit
    // on its first frame instead.
    mute: "1",
    // iOS otherwise takes the video fullscreen the moment it starts.
    playsinline: "1",
    // Ads are short; without this a 20-second film spends the rest of
    // its turn in the carousel showing YouTube's end screen.
    loop: "1",
    playlist: film.id,
    // No scrub bar, no play button, no title, no "More videos".
    controls: "0",
    // What lets the buttons below talk to the player.
    enablejsapi: "1",
    rel: "0",
  });

  return (
    <figure className={hero ? "ad-film ad-film-hero" : "ad-film"}>
      <div className="ad-film-frame">
        <iframe
          ref={frame}
          className="ad-film-player"
          src={`https://www.youtube-nocookie.com/embed/${film.id}?${params}`}
          title={`${campaignName} — ${film.title}`}
          allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
          allowFullScreen
          // The same crop the still gets, so a cinematic film fills the
          // frame while it plays instead of growing its bars back the
          // moment it starts.
          style={filmCropStyle(film, "fill")}
        />

        {/* Hidden until the pointer is over the film, and reachable by
            keyboard regardless — focus inside it keeps it visible. */}
        <div className="ad-film-controls">
          <button
            type="button"
            onClick={() => {
              command(paused ? "playVideo" : "pauseVideo");
              setPaused((p) => !p);
            }}
            aria-label={paused ? "Play the ad film" : "Pause the ad film"}
          >
            {paused ? (
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M8 5.5v13l11-6.5z" fill="currentColor" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M7 5h3.2v14H7zm6.8 0H17v14h-3.2z" fill="currentColor" />
              </svg>
            )}
          </button>
          <button
            type="button"
            onClick={() => {
              command(muted ? "unMute" : "mute");
              setMuted((m) => !m);
            }}
            aria-label={muted ? "Unmute the ad film" : "Mute the ad film"}
          >
            {muted ? (
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M4 9v6h3.5L12 19V5L7.5 9zm12.5 3 2.8-2.8-1-1L15.5 11l-2.8-2.8-1 1L14.5 12l-2.8 2.8 1 1 2.8-2.8 2.8 2.8 1-1z"
                  fill="currentColor"
                />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M4 9v6h3.5L12 19V5L7.5 9zm10.5 3a3.5 3.5 0 0 0-2-3.2v6.4a3.5 3.5 0 0 0 2-3.2zm-2-7v2.1a5.5 5.5 0 0 1 0 9.8V19a7.5 7.5 0 0 0 0-14z"
                  fill="currentColor"
                />
              </svg>
            )}
          </button>
        </div>
      </div>

      {/* Attribution is not decoration here: it is the claim that this
          is the advertiser's own upload, which is what makes embedding
          it legitimate. */}
      <figcaption className="ad-film-caption">
        <span className="truncate" title={film.title}>
          {film.title}
        </span>
        <span className="ad-film-source">
          {film.featuring && film.featuring.length > 0 && (
            <>Featuring {film.featuring.join(", ")} · </>
          )}
          Published by {film.channel} on YouTube
        </span>
      </figcaption>
    </figure>
  );
}
