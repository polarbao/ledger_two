import { DollarSign } from 'lucide-react';

export default function SettlementPage() {
  return (
    <div className="page-content animate-fade-in">
      <div className="glass-card header-banner">
        <DollarSign className="banner-icon" />
        <div>
          <h2>结算中心</h2>
          <p>进行双人债务轧差与结算补款</p>
        </div>
      </div>
      <div className="glass-card placeholder-item full-width">
        <h3>两端债务结清面板 (待集成)</h3>
        <p className="dimmed">在此您可以查看谁应向谁付多少钱，并生成结算记录。敬请期待后续任务开发。</p>
      </div>
    </div>
  );
}
