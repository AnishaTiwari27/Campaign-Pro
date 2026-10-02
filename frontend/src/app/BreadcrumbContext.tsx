import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

export interface Crumb {
  label: string;
  href?: string;
}

interface BreadcrumbContextValue {
  breadcrumbs: Crumb[];
  setBreadcrumbs: (b: Crumb[]) => void;
}

const BreadcrumbContext = createContext<BreadcrumbContextValue | null>(null);

export function BreadcrumbProvider({ children }: { children: ReactNode }) {
  const [breadcrumbs, setBreadcrumbs] = useState<Crumb[]>([]);
  return <BreadcrumbContext.Provider value={{ breadcrumbs, setBreadcrumbs }}>{children}</BreadcrumbContext.Provider>;
}

export function useBreadcrumbs() {
  const ctx = useContext(BreadcrumbContext);
  if (!ctx) throw new Error("useBreadcrumbs must be used within BreadcrumbProvider");
  return ctx;
}

// Pages call this to declare the command bar's breadcrumb trail for as long
// as they're mounted.
export function useSetBreadcrumbs(breadcrumbs: Crumb[]) {
  const { setBreadcrumbs } = useBreadcrumbs();
  const key = JSON.stringify(breadcrumbs);
  useEffect(() => {
    setBreadcrumbs(breadcrumbs);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
}
