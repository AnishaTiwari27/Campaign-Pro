import { useSearchParams } from "react-router-dom";
import type { ListParams } from "../api/types";

const DEFAULT_SORT = "reach";
const DEFAULT_DIR = "desc";
const DEFAULT_PAGE = 1;
const DEFAULT_PER = 10;

// Reads/writes every Campaigns filter, sort, and page from the URL's query
// string, so any view is shareable and survives a refresh.
export function useListParams(): [ListParams, (patch: Partial<ListParams>) => void] {
  const [searchParams, setSearchParams] = useSearchParams();

  const params: ListParams = {
    q: searchParams.get("q") ?? undefined,
    type: searchParams.get("type") ?? undefined,
    category: searchParams.get("category") ?? undefined,
    region: searchParams.get("region") ?? undefined,
    adType: searchParams.get("adType") ?? undefined,
    status: searchParams.get("status") ?? undefined,
    approval: searchParams.get("approval") ?? undefined,
    range: searchParams.get("range") ?? undefined,
    sort: searchParams.get("sort") ?? DEFAULT_SORT,
    dir: searchParams.get("dir") ?? DEFAULT_DIR,
    page: Number(searchParams.get("page") ?? DEFAULT_PAGE),
    per: Number(searchParams.get("per") ?? DEFAULT_PER),
  };

  function update(patch: Partial<ListParams>) {
    const next = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(patch)) {
      if (value === undefined || value === "") next.delete(key);
      else next.set(key, String(value));
    }
    // Any change other than an explicit page navigation resets to page 1.
    if (!("page" in patch)) next.delete("page");
    setSearchParams(next, { replace: true });
  }

  return [params, update];
}
