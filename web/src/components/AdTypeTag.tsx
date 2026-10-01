import type { CSSProperties } from "react";
import type { AdType } from "../api/types";

const AD_TYPE_VAR: Record<AdType, string> = {
  Social: "--ad-social",
  Influencer: "--ad-influencer",
  "Google Ads": "--ad-google-ads",
  Display: "--ad-display",
  Video: "--ad-video",
  Performance: "--ad-performance",
};

export function adTypeColorVar(adType: AdType): string {
  return `var(${AD_TYPE_VAR[adType]})`;
}

export function adTypeStripeStyle(adType: AdType): CSSProperties {
  return { "--stripe": adTypeColorVar(adType) } as CSSProperties;
}

export function AdTypeTag({ adType }: { adType: AdType }) {
  return (
    <span className="ad-tag" style={adTypeStripeStyle(adType)}>
      {adType}
    </span>
  );
}
