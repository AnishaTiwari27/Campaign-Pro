import { useEffect, useState } from "react";
import { adFilmThumb, filmCropStyle, type AdFilm as AdFilmData } from "../data/adFilms";
import "./AdFilm.css";

/** The campaign's actual ad film, played in place.
 *
 *  Nothing from YouTube loads until someone asks for it. Until then
 *  this is a still and a play button — the "facade" pattern, and not
 *  only for weight: an embedded player that mounts on page load sets
 *  cookies and phones home for every visitor who never watches, on a
 *  dashboard where most of them never will. Clicking is consent.
 *
 *  The still is the brand's own video frame, served by YouTube, so even
 *  before anyone presses play the card is showing real creative rather
 *  than a letter in a box.
 */
export function AdFilm({
  film,
  campaignName,
  onPlayingChange,
  hero = false,
}: {
  film: AdFilmData;
  campaignName: string;
  /** Lets a rotating container hold still while something is playing. */
  onPlayingChange?: (playing: boolean) => void;
  /** The spotlight's treatment: the film is the card, not an inset. */
  hero?: boolean;
}) {
  const [playing, setPlaying] = useState(false);
  // maxres does not exist for every upload — plenty of ad films predate
  // it — so a miss falls back to the size that always does.
  const [thumb, setThumb] = useState(adFilmThumb(film.id, "hero"));

  // A different campaign is a different film: stop the old one rather
  // than leave it playing under new numbers.
  useEffect(() => {
    setPlaying(false);
    setThumb(adFilmThumb(film.id, "hero"));
  }, [film.id]);

  useEffect(() => {
    onPlayingChange?.(playing);
  }, [playing, onPlayingChange]);

  return (
    <figure className={hero ? "ad-film ad-film-hero" : "ad-film"}>
      <div className="ad-film-frame">
        {playing ? (
          <iframe
            className="ad-film-player"
            // nocookie, and only ever after a click: the privacy-forward
            // host is pointless if the player has already loaded.
            src={`https://www.youtube-nocookie.com/embed/${film.id}?autoplay=1&rel=0&modestbranding=1`}
            title={film.title}
            allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
            allowFullScreen
          />
        ) : (
          <button
            type="button"
            className="ad-film-poster"
            onClick={() => setPlaying(true)}
            aria-label={`Play the ${campaignName} ad film: ${film.title}`}
          >
            <img
              src={thumb}
              alt=""
              loading="lazy"
              style={filmCropStyle(film, "fill")}
              // hq720 keeps the 16:9 frame; hqdefault would letterbox it.
              onError={() => setThumb(`https://i.ytimg.com/vi/${film.id}/hq720.jpg`)}
            />
            <span className="ad-film-scrim" aria-hidden="true" />
            <span className="ad-film-play" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path d="M8 5.5v13l11-6.5z" fill="currentColor" />
              </svg>
            </span>
            <span className="ad-film-badge" aria-hidden="true">
              Ad film
            </span>
          </button>
        )}
      </div>
      {/* Attribution is not decoration here: it is the claim that this
          is the advertiser's own upload, which is what makes embedding
          it legitimate. It stays visible whether or not it is playing —
          in hero mode it rides over the foot of the picture rather than
          taking a strip of its own. */}
      <figcaption className="ad-film-caption">
        <span className="truncate" title={film.title}>
          {film.title}
        </span>
        <span className="ad-film-source">
          Published by {film.channel} on YouTube
        </span>
      </figcaption>
    </figure>
  );
}
