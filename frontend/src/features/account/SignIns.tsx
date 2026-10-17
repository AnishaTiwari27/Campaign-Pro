import { useQuery } from "@tanstack/react-query";
import { get } from "../../api/client";
import { SkeletonBlock } from "../../components/Skeleton";
import { EmptyState } from "../../components/EmptyState";
import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import "./Account.css";

type SignIn = {
  at: string;
  userAgent: string;
  ip: string;
  current: boolean;
};

/** Reads a browser's user-agent string into something a person can
 *  recognise their own session from. Deliberately rough: the point is
 *  "was that me on my laptop", not device fingerprinting. */
function describe(ua: string): string {
  if (!ua) return "Unknown device";
  const browser = /Edg\//.test(ua)
    ? "Edge"
    : /Chrome\//.test(ua)
      ? "Chrome"
      : /Safari\//.test(ua)
        ? "Safari"
        : /Firefox\//.test(ua)
          ? "Firefox"
          : "Browser";
  const os = /Mac OS X/.test(ua)
    ? "macOS"
    : /Windows/.test(ua)
      ? "Windows"
      : /Android/.test(ua)
        ? "Android"
        : /iPhone|iPad/.test(ua)
          ? "iOS"
          : /Linux/.test(ua)
            ? "Linux"
            : "";
  return os ? `${browser} on ${os}` : browser;
}

export function SignIns() {
  useSetBreadcrumbs([{ label: "Login history" }]);
  const { data, isLoading } = useQuery({
    queryKey: ["sign-ins"],
    queryFn: () => get<SignIn[]>("/me/sign-ins"),
  });

  if (isLoading) return <SkeletonBlock height={280} />;

  return (
    <div className="account-page">
      <header>
        <h1>Login history</h1>
        <p className="account-page-lede">
          The last ten sessions opened on this account. Recorded at sign-in, and
          only ever your own.
        </p>
      </header>

      {!data || data.length === 0 ? (
        <EmptyState title="No sessions recorded" description="Sign in again and this will fill." />
      ) : (
        <ul className="signins card">
          {data.map((s, i) => (
            <li key={i} className="signin">
              <span className="signin-device">
                {describe(s.userAgent)}
                {s.current && <span className="signin-current">This session</span>}
              </span>
              <span className="signin-meta">
                {new Date(s.at).toLocaleString("en-IN", {
                  dateStyle: "medium",
                  timeStyle: "short",
                })}
                {s.ip ? ` · ${s.ip}` : ""}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
