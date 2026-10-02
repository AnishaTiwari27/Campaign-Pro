export function Sparkline({
  values,
  width = 120,
  height = 32,
  color = "var(--accent)",
}: {
  values: number[];
  width?: number;
  height?: number;
  color?: string;
}) {
  if (values.length === 0) return null;
  const max = Math.max(...values, 0);
  const min = Math.min(...values, 0);
  const range = max - min || 1;
  const stepX = width / (values.length - 1 || 1);
  const points = values.map((v, i) => {
    const x = i * stepX;
    const y = height - ((v - min) / range) * height;
    return `${x},${y}`;
  });
  const linePath = `M${points.join(" L")}`;
  const areaPath = `${linePath} L${width},${height} L0,${height} Z`;

  // Sized by its container (max `width`) rather than a fixed attribute, so a
  // narrow KPI tile shrinks the chart instead of overflowing the page.
  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      height={height}
      preserveAspectRatio="none"
      style={{ width: "100%", maxWidth: width, display: "block" }}
      role="img"
      aria-label="Trend sparkline"
    >
      <path d={areaPath} fill={color} opacity={0.14} />
      <path d={linePath} fill="none" stroke={color} strokeWidth={1.75} strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
