import { useEffect, useState } from "react";

// Re-renders every intervalMs so time-relative text ("Synced 3 min ago")
// stays fresh without each caller managing its own timer.
export function useNow(intervalMs = 30_000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), intervalMs);
    return () => clearInterval(id);
  }, [intervalMs]);
  return now;
}
