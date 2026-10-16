import { filmCropStyle, type AdFilm as AdFilmData } from "../data/adFilms";
import "./AdFilm.css";

/** The campaign's actual ad film, playing in place.
 *
 *  It starts on its own and muted, which is the only kind of autoplay
 *  a browser allows without a gesture. The player's own controls stay,
 *  so sound is one click away; nothing here can unmute for you.
 *
 *  This replaced a still-and-play-button facade. The facade existed so
 *  a visitor who never watched loaded nothing from YouTube — which was
 *  the better default for a dashboard, and is the cost of autoplay:
 *  the player now loads for everyone who opens Overview.
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
    rel: "0",
    modestbranding: "1",
  });

  return (
    <figure className={hero ? "ad-film ad-film-hero" : "ad-film"}>
      <div className="ad-film-frame">
        <iframe
          className="ad-film-player"
          src={`https://www.youtube-nocookie.com/embed/${film.id}?${params}`}
          title={`${campaignName} — ${film.title}`}
          allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
          allowFullScreen
          // The same crop the still gets, so a cinematic film fills the
          // frame while it plays instead of growing its bars back the
          // moment it starts. On a heavily letterboxed film this does
          // push the player's controls past the edge.
          style={filmCropStyle(film, "fill")}
        />
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
