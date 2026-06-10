import { BarChart3 } from 'lucide-react';

export default function AnalyticsPage() {
  return (
    <div className="page-content animate-fade-in">
      <div className="glass-card header-banner">
        <BarChart3 className="banner-icon" />
        <div>
          <h2>分类与标签统计</h2>
          <p>多维视角洞察日常消费比例</p>
        </div>
      </div>
      <div className="glass-card placeholder-item full-width">
        <h3>多维消费图表 (待集成)</h3>
        <p className="dimmed">在此您可以直观地查看当月分类及标签消费的比重。敬请期待后续任务开发。</p>
      </div>
    </div>
  );
}
