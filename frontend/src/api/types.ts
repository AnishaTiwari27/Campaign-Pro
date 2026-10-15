// Mirrors api/internal/http/dto.go field-for-field.

export type SubjectType = "brand" | "person";
export type AdType = "Social" | "Influencer" | "Google Ads" | "Display" | "Video" | "Performance";
export type CampaignStatus = "live" | "ended" | "scheduled" | "paused";
export type Approval = "pending" | "approved" | "rejected";
export type CurveShape = "fast" | "steady" | "slow";
export type PaceClass = "over" | "warn" | "good";

/** Where a campaign lands if it keeps following its own curve. */
export interface Projection {
  /** How much of the flight has run, 0–1 — the confidence signal. */
  elapsedShare: number;
  projectedReach: number;
  projectedSpend: number;
  /** Flight day the budget runs out; absent when it stays within. */
  exhaustsOnDay?: number;
  overspendBy?: number;
}

/** Where a campaign lands if it keeps following its own curve. */
export interface Projection {
  /** How much of the flight has run, 0-1 — the confidence signal. */
  elapsedShare: number;
  projectedReach: number;
  projectedSpend: number;
  /** Flight day the budget runs out; absent when it stays within. */
  exhaustsOnDay?: number;
  overspendBy?: number;
}

export interface Campaign {
  id: string;
  name: string;
  subjectType: SubjectType;
  role?: string;
  initials: string;
  category: string;
  region: string;
  adType: AdType;
  platform: string;
  status: CampaignStatus;
  daysRunning: number;
  reach: number;
  spend: number;
  budget: number;
  frequency: number;
  approval: Approval;
  curveShape: CurveShape;
  flagReason?: string;
  /** Resolves the brand's logo in /logos; absent for people and unknown brands. */
  brandDomain?: string;
  /** Set for person-subject campaigns; keys the creator's visual identity. */
  creatorId?: string;
  createdAt: string;
  updatedAt: string;
  pace: number;
  paceClass: PaceClass;
  cpm: number;
  index: number;
  categoryIndex: number;
  /** Planned flight length in days. 0 when nobody recorded one. */
  flightDays: number;
  /** Budget share the plan implies by now, as a percentage. */
  expectedPace: number;
  /** pace / expectedPace. 1.0 is exactly on plan, 1.4 is 40% ahead of the
   *  flight. 0 when the campaign has no recorded plan. */
  paceVsPlan: number;
  /** Absent when too early in the flight, or already finished. */
  projection?: Projection;
  /** The campaign's headline creative, so a card can show the
   *  advertisement rather than an avatar on a gradient. Absent until a
   *  campaign has an asset. */
  lead?: LeadCreative;
}

export interface LeadCreative {
  headline: string;
  kind: "video" | "banner" | "text";
  duration: string;
  festival?: string;
}

export interface Creative {
  id: string;
  headline: string;
  kind: "video" | "banner" | "text";
  durationLabel: string;
  reach: number;
  ctr: number;
  /** The CreativeAnalyzer's output. Absent until an asset is analysed. */
  language?: string;
  hookType?: string;
  claim?: string;
  /** The festival or tentpole this creative was cut for, when it was cut
   *  for one at all. Evergreen work has none. */
  festival?: string;
}

export interface AuditEvent {
  id: number;
  actor: string;
  action: string;
  kind: "sys" | "alert" | "user";
  createdAt: string;
}

export interface CampaignDetail extends Campaign {
  creatives: Creative[];
  audit: AuditEvent[];
  similar: Campaign[];
  position: number;
  total: number;
  prevId?: string;
  nextId?: string;
  medAll: number;
}

export interface ListResponse {
  items: Campaign[];
  total: number;
  page: number;
  pages: number;
}

export interface ListParams {
  q?: string;
  type?: string;
  category?: string;
  region?: string;
  adType?: string;
  status?: string;
  approval?: string;
  range?: string;
  sort?: string;
  dir?: string;
  page?: number;
  per?: number;
}

export interface SparklinePoint {
  DaysAgo: number;
  Reach: number;
  Spend: number;
  LiveCount: number;
}

export interface Sparkline {
  points: SparklinePoint[];
  growth: number;
}

export interface Overview {
  pendingCount: number;
  flaggedCount: number;
  pendingSpend: number;
  liveCount: number;
  liveSparkline: Sparkline;
  pendingSparkline: Sparkline;
  reachLive: number;
  reachSparkline: Sparkline;
  spendWindow: number;
  spendSparkline: Sparkline;
  approvedBudget: number;
  spendPctBudget: number;
  medAll: number;
  spotlight?: Campaign;
  /** Ranked attention list the Overview cycles through. */
  spotlightCandidates: Campaign[];
  needsDecision: Campaign[];
  flagged: Campaign[];
  movers: Campaign[];
  people: Campaign[];
}

export interface CategoryBenchmark {
  category: string;
  n: number;
  medianReach: number;
  topCampaign: string;
  topReach: number;
  totalSpend: number;
}

export interface Benchmark {
  medAll: number;
  categories: CategoryBenchmark[];
  festivals: FestivalStat[];
}

/** Analysed-creative performance grouped by the festival a creative was
 *  cut for. An empty festival is evergreen work — the baseline, which is
 *  why it carries no lift of its own. */
export interface FestivalStat {
  festival: string;
  creatives: number;
  avgReach: number;
  avgCtr: number;
  /** avgReach against evergreen work. 1.0 means a festival cut performed
   *  exactly like untagged work. 0 on the baseline row. */
  lift: number;
}

export interface AdTypeCount {
  adType: AdType;
  count: number;
}

export interface Region {
  region: string;
  count: number;
  liveCount: number;
  totalReach: number;
  totalSpend: number;
  adTypeSplit: AdTypeCount[];
}

export interface CategoryCount {
  category: string;
  count: number;
  reach: number;
}

export interface RegionDetail extends Region {
  pendingCount: number;
  shareOfBudget: number;
  campaigns: Campaign[];
  categoryBreakdown: CategoryCount[];
}

export type Cadence = "weekly_mon_9" | "weekday_830" | "monthly_1_9" | "on_flag";

export interface Report {
  id: string;
  name: string;
  enabled: boolean;
  cadence: Cadence;
  cadenceLabel: string;
  recipients: string[];
  scopeFilters: Record<string, unknown>;
  scopeLabel: string;
  columns: string[];
  createdAt: string;
  updatedAt: string;
  lastRun?: ReportRun;
}

export interface ReportRun {
  ranAt: string;
  result: "Delivered" | "Failed" | "Not sending";
  rowCount: number;
}

export interface SearchResult {
  kind: "section" | "campaign" | "creator" | "region" | "action";
  id: string;
  title: string;
  subtitle?: string;
}

export interface Me {
  id: string;
  name: string;
  email: string;
  role: "admin" | "approver" | "analyst" | "client" | "viewer";
  /** Agency staff see every account; clients see only their own. */
  isAgency: boolean;
  canApprove: boolean;
  isClient: boolean;
  /** Derived server-side and sent, so the client never re-derives a
   *  permission rule and gets it subtly different from the enforcer. */
  canManageUsers: boolean;
}

/** One account on the user-management screen. */
export interface DirectoryUser {
  id: string;
  email: string;
  name: string;
  role: string;
  isAgency: boolean;
  canApprove: boolean;
  createdAt: string;
  /** True for the signed-in admin's own row: the controls they must not
   *  use on themselves are disabled rather than hidden. */
  isSelf: boolean;
}

export interface HealthSource {
  name: string;
  status: string;
}

export interface Health {
  status: string;
  sources: HealthSource[];
}

export const REPORT_COLUMNS = [
  "Subject",
  "Subject type",
  "Category",
  "Region",
  "Ad type",
  "Platform",
  "Approval",
  "Reach",
  "Spend",
  "Budget pace",
  "CPM",
  "Waiting since",
  "Flag reason",
  "Category median",
  "Index",
] as const;

export const CADENCE_OPTIONS: { value: Cadence; label: string }[] = [
  { value: "weekly_mon_9", label: "Every Monday 9:00 AM IST" },
  { value: "weekday_830", label: "Every weekday 8:30 AM IST" },
  { value: "monthly_1_9", label: "First of the month 9:00 AM IST" },
  { value: "on_flag", label: "Within 15 min of a flag" },
];

export type Tier = "nano" | "micro" | "mid" | "macro" | "mega";

export interface Creator {
  id: string;
  name: string;
  role: string;
  initials: string;
  category: string;
  region: string;
  tier: Tier;
  tierLabel: string;
  followers: number;
  primaryPlatform: string;
  languages: string[];
  /** The creator's own photograph, where the agency has one it may
   *  publish. Absent is normal, and draws the generated avatar — which is
   *  the right likeness for someone whose photo cannot be used, not a
   *  placeholder for a missing one. */
  imageUrl?: string;
}

export interface CreatorPerformance extends Creator {
  campaigns: number;
  liveCampaigns: number;
  totalReach: number;
  totalSpend: number;
  avgReach: number;
  /** Median campaign reach across this creator's tier — their baseline. */
  tierMedian: number;
  /** avgReach / tierMedian. 1.0 is exactly typical for their size. */
  tierIndex: number;
  costPerLakh: number;
  /** 0-1; how repeatable their delivery is across campaigns. */
  consistency: number;
  /** Campaigns the score came from. Below 2 it isn't measurable. */
  consistencyN: number;
  /** Avg reach as a share of follower base. >100% means content travelled. */
  audienceReachPct: number;
  flagged: number;
  /** Median cost per lakh across this creator's tier. */
  tierCostMedian: number;
  /** costPerLakh / tierCostMedian. LOWER IS BETTER — 0.8x is cheaper
   *  reach than peers of their size. 0 means not yet measurable. */
  costIndex: number;
  /** How many priced creators the tier median came from. 1 means they are
   *  alone in their tier, so costIndex is 0 and nothing is shown. */
  tierCostPeers: number;
}

export interface LanguageReach {
  language: string;
  reach: number;
  count: number;
}

export interface CreatorDetail extends CreatorPerformance {
  campaignRows: Campaign[];
  languageBreakdown: LanguageReach[];
  tierPeers: number;
}

/** A freely-licensed photograph, with everything needed to credit it. */
export interface EndorsementPhoto {
  url: string;
  artist: string;
  license: string;
  licenseUrl: string;
  source: string;
}

export interface EndorsementDeal {
  brand: string;
  period?: string;
  note?: string;
  /** endorsement | ambassador | stake — an ownership stake is not an ad. */
  kind: "endorsement" | "ambassador" | "stake";
}

/**
 * A real public figure and the brand work reported about them.
 *
 * Carries no reach, spend, pace or approval, and never will: those exist
 * only in the agency's own fixture data, and attaching an invented figure
 * to a named real person would fabricate a commercial relationship.
 */
export interface EndorsementPerson {
  id: string;
  name: string;
  role: string;
  initials: string;
  field: string;
  photo: EndorsementPhoto;
  deals: EndorsementDeal[];
  source: string;
}

/** One category's standing on one day, from the benchmark history. */
export interface BenchmarkTrendPoint {
  category: string;
  /** A plain YYYY-MM-DD date: the grain is a day, not an instant. */
  day: string;
  medianReach: number;
  topReach: number;
  campaignCount: number;
  medianCostPerLakh: number;
}

export interface InsightFact {
  label: string;
  display: string;
}

/** A written diagnosis of why a campaign is flagged. */
export interface Insight {
  campaignId: string;
  text: string;
  /** "model" when written, "fallback" when assembled from the figures. */
  source: "model" | "fallback";
  /** The figures the text was allowed to draw on. */
  facts: InsightFact[];
}

export type LaunchReadiness = "soon" | "building" | "quiet" | "unclear";

/** A product or brand the radar has seen launch. */
export interface Launch {
  id: string;
  source: string;
  name: string;
  tagline: string;
  url: string;
  category: string;
  launchedAt: string;
  traction: number;
  hasSite: boolean;
  readiness: LaunchReadiness;
  /** The signals the band was computed from, so it can be argued with. */
  basis: string;
  firstCampaignAt?: string;
  daysToMarket?: number;
}
