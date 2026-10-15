/** The button that starts the OAuth round trip.
 *
 *  A link and not a fetch, deliberately: the provider answers with a
 *  top-level redirect to its own sign-in page, and the browser has to
 *  follow that itself. An XHR would receive the redirect and be unable
 *  to act on it.
 *
 *  The same wording on both forms, because it is the same flow and it
 *  does both jobs: an address the provider vouches for gets an account
 *  if it has none. Naming it "sign up" on one form and "sign in" on the
 *  other would suggest two routes where there is one.
 */
export function SsoButton({ label }: { label: string }) {
  return (
    <>
      <p className="auth-or">or</p>
      <a className="btn auth-submit auth-sso" href="/api/auth/sso/start">
        Continue with {label}
      </a>
    </>
  );
}

/** What went wrong on the way back from the provider.
 *
 *  The callback cannot answer with JSON — it is a browser navigation, so
 *  it redirects and says why in the query string. Without this the trip
 *  ends on a blank sign-in form, which looks like the app lost the
 *  attempt rather than refused it.
 */
export function ssoFailureMessage(search: string): string | null {
  switch (new URLSearchParams(search).get("sso")) {
    case "denied":
      return "That sign-in was cancelled. Nothing was shared.";
    case "expired":
      return "That sign-in took too long. Please start again.";
    case "unverified":
      return "Your provider has not verified that email address, so it cannot be used to sign in.";
    case "failed":
      return "That sign-in could not be completed. Please try again, or use your password.";
    default:
      return null;
  }
}
