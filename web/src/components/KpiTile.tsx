import { Link } from "react-router-dom";
import { Sparkline } from "./Sparkline";
import "./KpiTile.css";

export function KpiTile({
  label,
  value,
  growth,
  sparkValues,
  to,
  accent = "var(--accent)",
}: {
  label: string;
  value: string;
  growth?: number;
  sparkValues: number[];
  to: string;
  accent?: string;
}) {
  return (
    <Link to={to} className="kpi-tile card card-hover">
      <div className="kpi-tile-label">{label}</div>
      <div className="kpi-tile-value mono">{value}</div>
      <div className="kpi-tile-foot">
        {growth !== undefined && (
          <span className={`kpi-tile-growth ${growth >= 0 ? "kpi-tile-growth-up" : "kpi-tile-growth-down"}`}>
            {growth >= 0 ? "▲" : "▼"} {Math.abs(Math.round(growth))}%
          </span>
        )}
        <span className="kpi-tile-spark">
          <Sparkline values={sparkValues} width={110} height={26} color={accent} />
        </span>
      </div>
    </Link>
  );
}
