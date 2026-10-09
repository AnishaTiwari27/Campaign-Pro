import { useEffect, useState } from "react";
import { useGrants, useSetGrants } from "../../api/users";
import { SkeletonBlock } from "../../components/Skeleton";

/**
 * Which campaigns one client account may see.
 *
 * The set is edited locally and saved in one go, rather than a request per
 * checkbox: granting access is a deliberate act, and a half-applied set
 * after a dropped request is the wrong thing to leave behind.
 */
export function GrantEditor({ userId, userName, onClose }: { userId: string; userName: string; onClose: () => void }) {
  const { data, isLoading } = useGrants(userId);
  const save = useSetGrants();
  const [picked, setPicked] = useState<string[] | null>(null);

  // Seeded once the server's set arrives; after that the local set is the
  // one being edited.
  useEffect(() => {
    if (data && picked === null) setPicked(data.granted);
  }, [data, picked]);

  if (isLoading || !data || picked === null) return <SkeletonBlock height={220} />;

  const toggle = (id: string) =>
    setPicked((cur) => (cur ?? []).includes(id) ? (cur ?? []).filter((x) => x !== id) : [...(cur ?? []), id]);

  const dirty =
    picked.length !== data.granted.length || picked.some((id) => !data.granted.includes(id));

  return (
    <div className="grant-editor">
      <div className="grant-editor-head">
        <div>
          <h4>What {userName} can see</h4>
          <p>
            {picked.length === 0
              ? "Nothing selected — they will see an empty dashboard."
              : `${picked.length} of ${data.campaigns.length} campaigns.`}{" "}
            Everything else is hidden from them everywhere, not just here.
          </p>
        </div>
        <div className="grant-editor-actions">
          <button type="button" className="btn" onClick={() => setPicked(data.granted)} disabled={!dirty}>
            Reset
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!dirty || save.isPending}
            onClick={() => save.mutate({ id: userId, campaignIds: picked }, { onSuccess: onClose })}
          >
            {save.isPending ? "Saving…" : "Save access"}
          </button>
        </div>
      </div>

      <ul className="grant-list">
        {data.campaigns.map((c) => (
          <li key={c.id}>
            <label className="grant-row">
              <input type="checkbox" checked={picked.includes(c.id)} onChange={() => toggle(c.id)} />
              <span className="grant-row-name">{c.name}</span>
              <span className="grant-row-meta">{c.meta}</span>
            </label>
          </li>
        ))}
      </ul>
    </div>
  );
}
