import type { ReactNode } from "react";
import { NavLink } from "react-router-dom";
import { useSession, useLogout } from "../api/auth";
import { usePendingCount } from "../api/overview";
import {
  IconApprovals,
  IconBenchmarks,
  IconCampaigns,
  IconOverview,
  IconCreators,
  IconRegions,
  IconReports,
  IconLogout,
  IconSettings,
} from "./icons";
import "./Rail.css";

const ROLE_LABEL: Record<string, string> = {
  admin: "Admin",
  approver: "Approver",
  analyst: "Analyst",
  client: "Client",
  viewer: "Viewer",
};

function NavItem({ to, icon, label, badge }: { to: string; icon: ReactNode; label: string; badge?: number }) {
  return (
    <NavLink to={to} className={({ isActive }) => `rail-item${isActive ? " rail-item-active" : ""}`}>
      <span className="rail-item-icon">{icon}</span>
      <span className="rail-item-label">{label}</span>
      {!!badge && <span className="rail-item-badge">{badge}</span>}
    </NavLink>
  );
}

export function Rail() {
  const { data: me } = useSession();
  const logout = useLogout();
  const { data: pendingCount } = usePendingCount();

  return (
    <nav className="rail" aria-label="Primary">
      <div className="rail-brand">
        <span className="rail-brand-mark">CTP</span>
        <span className="rail-brand-text">
          Campaign Tracker Pro
          <span className="rail-brand-sub">India · all regions</span>
        </span>
      </div>

      <div className="rail-group">
        <div className="rail-group-label">Workspace</div>
        <NavItem to="/overview" icon={<IconOverview />} label="Overview" />
        <NavItem to="/campaigns" icon={<IconCampaigns />} label="Campaigns" />
        <NavItem to="/approvals" icon={<IconApprovals />} label="Approvals" badge={pendingCount} />
      </div>

      <div className="rail-group">
        <div className="rail-group-label">Analysis</div>
        <NavItem to="/creators" icon={<IconCreators />} label="Creators" />
        <NavItem to="/regions" icon={<IconRegions />} label="Regions" />
        <NavItem to="/benchmarks" icon={<IconBenchmarks />} label="Benchmarks" />
      </div>

      <div className="rail-group">
        <div className="rail-group-label">Delivery</div>
        <NavItem to="/reports" icon={<IconReports />} label="Reports" />
      </div>

      <div className="rail-spacer" />

      <div className="rail-footer">
        <NavItem to="/settings" icon={<IconSettings />} label="Settings" />
        {me && (
          <div className="rail-user-row">
            <NavLink to="/settings" className="rail-user-chip">
              <span className="rail-user-initials">{me.name.split(" ").map((p) => p[0]).join("").slice(0, 2)}</span>
              <span className="rail-user-info">
                <span className="rail-user-name truncate">{me.name}</span>
                <span className="rail-user-role">
                  {ROLE_LABEL[me.role] ?? me.role}
                  {me.canApprove ? " · approver" : ""}
                </span>
              </span>
            </NavLink>
            <button
              type="button"
              className="rail-logout"
              aria-label="Sign out"
              title="Sign out"
              disabled={logout.isPending}
              onClick={() => logout.mutate()}
            >
              <IconLogout />
            </button>
          </div>
        )}
      </div>
    </nav>
  );
}
