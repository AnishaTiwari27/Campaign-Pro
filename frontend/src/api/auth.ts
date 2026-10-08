import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, post } from "./client";
import { queryKeys } from "./queryKeys";
import type { Me } from "./types";

// The session cookie is httpOnly, so the client can't read it. /me is how
// the app discovers whether it has a session at all.
export function useSession() {
  return useQuery({
    queryKey: queryKeys.me(),
    queryFn: () => get<Me>("/me"),
    retry: false, // a 401 is an answer, not a failure to retry
    staleTime: 5 * 60_000,
    // Stated explicitly rather than inherited: re-checking the session on
    // tab focus is what blanked the login form.
    refetchOnWindowFocus: false,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { email: string; password: string }) => post<Me>("/auth/login", v),
    onSuccess: (me) => {
      qc.setQueryData(queryKeys.me(), me);
      // Anything fetched before sign-in was fetched as nobody.
      qc.invalidateQueries();
    },
  });
}

/** What the sign-in screen is allowed to offer. Public, so it resolves
 *  before anyone has a session. */
export type AuthOptions = { signupEnabled: boolean; minPasswordLength: number };

// Asked once per load. If it fails, the screen falls back to sign-in only:
// better to hide a working signup link than to show one that 404s.
export function useAuthOptions() {
  return useQuery({
    queryKey: queryKeys.authOptions(),
    queryFn: () => get<AuthOptions>("/auth/options"),
    retry: false,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
}

/** Registers an account. Deliberately returns no session — the new user
 *  signs in next, which proves the password works. */
export function useSignup() {
  return useMutation({
    mutationFn: (v: { name: string; email: string; password: string }) =>
      post<Me>("/auth/signup", v),
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => post<{ ok: boolean }>("/auth/logout"),
    onSettled: () => {
      // Clearing the query cache alone is not enough — mounted observers
      // can keep rendering the previous user's data. A full reload is both
      // more reliable and more correct: on a shared machine, no in-memory
      // state from the previous session should survive sign-out.
      qc.clear();
      window.location.assign("/");
    },
  });
}
