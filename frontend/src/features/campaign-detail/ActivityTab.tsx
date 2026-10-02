import { useState, type FormEvent } from "react";
import type { AuditEvent } from "../../api/types";
import { useAddNote } from "../../api/campaigns";
import { formatDateTime } from "../../lib/format";
import "./ActivityTab.css";

export function ActivityTab({ campaignId, audit }: { campaignId: string; audit: AuditEvent[] }) {
  const [text, setText] = useState("");
  const addNote = useAddNote(campaignId);

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!text.trim()) return;
    addNote.mutate(text, { onSuccess: () => setText("") });
  }

  return (
    <div className="activity-tab">
      <form className="activity-note-form" onSubmit={submit}>
        <input className="input" placeholder="Add a note…" value={text} onChange={(e) => setText(e.target.value)} />
        <button type="submit" className="btn btn-primary" disabled={!text.trim() || addNote.isPending}>
          Add note
        </button>
      </form>

      <ul className="activity-timeline">
        {audit.map((e) => (
          <li key={e.id} className="activity-item">
            <span className={`activity-dot activity-dot-${e.kind}`} />
            <div className="activity-item-body">
              <div className="activity-item-head">
                <span className="activity-actor">{e.actor}</span>
                <span className="activity-time">{formatDateTime(e.createdAt)}</span>
              </div>
              <div className="activity-action">{e.action}</div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
