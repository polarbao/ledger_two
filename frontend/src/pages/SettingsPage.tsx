import { Settings } from 'lucide-react';

export default function SettingsPage() {
  return (
    <div className="page-content animate-fade-in">
      <div className="glass-card header-banner">
        <Settings className="banner-icon" />
        <div>
          <h2>系统设置</h2>
          <p>管理账本的分类、标签和数据导出备份</p>
        </div>
      </div>
      <div className="glass-card placeholder-item full-width">
        <h3>系统设置配置项 (待集成)</h3>
        <p className="dimmed">在此您可以增删分类、标签，或者导出账本为 CSV 等。敬请期待后续任务开发。</p>
      </div>
    </div>
  );
}
