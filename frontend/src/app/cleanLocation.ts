/** Whether a character can appear in a path the router will match.
 *
 *  C0 controls and DEL cannot. A newline is the one that actually
 *  turns up: a URL copied with its line break still attached arrives
 *  as "/overview%0A".
 *
 *  Written as a codepoint test rather than a character class, because
 *  a regex holding literal control characters is the kind of thing a
 *  linter objects to and a reader has to squint at.
 */
function routable(ch: string): boolean {
  const code = ch.codePointAt(0) ?? 0;
  return code > 0x1f && code !== 0x7f;
}

/** Percent-encoded C0 controls and DEL: %00–%1F and %7F.
 *
 *  These have to be matched in the encoded form as well as the
 *  decoded one. window.location.pathname hands back the path still
 *  encoded, so a pasted line break arrives as the three characters
 *  "%0A" — every one of them printable, and none caught by a filter
 *  looking for control characters. The router decodes before it
 *  matches, which is why it sees a newline and this did not.
 */
const ENCODED_CONTROL = /%(?:0[0-9A-Fa-f]|1[0-9A-Fa-f]|7F)/gi;

/** Strips what cannot be routed from a pathname.
 *
 *  Also trims trailing whitespace, encoded or not, which is legal in
 *  a path but never meant — nothing in this app has a route ending in
 *  one.
 */
export function cleanPath(pathname: string): string {
  const cleaned = [...pathname.replace(ENCODED_CONTROL, "")]
    .filter(routable)
    .join("")
    .replace(/(?:%20|\s)+$/i, "");
  return cleaned === "" ? "/" : cleaned;
}

/** Rewrites the address bar when the path holds something unroutable.
 *
 *  This exists because of a sharp edge in the router, not a taste for
 *  tidy URLs. React Router compiles a route pattern to a regular
 *  expression, and the catch-all becomes `(.*)`. In JavaScript `.`
 *  does not match a newline unless the pattern is compiled with the
 *  dotAll flag, which it is not — so a path containing a newline
 *  matches no route at all. Not the page, and not the catch-all that
 *  exists to say "page not found". The app renders nothing: no rail,
 *  no error, a blank screen.
 *
 *  Run before the router reads the location, so the first render
 *  already has a path it can match. replaceState rather than a
 *  redirect, because the newline was never meant to be there and does
 *  not deserve an entry in the history someone has to press Back
 *  through.
 */
export function cleanLocationInPlace(win: Window = window): void {
  const { pathname, search, hash } = win.location;
  const cleaned = cleanPath(pathname);
  if (cleaned === pathname) return;
  win.history.replaceState(null, "", cleaned + search + hash);
}
