interface SkeletonCardProps {
  height?: string;
  count?: number;
}

export default function SkeletonCard({ height = '140px', count = 1 }: SkeletonCardProps) {
  return (
    <>
      {Array.from({ length: count }).map((_, idx) => (
        <div 
          key={idx} 
          className="glass-card skeleton-block" 
          style={{ 
            height: height, 
            padding: '20px', 
            display: 'flex', 
            flexDirection: 'column', 
            gap: '14px', 
            justifyContent: 'space-between',
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div className="skeleton-item" style={{ width: '40%', height: '18px', borderRadius: '4px' }}></div>
            <div className="skeleton-item" style={{ width: '18px', height: '18px', borderRadius: '50%' }}></div>
          </div>
          <div className="skeleton-item" style={{ width: '60%', height: '32px', borderRadius: '6px' }}></div>
          <div className="skeleton-item" style={{ width: '80%', height: '14px', borderRadius: '4px' }}></div>
        </div>
      ))}
    </>
  );
}
