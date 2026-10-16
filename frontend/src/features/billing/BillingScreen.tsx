import { useBilling, useCheckout, useStartTrial, formatPrice, type Plan } from "../../api/billing";
import { SkeletonBlock } from "../../components/Skeleton";
import { ApiError } from "../../api/client";
import "./BillingScreen.css";

/** The plans, the current standing, and the way to change it.
 *
 *  One component for two jobs: the page you open to look at your
 *  subscription, and the wall you hit when it has run out. They differ
 *  by a banner and whether the rest of the app is behind them, which
 *  is not enough to justify two screens that would drift apart.
 */
export function BillingScreen({ locked = false, reason }: { locked?: boolean; reason?: string }) {
  const { data, isLoading } = useBilling();
  const trial = useStartTrial();
  const checkout = useCheckout();

  if (isLoading || !data) return <SkeletonBlock height={420} />;

  const neverSubscribed = data.status === "none";
  const checkoutError =
    checkout.error instanceof ApiError ? checkout.error.message : null;

  return (
    <div className={`billing${locked ? " billing-locked" : ""}`}>
      <header className="billing-head">
        <h1>{locked ? "Your access has paused" : "Subscription"}</h1>
        <p className="billing-lede">
          {locked
            ? (reason ?? "This organisation does not have an active subscription.")
            : "What this organisation is on, and what else there is."}
        </p>
      </header>

      <div className="billing-status card">
        <div>
          <span className="billing-status-label">Status</span>
          <span className={`billing-status-value billing-status-${data.status}`}>
            {statusLabel(data.status)}
          </span>
        </div>
        <div>
          <span className="billing-status-label">Plan</span>
          <span className="billing-status-value">{planLabel(data)}</span>
        </div>
        <div>
          <span className="billing-status-label">{remainingLabel(data)}</span>
          <span className="billing-status-value">{remainingValue(data)}</span>
        </div>
      </div>

      {neverSubscribed && (
        <div className="billing-trial card">
          <div>
            <strong>Start with 14 days free</strong>
            <p>No card, no charge. It covers everything in Starter.</p>
          </div>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => trial.mutate()}
            disabled={trial.isPending}
          >
            {trial.isPending ? "Starting…" : "Start the trial"}
          </button>
        </div>
      )}

      {/* Said plainly rather than hidden behind a button that fails:
          this deployment has no payment keys, so nothing can be
          bought here yet. */}
      {!data.payable && (
        <p className="billing-notice" role="status">
          Payments are not configured on this deployment, so the plans below cannot be
          bought yet. The trial still works.
        </p>
      )}

      {checkoutError && (
        <p className="billing-error" role="alert">
          {checkoutError}
        </p>
      )}

      <div className="billing-plans">
        {data.plans.map((plan) => (
          <PlanCard
            key={plan.id}
            plan={plan}
            current={data.plan === plan.id && data.status === "active"}
            disabled={!data.payable || checkout.isPending}
            onChoose={() => checkout.mutate(plan.id)}
          />
        ))}
      </div>
    </div>
  );
}

function PlanCard({
  plan,
  current,
  disabled,
  onChoose,
}: {
  plan: Plan;
  current: boolean;
  disabled: boolean;
  onChoose: () => void;
}) {
  return (
    <div className={`billing-plan card${current ? " billing-plan-current" : ""}`}>
      <div className="billing-plan-head">
        <h2>{plan.name}</h2>
        {current && <span className="billing-plan-badge">Current</span>}
      </div>
      <div className="billing-plan-price">
        <strong>{formatPrice(plan.amountPaise)}</strong>
        <span>/{plan.interval === "monthly" ? "month" : plan.interval}</span>
      </div>
      <p className="billing-plan-blurb">{plan.blurb}</p>
      <ul className="billing-plan-includes">
        {plan.includes.map((line) => (
          <li key={line}>{line}</li>
        ))}
      </ul>
      <button
        type="button"
        className={`btn ${current ? "" : "btn-primary"} billing-plan-cta`}
        onClick={onChoose}
        disabled={disabled || current}
      >
        {current ? "Your plan" : `Choose ${plan.name}`}
      </button>
    </div>
  );
}

function remainingLabel(data: { status: string; allowed: boolean }): string {
  if (!data.allowed) return "Access";
  return data.status === "trialing" ? "Trial ends" : "Renews";
}

/** What is left, in words.
 *
 *  The awkward case is a cancelled subscription with no end date: the
 *  server reports -1 days, meaning "no expiry", which read literally
 *  would put "Does not expire" next to "Cancelled". Access is the
 *  thing being described, so a subscription that is not allowing
 *  anything has ended regardless of what its dates say.
 */
function remainingValue(data: { allowed: boolean; daysLeft: number }): string {
  if (!data.allowed) return "Ended";
  if (data.daysLeft < 0) return "Does not expire";
  if (data.daysLeft === 0) return "Ends today";
  return `${data.daysLeft} day${data.daysLeft === 1 ? "" : "s"}`;
}

function statusLabel(status: string): string {
  switch (status) {
    case "trialing":
      return "On trial";
    case "active":
      return "Active";
    case "past_due":
      return "Payment failed";
    case "cancelled":
      return "Cancelled";
    default:
      return "No subscription";
  }
}

function planLabel(data: { plan: string; status: string }): string {
  if (data.plan === "grandfathered") return "Early access";
  if (!data.plan || data.status === "none") return "—";
  return data.plan.charAt(0).toUpperCase() + data.plan.slice(1);
}
