import { useEffect, useRef } from "react";

export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || target.isContentEditable;
}

export interface ShortcutHandlers {
  onPalette?: () => void;
  onSearchFocus?: () => void;
  onBack?: () => void;
  onPrev?: () => void;
  onNext?: () => void;
}

// Global shortcuts: Cmd/Ctrl+K toggles the palette; "/" focuses the page's
// own search (or opens the palette when there isn't one); Left/Right step
// through the detail page's prev/next; Backspace goes back. All except
// Cmd/Ctrl+K are ignored while typing in a field.
export function useKeyboardShortcuts(handlers: ShortcutHandlers) {
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      const h = handlersRef.current;
      const typing = isTypingTarget(e.target);

      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        h.onPalette?.();
        return;
      }

      if (typing) return;

      if (e.key === "/") {
        e.preventDefault();
        if (h.onSearchFocus) h.onSearchFocus();
        else h.onPalette?.();
        return;
      }
      if (e.key === "ArrowLeft") {
        h.onPrev?.();
        return;
      }
      if (e.key === "ArrowRight") {
        h.onNext?.();
        return;
      }
      if (e.key === "Backspace") {
        h.onBack?.();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);
}
