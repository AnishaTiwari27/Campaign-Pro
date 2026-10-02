import { useToastStore } from "../app/toastStore";
import "./Toasts.css";

export function Toasts() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);

  if (toasts.length === 0) return null;

  return (
    <div className="toast-stack" role="status" aria-live="polite">
      {toasts.map((t) => (
        <div key={t.id} className={`toast toast-${t.kind}`}>
          <span className="toast-icon" aria-hidden="true">
            {t.kind === "positive" ? "✓" : "!"}
          </span>
          <span className="toast-message">{t.message}</span>
          <button className="toast-dismiss" aria-label="Dismiss" onClick={() => dismiss(t.id)}>
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
