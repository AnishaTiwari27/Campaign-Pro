export function SkeletonRows({ count = 5, height = 44 }: { count?: number; height?: number }) {
  return (
    <div className="stack" style={{ gap: 8 }}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="skeleton" style={{ height }} />
      ))}
    </div>
  );
}

export function SkeletonBlock({ height = 220 }: { height?: number }) {
  return <div className="skeleton" style={{ height, width: "100%" }} />;
}
