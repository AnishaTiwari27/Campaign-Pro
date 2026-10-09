import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, patch, post } from "./client";
import { queryKeys } from "./queryKeys";
import type { DirectoryUser } from "./types";

type UsersResponse = { items: DirectoryUser[]; roles: string[] };

/** The roster. Admin-only; the server refuses anyone else, so a non-admin
 *  reaching this hook gets a 403 rather than an empty list. */
export function useUsers(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.users(),
    queryFn: () => get<UsersResponse>("/users"),
    enabled,
  });
}

/** Creates an account and returns its generated password, which is the
 *  only time that password exists anywhere readable. */
export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { name: string; email: string; role: string; isAgency: boolean }) =>
      post<{ user: DirectoryUser; password: string }>("/users", v),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.users() }),
  });
}

export function useSetUserRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { id: string; role: string; isAgency: boolean }) =>
      patch<{ ok: boolean }>(`/users/${v.id}`, { role: v.role, isAgency: v.isAgency }),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.users() }),
  });
}

/** Issues a new password and returns it once. This is the whole of
 *  password recovery here: the only mailer implemented writes to the log,
 *  and a reset link in a log file is worse than no reset at all. */
export function useResetPassword() {
  return useMutation({
    mutationFn: (id: string) => post<{ password: string }>(`/users/${id}/reset-password`, {}),
  });
}
