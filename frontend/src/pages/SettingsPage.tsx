import { useState, useEffect } from 'react';
import { 
  Settings, 
  Download, 
  Database, 
  FileSpreadsheet, 
  FileJson, 
  RefreshCw, 
  Clock, 
  HardDrive, 
  AlertTriangle, 
  X 
} from 'lucide-react';
import { api, ApiError } from '../api/client';
import EmptyState from '../components/ui/EmptyState';

interface BackupInfo {
  filename: string;
  size_bytes: number;
  created_at: string;
}

type ModalType = 'backup' | 'csv' | 'json' | null;

export default function SettingsPage() {
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loadingBackups, setLoadingBackups] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [selectedMonth, setSelectedMonth] = useState<string>('');

  // 确认弹窗状态
  const [modalType, setModalType] = useState<ModalType>(null);

  // 加载备份列表
  const fetchBackups = async () => {
    setLoadingBackups(true);
    setErrorMsg(null);
    try {
      const data = await api.get<BackupInfo[]>('/api/admin/backups');
      setBackups(data);
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(`加载备份列表失败: ${err.message}`);
      } else {
        setErrorMsg('加载备份列表失败');
      }
    } finally {
      setLoadingBackups(false);
    }
  };

  useEffect(() => {
    fetchBackups();
  }, []);

  // 执行手动备份
  const handleBackupSubmit = async () => {
    setModalType(null);
    setActionLoading(true);
    setErrorMsg(null);
    setSuccessMsg(null);
    try {
      const res = await api.post<{ filename: string }>('/api/admin/backup');
      setSuccessMsg(`备份创建成功: ${res.filename}`);
      fetchBackups();
    } catch (err: unknown) {
      if (err instanceof ApiError) {
        setErrorMsg(`备份失败: ${err.message}`);
      } else {
        setErrorMsg('备份失败，请检查备份目录写权限');
      }
    } finally {
      setActionLoading(false);
    }
  };

  // 带 Credentials & 错误拦截的物理流下载
  const triggerDownload = async (url: string, defaultFilename: string) => {
    setActionLoading(true);
    setErrorMsg(null);
    setSuccessMsg(null);
    try {
      const res = await fetch(url, { credentials: 'include' });
      if (!res.ok) {
        let errMsg = '下载失败';
        try {
          const errBody = await res.json();
          if (errBody?.error?.message) {
            errMsg = errBody.error.message;
          }
        } catch {
          // Ignore
        }
        throw new Error(errMsg);
      }

      const blob = await res.blob();
      const blobUrl = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = blobUrl;
      a.download = defaultFilename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(blobUrl);
      setSuccessMsg('文件下载成功！');
    } catch (err: any) {
      setErrorMsg(err.message || '文件下载失败，请重试');
    } finally {
      setActionLoading(false);
    }
  };

  // 格式化文件大小
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // 格式化时间
  const formatDate = (dateStr: string) => {
    try {
      const d = new Date(dateStr);
      return d.toLocaleString('zh-CN', { hour12: false });
    } catch {
      return dateStr;
    }
  };

  // 点击导出按钮，打开对应的二次确认弹窗
  const openConfirmModal = (type: ModalType) => {
    setModalType(type);
  };

  return (
    <div className="page-content animate-fade-in text-left">
      {/* 头部 Banner */}
      <div className="glass-card header-banner">
        <Settings className="banner-icon" />
        <div>
          <h2>系统设置</h2>
          <p>管理数据导出备份、数据防丢以及高风险审计记录</p>
        </div>
      </div>

      {/* 消息通知区 */}
      {errorMsg && (
        <div className="error-banner animate-fade-in" style={{ margin: '0 0 16px 0', borderRadius: '12px' }}>
          <AlertTriangle size={18} style={{ marginRight: '8px', flexShrink: 0 }} />
          <span>{errorMsg}</span>
        </div>
      )}
      {successMsg && (
        <div className="glass-card text-green animate-fade-in" style={{ padding: '12px 20px', margin: '0 0 16px 0', borderRadius: '12px', background: 'rgba(16, 185, 129, 0.06)', border: '1px solid rgba(16, 185, 129, 0.2)' }}>
          <span>{successMsg}</span>
        </div>
      )}

      {/* 两栏布局 */}
      <div className="form-row-2">
        {/* 左栏：数据导出 */}
        <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
            <Download size={20} className="partner-highlight" />
            <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>数据导出中心</h3>
          </div>

          <p className="dimmed-desc" style={{ margin: 0 }}>
            您可以随时导出您的账本流水明细及全量配置。为了保护您的个人账务隐私，系统支持基于当前用户身份的可见性隔离导出。
          </p>

          {/* CSV 导出项 */}
          <div style={{ background: 'rgba(255,255,255,0.01)', border: '1px solid rgba(255,255,255,0.03)', padding: '16px', borderRadius: '12px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <FileSpreadsheet className="text-green" size={20} />
              <strong style={{ fontSize: '14px' }}>CSV 交易流水导出</strong>
            </div>
            <p className="dimmed-desc" style={{ fontSize: '12px', margin: 0 }}>
              包含发生时间、标题、分类、金额（元/分）、付款人、可见性、备注等，适合 Excel 人工审计。
            </p>
            <div style={{ display: 'flex', gap: '10px', alignItems: 'center', marginTop: '4px' }}>
              <div style={{ display: 'flex', flexDirection: 'column', flexGrow: 1 }}>
                <input 
                  type="month" 
                  value={selectedMonth}
                  onChange={(e) => setSelectedMonth(e.target.value)}
                  style={{ width: '100%', padding: '8px 12px', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.08)', background: 'rgba(10,12,16,0.6)', color: '#fff' }}
                  placeholder="选择特定月份（可选）"
                />
              </div>
              <button 
                onClick={() => openConfirmModal('csv')}
                className="btn-primary" 
                style={{ padding: '8px 16px', fontSize: '13px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}
                disabled={actionLoading}
              >
                <Download size={14} /> 导出 CSV
              </button>
            </div>
            {selectedMonth && (
              <span className="dimmed-desc" style={{ fontSize: '11px', color: 'var(--accent-green)' }}>
                已选择按月份：{selectedMonth} 导出
              </span>
            )}
          </div>

          {/* JSON 全量导出 */}
          <div style={{ background: 'rgba(255,255,255,0.01)', border: '1px solid rgba(255,255,255,0.03)', padding: '16px', borderRadius: '12px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <FileJson style={{ color: '#60a5fa' }} size={20} />
              <strong style={{ fontSize: '14px' }}>JSON 全量数据包导出</strong>
            </div>
            <p className="dimmed-desc" style={{ fontSize: '12px', margin: 0 }}>
              包含经过完全脱敏的成员账户、分类、标签、交易分摊和结算记录。可用于数据库全量备份迁移。
            </p>
            <button 
              onClick={() => openConfirmModal('json')}
              className="btn-secondary" 
              style={{ width: '100%', padding: '10px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px', fontSize: '13px' }}
              disabled={actionLoading}
            >
              <Download size={14} /> 导出全量 JSON 数据包
            </button>
          </div>
        </div>

        {/* 右栏：数据库物理备份与管理 */}
        <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <Database size={20} className="partner-highlight" />
              <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>SQLite 物理安全备份</h3>
            </div>
            <button 
              onClick={fetchBackups} 
              className="btn-close-drawer" 
              style={{ padding: '6px' }}
              title="刷新备份列表"
              disabled={loadingBackups}
            >
              <RefreshCw size={16} className={loadingBackups ? 'animate-spin' : ''} />
            </button>
          </div>

          <p className="dimmed-desc" style={{ margin: 0 }}>
            利用 SQLite 在线事务安全备份机制（VACUUM INTO），零锁死保障在写事务期间生成高度一致的物理数据库镜像文件。
          </p>

          <button 
            onClick={() => openConfirmModal('backup')}
            className="btn-primary" 
            style={{ padding: '12px', borderRadius: '10px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', fontSize: '14px', fontWeight: 600, boxShadow: '0 4px 12px rgba(168,85,247,0.15)' }}
            disabled={actionLoading}
          >
            <Database size={16} /> 立即创建手动安全备份
          </button>

          {/* 备份列表 */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <strong style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>历史手动备份文件</strong>
            
            {loadingBackups && backups.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '24px', color: 'var(--text-muted)' }}>
                <RefreshCw size={18} className="animate-spin" style={{ margin: '0 auto 8px' }} />
                <span>扫描备份文件中...</span>
              </div>
            ) : backups.length === 0 ? (
              <EmptyState 
                title="暂无手动备份"
                description="系统暂未生成备份文件。建议在日常正式记账前，先创建一次手动备份以确保安全。"
              />
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '240px', overflowY: 'auto', paddingRight: '4px' }}>
                {backups.map((b) => (
                  <div key={b.filename} style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.03)', borderRadius: '10px', padding: '10px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', transition: 'all 0.2s' }}>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', textAlign: 'left' }}>
                      <span style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', wordBreak: 'break-all' }}>
                        {b.filename.replace('manual/', '')}
                      </span>
                      <div style={{ display: 'flex', gap: '12px', fontSize: '11px', color: 'var(--text-muted)' }}>
                        <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                          <HardDrive size={12} /> {formatBytes(b.size_bytes)}
                        </span>
                        <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                          <Clock size={12} /> {formatDate(b.created_at)}
                        </span>
                      </div>
                    </div>
                    <button 
                      onClick={() => triggerDownload(`/api/admin/backups/${encodeURIComponent(b.filename)}`, b.filename.split('/').pop() || 'backup.db')}
                      className="btn-secondary" 
                      style={{ padding: '6px 12px', fontSize: '12px', borderRadius: '6px', display: 'flex', alignItems: 'center', gap: '4px', flexShrink: 0 }}
                      disabled={actionLoading}
                    >
                      <Download size={12} /> 下载
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ==========================================
         UI 安全防范二次确认弹窗 Modal (Danger 警示样式按钮)
         ========================================== */}
      {modalType && (
        <div className="drawer-overlay show" style={{ alignItems: 'center', justifyContent: 'center' }}>
          <div className="confirm-modal-box animate-fade-in">
            <div className="drawer-header" style={{ padding: '16px 20px' }}>
              <div className="header-title" style={{ color: '#ef4444' }}>
                <AlertTriangle className="title-icon" style={{ color: 'inherit' }} />
                <h3 style={{ fontSize: '16px' }}>
                  {modalType === 'backup' && '立即创建手动备份？'}
                  {modalType === 'csv' && '确认导出交易流水 CSV？'}
                  {modalType === 'json' && '确认导出 JSON 数据包？'}
                </h3>
              </div>
              <button className="btn-close-drawer" onClick={() => setModalType(null)}>
                <X size={18} />
              </button>
            </div>

            <div className="modal-body-padding" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <p className="modal-alert-text">
                {modalType === 'backup' && '系统将基于当前真实的 SQLite 数据库，生成一个只读备份文件并存放在 NAS manual 目录下。该文件包含完整的物理账目，请确保 NAS 硬盘容量充足。'}
                {modalType === 'csv' && `即将导出 ${selectedMonth ? selectedMonth + ' 月份的' : '全量'} 账单流水文件。导出的 CSV 文件包含明文账目信息，为了您的信息安全，请妥善保管所下载的明文表格。`}
                {modalType === 'json' && '即将导出全量 JSON 账目归档。导出的 JSON 数据包中已经去除了所有用户的登录密码 Hash 等系统敏感密钥凭证。该文件包含核心财务流水，切勿随意发送给外部他人。'}
              </p>

              <div style={{ background: 'rgba(239, 68, 68, 0.04)', border: '1px solid rgba(239, 68, 68, 0.15)', borderRadius: '8px', padding: '10px 14px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#fca5a5', textAlign: 'left' }}>
                <AlertTriangle size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
                <span>此操作作为高风险数据变动动作，将被自动记录并同步写入系统的 `audit_logs` 审计表中以备历史追溯。</span>
              </div>

              <div className="drawer-footer" style={{ borderTop: 'none', paddingTop: 0, marginTop: '8px', display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
                <button className="btn-secondary" style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} onClick={() => setModalType(null)}>
                  取消
                </button>
                {modalType === 'backup' && (
                  <button className="btn-danger" style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} onClick={handleBackupSubmit}>
                    立即备份
                  </button>
                )}
                {modalType === 'csv' && (
                  <button 
                    className="btn-danger" 
                    style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                    onClick={() => {
                      setModalType(null);
                      triggerDownload(
                        `/api/export/transactions.csv${selectedMonth ? '?month=' + selectedMonth : ''}`, 
                        `transactions${selectedMonth ? '_' + selectedMonth : ''}.csv`
                      );
                    }}
                  >
                    下载 CSV 账单
                  </button>
                )}
                {modalType === 'json' && (
                  <button 
                    className="btn-danger" 
                    style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} 
                    onClick={() => {
                      setModalType(null);
                      triggerDownload('/api/export/full.json', 'ledger_full_export.json');
                    }}
                  >
                    下载 JSON 归档
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
