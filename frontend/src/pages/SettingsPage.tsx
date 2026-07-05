import { useState, useEffect, useCallback, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
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
  X,
  RotateCcw,
  User,
  Users,
  Tags,
  CreditCard,
  ShieldCheck,
  Activity,
  Lock,
  ChevronRight,
} from 'lucide-react';
import { api, ApiError } from '../api/client';
import EmptyState from '../components/ui/EmptyState';
import LedgerSettings from '../components/ledger/LedgerSettings';
import RestoreBackupModal from '../components/ui/RestoreBackupModal';
import PermissionGate, { useHasLedgerRole } from '../components/ledger/PermissionGate';
import { useAuthStore } from '../stores/auth.store';
import { useLedgerStore } from '../stores/ledger.store';

interface BackupInfo {
  filename: string;
  size_bytes: number;
  created_at: string;
}

type ModalType = 'backup' | 'csv' | 'json' | null;

interface SettingsSectionProps {
  title: string;
  description: string;
  children: ReactNode;
}

interface SettingsActionCardProps {
  icon: ReactNode;
  title: string;
  description: string;
  badge?: string;
  children?: ReactNode;
  danger?: boolean;
}

function SettingsSection({ title, description, children }: SettingsSectionProps) {
  return (
    <section style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <h3 style={{ margin: 0, fontSize: '17px', fontWeight: 700 }}>{title}</h3>
        <p className="dimmed-desc" style={{ margin: 0, fontSize: '12px' }}>{description}</p>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: '14px' }}>
        {children}
      </div>
    </section>
  );
}

function SettingsActionCard({ icon, title, description, badge, children, danger = false }: SettingsActionCardProps) {
  return (
    <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '12px', minWidth: 0 }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', minWidth: 0 }}>
          <span style={{ color: danger ? '#f87171' : 'var(--accent-purple)', display: 'inline-flex', flexShrink: 0 }}>
            {icon}
          </span>
          <strong style={{ fontSize: '14px', lineHeight: 1.35 }}>{title}</strong>
        </div>
        {badge && (
          <span style={{ fontSize: '11px', color: danger ? '#fca5a5' : '#c084fc', border: `1px solid ${danger ? 'rgba(239,68,68,0.18)' : 'rgba(168,85,247,0.18)'}`, borderRadius: '999px', padding: '2px 8px', whiteSpace: 'nowrap' }}>
            {badge}
          </span>
        )}
      </div>
      <p className="dimmed-desc" style={{ fontSize: '12px', margin: 0, lineHeight: 1.6 }}>
        {description}
      </p>
      {children}
    </div>
  );
}

function NoPermissionHint({ text }: { text: string }) {
  return (
    <div style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.05)', borderRadius: '8px', padding: '10px 12px', color: 'var(--text-muted)', fontSize: '12px', display: 'flex', alignItems: 'center', gap: '8px' }}>
      <Lock size={14} />
      <span>{text}</span>
    </div>
  );
}

export default function SettingsPage() {
  const currentUser = useAuthStore((state) => state.user);
  const activeRole = useLedgerStore((state) => state.activeRole);
  const canExportData = useHasLedgerRole(['owner', 'editor']);
  const canManageSafety = useHasLedgerRole(['owner']);
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loadingBackups, setLoadingBackups] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [selectedMonth, setSelectedMonth] = useState<string>('');
  const [selectedBackup, setSelectedBackup] = useState<BackupInfo | null>(null);

  // 确认弹窗状态
  const [modalType, setModalType] = useState<ModalType>(null);

  // 加载备份列表
  const fetchBackups = useCallback(async () => {
    if (!canManageSafety) return;
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
  }, [canManageSafety]);

  useEffect(() => {
    if (canManageSafety) {
      Promise.resolve().then(() => {
        fetchBackups();
      });
    }
  }, [canManageSafety, fetchBackups]);

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
    } catch (err: unknown) {
      if (err instanceof Error) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg('文件下载失败，请重试');
      }
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
          <p>按账号、账本、元数据、导入导出、备份恢复和诊断能力分区管理。</p>
        </div>
      </div>

      {/* 当前身份摘要 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '14px', marginBottom: '18px' }}>
        <SettingsActionCard
          icon={<User size={18} />}
          title="账号与登录"
          description="当前登录账号和显示名称。登录态由浏览器 Cookie 维护。"
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: '13px' }}>
            <span>显示名：<strong>{currentUser?.display_name || '-'}</strong></span>
            <span className="dimmed-desc">用户名：@{currentUser?.username || '-'}</span>
          </div>
        </SettingsActionCard>
        <SettingsActionCard
          icon={<ShieldCheck size={18} />}
          title="当前账本角色"
          description="前端会按角色隐藏高风险操作，后端仍是最终权限边界。"
          badge={activeRole || '未选择'}
        >
          {!canExportData && (
            <NoPermissionHint text="当前角色可以查看数据，但不能新增、导入、导出、备份或恢复。" />
          )}
        </SettingsActionCard>
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

      <div style={{ display: 'flex', flexDirection: 'column', gap: '26px' }}>
        <SettingsSection
          title="账本与成员"
          description="管理当前账本、已有成员和成员角色。当前是直接添加已有用户，不是公开邀请机制。"
        >
          <SettingsActionCard
            icon={<Users size={18} />}
            title="成员与权限"
            description="Owner 可以创建账本、添加已有用户、调整成员角色或移除成员。Editor 和 Viewer 只展示权限说明。"
            badge="owner 管理"
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-muted)', fontSize: '12px' }}>
              <ChevronRight size={14} />
              <span>下方成员管理区保留现有能力</span>
            </div>
          </SettingsActionCard>
        </SettingsSection>

        <LedgerSettings />

        <SettingsSection
          title="分类、标签与支付账户"
          description="长期记账的基础元数据。后端归档基础已建立，前端管理页将在后续 Task35.2 接入。"
        >
          <SettingsActionCard
            icon={<Tags size={18} />}
            title="分类管理"
            description="新增、编辑、排序、归档和恢复分类。归档项不会进入新增账单默认选择器。"
            badge="待接入"
          />
          <SettingsActionCard
            icon={<Tags size={18} />}
            title="标签管理"
            description="维护账单标签和自动补全数据源。历史账单会保留已归档标签展示。"
            badge="待接入"
          />
          <SettingsActionCard
            icon={<CreditCard size={18} />}
            title="支付账户"
            description="管理现金、银行卡、支付宝、微信等支付来源，服务导入和快捷记账。"
            badge="待接入"
          />
        </SettingsSection>

        <SettingsSection
          title="周期规则与模板"
          description="管理可复用的记账配置，减少重复录入。周期规则需要用户确认后才会生成真实账单。"
        >
          <SettingsActionCard
            icon={<Clock size={18} />}
            title="周期账单规则"
            description="配置每周、每月、每年自动触发的待确认记账提醒。"
          >
            <Link
              to="/recurring-rules"
              className="btn-secondary"
              style={{ width: '100%', padding: '10px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px', fontSize: '13px', textDecoration: 'none' }}
            >
              <Settings size={14} /> 进入周期规则
            </Link>
          </SettingsActionCard>
          <SettingsActionCard
            icon={<FileJson size={18} />}
            title="账单模板"
            description="模板入口当前集成在记账抽屉中。后续会拆出独立模板管理区。"
            badge="抽屉内管理"
          />
        </SettingsSection>

        <SettingsSection
          title="导入与导出"
          description="用于补账、审计和迁移。导出文件包含明文账目信息，请妥善保管。"
        >
          <SettingsActionCard
            icon={<FileSpreadsheet size={18} />}
            title="CSV 账单导入"
            description="支持将微信、支付宝等账单 CSV 上传到预览工作区，核对后再提交。"
          >
            <PermissionGate
              allow={['owner', 'editor']}
              fallback={<NoPermissionHint text="观察者不能导入账单。" />}
            >
              <Link
                to="/import"
                className="btn-secondary"
                style={{ width: '100%', padding: '10px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px', fontSize: '13px', textDecoration: 'none' }}
              >
                <RefreshCw size={14} /> 进入 CSV 导入工作区
              </Link>
            </PermissionGate>
          </SettingsActionCard>

          <SettingsActionCard
            icon={<FileSpreadsheet size={18} />}
            title="CSV 交易流水导出"
            description="包含发生时间、标题、分类、金额、付款人、可见性和备注，适合 Excel 审计。"
          >
            <PermissionGate
              allow={['owner', 'editor']}
              fallback={<NoPermissionHint text="观察者不能导出账本数据。" />}
            >
              <div style={{ display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
                <input
                  type="month"
                  value={selectedMonth}
                  onChange={(e) => setSelectedMonth(e.target.value)}
                  style={{ flex: '1 1 150px', minWidth: 0, padding: '8px 12px', borderRadius: '8px', border: '1px solid rgba(255,255,255,0.08)', background: 'rgba(10,12,16,0.6)', color: '#fff' }}
                />
                <button
                  onClick={() => openConfirmModal('csv')}
                  className="btn-primary"
                  style={{ padding: '8px 16px', fontSize: '13px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '6px', flex: '0 0 auto' }}
                  disabled={actionLoading || !canExportData}
                >
                  <Download size={14} /> 导出 CSV
                </button>
              </div>
              {selectedMonth && (
                <span className="dimmed-desc" style={{ fontSize: '11px', color: 'var(--accent-green)' }}>
                  已选择按月份：{selectedMonth} 导出
                </span>
              )}
            </PermissionGate>
          </SettingsActionCard>

          <SettingsActionCard
            icon={<FileJson size={18} />}
            title="JSON 全量数据包导出"
            description="包含脱敏后的成员、分类、标签、交易分摊和结算记录，可用于迁移归档。"
          >
            <PermissionGate
              allow={['owner', 'editor']}
              fallback={<NoPermissionHint text="观察者不能导出账本数据。" />}
            >
              <button
                onClick={() => openConfirmModal('json')}
                className="btn-secondary"
                style={{ width: '100%', padding: '10px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px', fontSize: '13px' }}
                disabled={actionLoading || !canExportData}
              >
                <Download size={14} /> 导出全量 JSON 数据包
              </button>
            </PermissionGate>
          </SettingsActionCard>
        </SettingsSection>

        <SettingsSection
          title="备份与恢复"
          description="SQLite 物理备份和恢复属于高风险操作，仅 Owner 显示入口。恢复前仍需要二次确认。"
        >
          <SettingsActionCard
            icon={<Database size={18} />}
            title="SQLite 物理安全备份"
            description="利用 SQLite 在线事务安全备份机制生成数据库镜像文件。"
            badge="Owner"
            danger
          >
            <PermissionGate
              allow={['owner']}
              fallback={<NoPermissionHint text="只有 Owner 可以创建、恢复或下载物理备份。" />}
            >
              <button
                onClick={() => openConfirmModal('backup')}
                className="btn-primary"
                style={{ padding: '12px', borderRadius: '10px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', fontSize: '14px', fontWeight: 600 }}
                disabled={actionLoading || !canManageSafety}
              >
                <Database size={16} /> 立即创建手动安全备份
              </button>
            </PermissionGate>
          </SettingsActionCard>

          <PermissionGate
            allow={['owner']}
            fallback={null}
          >
            <SettingsActionCard
              icon={<HardDrive size={18} />}
              title="历史手动备份文件"
              description="展示已生成的备份文件，可下载或发起恢复。恢复前系统会再次确认。"
              danger
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '10px' }}>
                <strong style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>备份列表</strong>
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
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '260px', overflowY: 'auto', paddingRight: '4px' }}>
                  {backups.map((b) => (
                    <div key={b.filename} style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.03)', borderRadius: '10px', padding: '10px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', textAlign: 'left', minWidth: 0 }}>
                        <span style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-primary)', wordBreak: 'break-all' }}>
                          {b.filename.replace('manual/', '')}
                        </span>
                        <div style={{ display: 'flex', gap: '12px', fontSize: '11px', color: 'var(--text-muted)', flexWrap: 'wrap' }}>
                          <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                            <HardDrive size={12} /> {formatBytes(b.size_bytes)}
                          </span>
                          <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                            <Clock size={12} /> {formatDate(b.created_at)}
                          </span>
                        </div>
                      </div>
                      <div style={{ display: 'flex', gap: '8px' }}>
                        <button
                          onClick={() => setSelectedBackup(b)}
                          className="btn-secondary"
                          style={{ padding: '6px 12px', fontSize: '12px', borderRadius: '6px', display: 'flex', alignItems: 'center', gap: '4px', flexShrink: 0, color: 'var(--accent-danger)' }}
                          disabled={actionLoading}
                        >
                          <RotateCcw size={12} /> 恢复
                        </button>
                        <button
                          onClick={() => triggerDownload(`/api/admin/backups/${encodeURIComponent(b.filename)}`, b.filename.split('/').pop() || 'backup.db')}
                          className="btn-secondary"
                          style={{ padding: '6px 12px', fontSize: '12px', borderRadius: '6px', display: 'flex', alignItems: 'center', gap: '4px', flexShrink: 0 }}
                          disabled={actionLoading}
                        >
                          <Download size={12} /> 下载
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </SettingsActionCard>
          </PermissionGate>
        </SettingsSection>

        <SettingsSection
          title="系统诊断"
          description="用于定位配置、数据库、备份目录、上传目录和运行环境问题。完整诊断接口将在 Task40 接入。"
        >
          <SettingsActionCard
            icon={<Activity size={18} />}
            title="诊断面板"
            description="后续展示 APP_ENV、数据库状态、schema version、目录可写性和 Cookie 策略，不展示 secret 或绝对敏感路径。"
            badge="Task40"
          />
        </SettingsSection>
      </div>

      {/* 恢复备份确认弹窗 */}
      {selectedBackup && (
        <RestoreBackupModal
          backup={selectedBackup}
          onClose={() => setSelectedBackup(null)}
          onSuccess={(instructions) => {
            setSuccessMsg(instructions);
            setSelectedBackup(null);
          }}
        />
      )}

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
