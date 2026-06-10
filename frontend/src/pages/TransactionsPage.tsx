import { useUIStore } from '../stores/ui.store';
import { ReceiptText } from 'lucide-react';

export default function TransactionsPage() {
  const currentMonth = useUIStore((state) => state.currentMonth);

  return (
    <div className="page-content animate-fade-in">
      <div className="glass-card header-banner">
        <ReceiptText className="banner-icon" />
        <div>
          <h2>交易明细流水</h2>
          <p>当前月份：{currentMonth}</p>
        </div>
      </div>
      <div className="glass-card placeholder-item full-width">
        <h3>交易流水列表 (待集成)</h3>
        <p className="dimmed">在此您可以查看和筛选普通支出、收入及共同支出流水。敬请期待后续任务开发。</p>
      </div>
    </div>
  );
}
