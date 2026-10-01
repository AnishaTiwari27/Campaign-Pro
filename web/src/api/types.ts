// Mirrors api/internal/http/dto.go field-for-field.

export type SubjectType = "brand" | "person";
export type AdType = "Social" | "Influencer" | "Google Ads" | "Display" | "Video" | "Performance";
export type CampaignStatus = "live" | "ended" | "scheduled" | "paused";
export type Approval = "pending" | "approved" | "rejected";
export type CurveShape = "fast" | "steady" | "slow";
export type PaceClass = "over" | "warn" | "good";

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
  createdAt: string;
  updatedAt: string;
  pace: number;
  paceClass: PaceClass;
  cpm: number;
  index: number;
  categoryIndex: number;
}

export interface Creative {
  id: string;
  headline: string;
  kind: "video" | "banner" | "text";
  durationLabel: string;
  reach: number;
  ctr: number;
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
  kind: "section" | "campaign" | "region" | "action";
  id: string;
  title: string;
  subtitle?: string;
}

export interface Me {
  id: string;
  name: string;
  email: string;
  role: string;
  canApprove: boolean;
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
