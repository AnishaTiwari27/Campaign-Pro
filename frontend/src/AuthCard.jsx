import React from "react";
import { AlertTriangle } from "lucide-react";

// Shared shell for LoginForm and SignupForm — same card, same styles, same
// header, so the two auth screens can't visually drift from each other.
// Renders its own <form>; the caller supplies onSubmit and its own fields
// as children.
export default function AuthCard({ onSubmit, error, children }) {
  return (
    <div className="ctp-login-root flex min-h-screen w-full items-center justify-center text-sm">
      <style>{`
        .ctp-login-root { background:var(--base); color:var(--ink); font-family:Inter,ui-sans-serif,system-ui; }
        .ctp-login-root .ctp-display { font-family:Archivo,Inter,ui-sans-serif,system-ui; }
        .ctp-login-card { background:var(--surface); border:1px solid var(--line); width:360px; max-width:92vw; }
        .ctp-login-input { border:1px solid var(--line); background:var(--surface); color:var(--ink); outline:none; }
        .ctp-login-input:focus { border-color:var(--accent); }
        .ctp-login-submit { background:var(--ink); color:var(--base); cursor:pointer; transition:opacity .12s ease; }
        .ctp-login-submit:hover { opacity:.88; }
        .ctp-login-submit:disabled { opacity:.5; cursor:default; }
        .ctp-login-error { background:var(--alert); color:#fff; }
        .ctp-login-link { background:none; border:none; padding:0; cursor:pointer; text-decoration:underline; font:inherit; }
      `}</style>

      <form onSubmit={onSubmit} className="ctp-login-card px-8 py-8 flex flex-col gap-4">
        <div>
          <div className="ctp-display text-lg font-bold leading-none">Campaign</div>
          <div className="ctp-display text-lg font-bold leading-none" style={{ color: "var(--accent)" }}>Tracker Pro</div>
        </div>

        {error && (
          <div className="ctp-login-error flex items-center gap-2 px-3 py-2 text-xs">
            <AlertTriangle size={14} />{error}
          </div>
        )}

        {children}
      </form>
    </div>
  );
}
