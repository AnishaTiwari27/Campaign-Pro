// The pace/index quadrant: where each campaign sits when delivery is read
// against spend. Both inputs already exist on every campaign — `pace` is
// spend as a percentage of budget, `index` is reach against the
// all-campaign median — so this file only decides what a pair of them
// *means*, and nothing here recomputes either.

import type { Campaign } from "../api/types";

/** Exactly on plan. Above it a campaign is spending ahead of its flight.
 *  This replaced a raw "100% of budget" line, which could not tell a
 *  campaign halfway through its flight from one that was underspending. */
export const PACE_MIDLINE = 1;

/** 1.0x is exactly typical: the all-campaign median reach (MED_ALL). */
export const INDEX_MIDLINE = 1;

export type QuadrantKey = "scale" | "protect" | "fix" | "cut";

export interface Quadrant {
  key: QuadrantKey;
  /** The verb, because the point of this view is the decision. */
  label: string;
  /** What being here actually means, in one line. */
  blurb: string;
  /** Maps to the app's good/warn/crit tokens. */
  tone: "good" | "accent" | "warn" | "crit";
}

export const QUADRANTS: Record<QuadrantKey, Quadrant> = {
  scale: {
    key: "scale",
    label: "Scale",
    blurb: "Beating the median while spending behind its flight. The cheapest reach you can buy is more of this.",
    tone: "good",
  },
  protect: {
    key: "protect",
    label: "Protect",
    blurb: "Beating the median and spending to plan or faster. Working as intended — leave it alone.",
    tone: "accent",
  },
  fix: {
    key: "fix",
    label: "Fix",
    blurb: "Below the median, but behind its flight with budget still to run. Time to change the creative or the targeting.",
    tone: "warn",
  },
  cut: {
    key: "cut",
    label: "Cut",
    blurb: "Below the median and burning ahead of plan. Every further rupee buys less than the fleet average.",
    tone: "crit",
  },
};

/** Fixed order: best to worst, so every list and legend agrees. */
export const QUADRANT_ORDER: QuadrantKey[] = ["scale", "protect", "fix", "cut"];

/**
 * Which quadrant a campaign falls in.
 *
 * Both boundaries are inclusive upward: index exactly 1.0 counts as
 * meeting the median rather than missing it, and pace exactly 100 counts
 * as fully spent. A campaign sitting precisely on the line is the
 * uninteresting case, and reading it the generous way for delivery and
 * the strict way for budget keeps the "Cut" quadrant free of campaigns
 * that have not actually underperformed.
 */
export function quadrantOf(paceVsPlan: number, index: number): QuadrantKey {
  const delivering = index >= INDEX_MIDLINE;
  const spent = paceVsPlan >= PACE_MIDLINE;
  if (delivering) return spent ? "protect" : "scale";
  return spent ? "cut" : "fix";
}

export interface SignalPoint {
  campaign: Campaign;
  /** Pace against plan, the x-axis. 1.0 is exactly on plan. */
  paceVsPlan: number;
  index: number;
  quadrant: QuadrantKey;
}

/**
 * Two exclusions, both because the point would be a lie rather than a
 * measurement. Scheduled campaigns have delivered and spent nothing, so
 * they would all pile into the corner of "Fix". A campaign with no
 * recorded flight length has no plan to be measured against, and placing
 * it would mean inventing one — the page says how many were left out.
 */
export function signalPoints(campaigns: Campaign[]): SignalPoint[] {
  return campaigns
    .filter((c) => c.status !== "scheduled" && c.paceVsPlan > 0)
    .map((c) => ({
      campaign: c,
      paceVsPlan: c.paceVsPlan,
      index: c.index,
      quadrant: quadrantOf(c.paceVsPlan, c.index),
    }));
}

export interface QuadrantSummary extends Quadrant {
  points: SignalPoint[];
  /** Budget sitting in this quadrant — the number that makes it a decision. */
  budget: number;
  spend: number;
}

export function summarise(points: SignalPoint[]): QuadrantSummary[] {
  return QUADRANT_ORDER.map((key) => {
    const mine = points.filter((p) => p.quadrant === key);
    return {
      ...QUADRANTS[key],
      points: mine,
      budget: mine.reduce((sum, p) => sum + p.campaign.budget, 0),
      spend: mine.reduce((sum, p) => sum + p.campaign.spend, 0),
    };
  });
}
