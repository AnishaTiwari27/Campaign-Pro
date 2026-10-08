import { useState } from "react";
import { useAuthOptions } from "../../api/auth";
import { AuthArt, FeatureIcon, type FeatureIconName } from "./AuthArt";
import { SignInForm } from "./SignInForm";
import { SignUpForm } from "./SignUpForm";
import "./AuthPage.css";

const FEATURES: { icon: FeatureIconName; title: string; body: string }[] = [
  {
    icon: "chart",
    title: "Spend and ROAS, as they land",
    body: "Six ad types in one view, with anomalies flagged while the budget can still be moved.",
  },
  {
    icon: "shield",
    title: "Approvals that leave a trail",
    body: "Every sign-off is recorded against the person who made it, so a spend decision can be traced back.",
  },
  {
    icon: "eye",
    title: "Client accounts see less, by design",
    body: "No approval queue, no cross-account benchmarks, and no way to sign off on spend — enforced on the server, not just hidden.",
  },
];

type Mode = "signin" | "signup";

export function AuthPage() {
  // #signup opens the form directly, so a link can point at it — an
  // invite mail, or the demo link itself. Read once; this screen is
  // replaced by the app as soon as a session exists.
  const [mode, setMode] = useState<Mode>(() =>
    window.location.hash === "#signup" ? "signup" : "signin",
  );
  // Carried across the switch so someone who has just registered does not
  // retype the address they typed a second ago.
  const [justCreated, setJustCreated] = useState<string | null>(null);

  // If this request fails the screen stays sign-in only. Hiding a working
  // link is a smaller harm than offering one that leads to a 404.
  const { data: options } = useAuthOptions();
  const signupEnabled = options?.signupEnabled ?? false;
  const minPasswordLength = options?.minPasswordLength ?? 10;

  // Defensive: if signup is switched off server-side while this tab is
  // open, fall back rather than render a form that cannot submit.
  const showSignup = mode === "signup" && signupEnabled;

  return (
    <div className="auth-page">
      <div className="auth-shell">
        <aside className="auth-aside">
          <div className="auth-brand">
            <span className="auth-mark">CTP</span>
            <span className="auth-brand-text">
              Campaign Tracker Pro
              <span className="auth-brand-sub">India · all regions</span>
            </span>
          </div>

          <div className="auth-pitch">
            <p className="auth-eyebrow">Campaign intelligence</p>
            <h2>Every rupee, every creator, one view.</h2>
            <p className="auth-lede">
              Plan, track and sign off on campaigns across social, influencer,
              search, display and video — without three spreadsheets and a
              guess.
            </p>
          </div>

          <AuthArt />

          <ul className="auth-features">
            {FEATURES.map((f) => (
              <li key={f.title}>
                <span className="auth-feature-badge">
                  <FeatureIcon name={f.icon} />
                </span>
                <span>
                  <strong>{f.title}</strong>
                  <small>{f.body}</small>
                </span>
              </li>
            ))}
          </ul>

          <p className="auth-foot">Reporting cadences computed in IST.</p>
        </aside>

        <main className="auth-main">
          <div className="auth-card card">
            {/* The brand repeats here because the panel beside it is hidden
                on a narrow screen, where this is the whole page. */}
            <div className="auth-brand auth-brand-compact">
              <span className="auth-mark">CTP</span>
              <span className="auth-brand-text">
                Campaign Tracker Pro
                <span className="auth-brand-sub">India · all regions</span>
              </span>
            </div>

            {showSignup ? (
              <SignUpForm
                minPasswordLength={minPasswordLength}
                onCreated={(email) => {
                  setJustCreated(email);
                  setMode("signin");
                }}
                onSwitchToSignIn={() => setMode("signin")}
              />
            ) : (
              <SignInForm
                initialEmail={justCreated ?? ""}
                notice={
                  justCreated
                    ? "Account created. Sign in with the password you just chose."
                    : null
                }
                onSwitchToSignUp={
                  signupEnabled
                    ? () => {
                        setJustCreated(null);
                        setMode("signup");
                      }
                    : undefined
                }
              />
            )}
          </div>
        </main>
      </div>
    </div>
  );
}
