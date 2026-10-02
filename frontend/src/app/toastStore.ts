import { create } from "zustand";

// Only two icon colors per spec: green for a positive outcome, red for a
// negative one — reject/pause use red even on success, since they're
// negative actions; actual API failures also use red.
export type ToastKind = "positive" | "negative";

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

interface ToastState {
  toasts: Toast[];
  push: (kind: ToastKind, message: string) => void;
  dismiss: (id: number) => void;
}

let nextId = 1;

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],
  push: (kind, message) => {
    const id = nextId++;
    set((s) => ({ toasts: [...s.toasts, { id, kind, message }] }));
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
    }, 4000);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));

export function toastPositive(message: string) {
  useToastStore.getState().push("positive", message);
}
export function toastNegative(message: string) {
  useToastStore.getState().push("negative", message);
}
