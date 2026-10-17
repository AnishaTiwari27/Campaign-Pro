import type { ReactNode } from "react";
import { NavLink } from "react-router-dom";
import { useSession } from "../api/auth";
import { usePendingCount } from "../api/overview";
import {
  IconApprovals,
  IconBenchmarks,
  IconSignals,
  IconCampaigns,
  IconOverview,
  IconCreators,
  IconEndorsements,
  IconRadar,
  IconRegions,
  IconReports,
} from "./icons";
import { AccountMenu } from "./AccountMenu";
import "./Rail.css";

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
  const { data: pendingCount } = usePendingCount();
  // An external client is not agency staff. The approval queue is the
  // agency deciding its own spend, and benchmarks compare accounts across
  // the whole book of business — neither is theirs to read. An analyst is
  // agency staff and does still see both; they simply cannot act on them.
  const isClient = me?.isClient ?? false;
  const canManageUsers = me?.canManageUsers ?? false;

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
        {!isClient && (
          <NavItem to="/approvals" icon={<IconApprovals />} label="Approvals" badge={pendingCount} />
        )}
      </div>

      <div className="rail-group">
        <div className="rail-group-label">Analysis</div>
        {/* Label and route deliberately differ. The section is named for
            what it holds — influencers, actors, cricketers, singers — while
            the route, the API path, the Go package and the table stay
            "creators". Renaming those would break every link already shared
            and buy nothing a reader can see. */}
        <NavItem to="/creators" icon={<IconCreators />} label="Influencers" />
        {/* Public figures and their reported brand work. Sits apart from
            Influencers on purpose: that page is the agency's roster with
            its performance data, this one is published fact with none. */}
        {/* Prospecting data — who to approach before they start
            spending — so it is agency-only like Benchmarks. */}
        {!isClient && <NavItem to="/radar" icon={<IconRadar />} label="Launch radar" />}
        <NavItem to="/endorsements" icon={<IconEndorsements />} label="Public record" />
        <NavItem to="/regions" icon={<IconRegions />} label="Regions" />
        <NavItem to="/signals" icon={<IconSignals />} label="Signals" />
        {!isClient && <NavItem to="/benchmarks" icon={<IconBenchmarks />} label="Benchmarks" />}
      </div>

      <div className="rail-group">
        <div className="rail-group-label">Delivery</div>
        <NavItem to="/reports" icon={<IconReports />} label="Reports" />
      </div>

      <div className="rail-spacer" />

      <div className="rail-footer">
        {/* Team, Billing and Settings used to sit here as three more
            rows. They are about the account rather than the data, and
            they are in the account menu now — listing them in both
            places is two routes to one screen and a longer rail for
            nothing. */}
        <AccountMenu canManageUsers={canManageUsers} />
      </div>
    </nav>
  );
}
