interface SkeletonTableProps {
  rows?: number;
}

export default function SkeletonTable({ rows = 5 }: SkeletonTableProps) {
  return (
    <div className="glass-card" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '16px', background: 'rgba(22, 27, 39, 0.4)' }}>
      {Array.from({ length: rows }).map((_, idx) => (
        <div 
          key={idx} 
          style={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center', 
            paddingBottom: '12px', 
            borderBottom: idx === rows - 1 ? 'none' : '1px solid rgba(255, 255, 255, 0.05)' 
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexGrow: 1 }}>
            <div className="skeleton-item" style={{ width: '40px', height: '20px', borderRadius: '4px' }}></div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', width: '50%' }}>
              <div className="skeleton-item" style={{ width: '60%', height: '14px', borderRadius: '4px' }}></div>
              <div className="skeleton-item" style={{ width: '40%', height: '10px', borderRadius: '3px' }}></div>
            </div>
          </div>
          <div className="skeleton-item" style={{ width: '60px', height: '18px', borderRadius: '4px' }}></div>
        </div>
      ))}
    </div>
  );
}
