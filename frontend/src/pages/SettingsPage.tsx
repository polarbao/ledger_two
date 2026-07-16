import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock,
  CreditCard,
  Database,
  Download,
  FileJson,
  FileSpreadsheet,
  HardDrive,
  Lock,
  RefreshCw,
  RotateCcw,
  ShieldCheck,
  Tags,
  User,
} from 'lucide-react';
import { ApiError } from '../api/client';
import { queryKeys } from '../api/queryKeys';
import { safetyApi, type BackupInfo, type DiagnosticStatus } from '../api/safety.api';
import LedgerSettings from '../components/ledger/LedgerSettings';
import PermissionGate from '../components/ledger/PermissionGate';
import { useHasLedgerRole } from '../components/ledger/useLedgerPermission';
import Button from '../components/ui/Button';
import ConfirmDialog from '../components/ui/ConfirmDialog';
import EmptyState from '../components/ui/EmptyState';
import RestoreBackupModal from '../components/ui/RestoreBackupModal';
import StatusChip, { type StatusChipTone } from '../components/ui/StatusChip';
import { getDeploymentChannelMeta } from '../components/layout/deploymentChannel';
import { useAuthStore } from '../stores/auth.store';
import { useLedgerStore } from '../stores/ledger.store';
import './SettingsPage.css';

type ModalType = 'backup' | 'csv' | 'json' | null;

interface SettingsSectionProps {
  id: string;
  eyebrow: string;
  title: string;
  description: string;
  children: ReactNode;
}

interface SettingsActionCardProps {
  icon: ReactNode;
  title: string;
  description: string;
  badge?: ReactNode;
  children?: ReactNode;
  tone?: 'default' | 'danger';
  wide?: boolean;
}

const roleLabels: Record<string, string> = {
  owner: 'Owner 管理员',
  editor: 'Editor 编辑者',
  viewer: 'Viewer 观察者',
};

function SettingsSection({ id, eyebrow, title, description, children }: SettingsSectionProps) {
  return (
    <section id={id} className="settings-section" aria-labelledby={`${id}-title`}>
      <header className="settings-section__header">
        <span>{eyebrow}</span>
        <h2 id={`${id}-title`}>{title}</h2>
        <p>{description}</p>
      </header>
      <div className="settings-section__content">{children}</div>
    </section>
  );
}

function SettingsActionCard({
  icon,
  title,
  description,
  badge,
  children,
  tone = 'default',
  wide = false,
}: SettingsActionCardProps) {
  return (
    <article className={`settings-card settings-card--${tone}${wide ? ' settings-card--wide' : ''}`}>
      <header className="settings-card__header">
        <span className="settings-card__icon" aria-hidden="true">{icon}</span>
        <div className="settings-card__copy">
          <div className="settings-card__title-row">
            <h3>{title}</h3>
            {badge}
          </div>
          <p>{description}</p>
        </div>
      </header>
      {children ? <div className="settings-card__body">{children}</div> : null}
    </article>
  );
}

function NoPermissionHint({ text }: { text: string }) {
  return (
    <div className="settings-permission-hint">
      <Lock size={16} aria-hidden="true" />
      <span>{text}</span>
    </div>
  );
}

function statusText(status: string) {
  if (status === 'ok') return '正常';
  if (status === 'warning') return '需关注';
  if (status === 'error') return '异常';
  return status;
}

function statusTone(status: string): StatusChipTone {
  if (status === 'ok') return 'success';
  if (status === 'warning') return 'warning';
  if (status === 'error') return 'danger';
  return 'neutral';
}

function DiagnosticLine({ item }: { item: DiagnosticStatus }) {
  const detail = item.version
    ? `schema v${item.version}`
    : item.message || (item.configured ? '已配置' : '未配置');
  const writable = typeof item.writable === 'boolean'
    ? ` · ${item.writable ? '可写' : '不可写'}`
    : '';

  return (
    <div className="settings-diagnostic-line">
      <div>
        <strong>{item.label}</strong>
        <span>{detail}{writable}</span>
      </div>
      <StatusChip tone={statusTone(item.status)}>{statusText(item.status)}</StatusChip>
    </div>
  );
}

function formatBytes(bytes: number) {
  if (bytes === 0) return '0 Bytes';
  const unit = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(unit)), sizes.length - 1);
  return `${parseFloat((bytes / Math.pow(unit, index)).toFixed(2))} ${sizes[index]}`;
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr);
  return Number.isNaN(date.getTime())
    ? dateStr
    : date.toLocaleString('zh-CN', { hour12: false });
}

function confirmationFor(type: Exclude<ModalType, null>, month: string) {
  if (type === 'backup') {
    return {
      title: '创建当前数据库的安全备份？',
      description: '系统会生成一份 SQLite 物理备份并写入审计日志，不会修改当前账单。备份文件包含完整财务数据。',
      confirmLabel: '创建安全备份',
    };
  }
  if (type === 'csv') {
    return {
      title: '导出交易流水 CSV？',
      description: `将下载${month ? `${month} 月` : '当前账本全量'}明文交易流水。导出不会改变账单状态，但文件需要按敏感财务数据保管。`,
      confirmLabel: '确认导出 CSV',
    };
  }
  return {
    title: '导出全量 JSON 数据包？',
    description: '将下载当前账本的脱敏成员信息、元数据、交易分摊和结算记录。导出不会改变业务数据。',
    confirmLabel: '确认导出 JSON',
  };
}

export default function SettingsPage() {
  const currentUser = useAuthStore((state) => state.user);
  const activeRole = useLedgerStore((state) => state.activeRole);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const canImportData = useHasLedgerRole(['owner']);
  const canExportData = useHasLedgerRole(['owner', 'editor']);
	const canManageSafety = Boolean(currentUser?.instance_admin);
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loadingBackups, setLoadingBackups] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [selectedMonth, setSelectedMonth] = useState('');
  const [selectedBackup, setSelectedBackup] = useState<BackupInfo | null>(null);
  const [modalType, setModalType] = useState<ModalType>(null);

  const fetchBackups = useCallback(async () => {
    if (!canManageSafety) return;
    setLoadingBackups(true);
    setErrorMsg(null);
    try {
      const data = await safetyApi.getBackups();
      setBackups(Array.isArray(data) ? data : []);
    } catch (error: unknown) {
      setErrorMsg(error instanceof ApiError ? `加载备份列表失败：${error.message}` : '加载备份列表失败');
    } finally {
      setLoadingBackups(false);
    }
  }, [canManageSafety]);

  useEffect(() => {
    if (canManageSafety) {
      Promise.resolve().then(() => fetchBackups());
    }
  }, [canManageSafety, fetchBackups]);

  const handleBackupSubmit = async () => {
    setModalType(null);
    setActionLoading(true);
    setErrorMsg(null);
    setSuccessMsg(null);
    try {
      const response = await safetyApi.createBackup();
      setSuccessMsg(`备份创建成功：${response.filename}`);
      await fetchBackups();
    } catch (error: unknown) {
      setErrorMsg(error instanceof ApiError ? `备份失败：${error.message}` : '备份失败，请检查备份目录写权限');
    } finally {
      setActionLoading(false);
    }
  };

	const triggerDownload = async (url: string, defaultFilename: string, ledgerScoped = false) => {
    setActionLoading(true);
    setErrorMsg(null);
    setSuccessMsg(null);
    try {
		if (ledgerScoped && !activeLedgerId) {
			throw new Error('请先选择账本');
		}
		const response = await fetch(url, {
			credentials: 'include',
			headers: ledgerScoped ? { 'X-Ledger-Id': activeLedgerId as string } : undefined,
		});
      if (!response.ok) {
        let message = '下载失败';
        try {
          const body = await response.json();
          if (body?.error?.message) message = body.error.message;
        } catch {
          // Keep the stable fallback when the server did not return JSON.
        }
        throw new Error(message);
      }
      const blobUrl = window.URL.createObjectURL(await response.blob());
      const anchor = document.createElement('a');
      anchor.href = blobUrl;
      anchor.download = defaultFilename;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.URL.revokeObjectURL(blobUrl);
      setSuccessMsg('文件下载成功');
    } catch (error: unknown) {
      setErrorMsg(error instanceof Error ? error.message : '文件下载失败，请重试');
    } finally {
      setActionLoading(false);
    }
  };

  const {
    data: diagnostics,
    isLoading: loadingDiagnostics,
    isFetching: fetchingDiagnostics,
    isError: isDiagnosticsError,
    error: diagnosticsError,
    refetch: refetchDiagnostics,
  } = useQuery({
		queryKey: queryKeys.safety.diagnostics,
    queryFn: safetyApi.getDiagnostics,
    enabled: canManageSafety,
  });

  const executeConfirmedAction = () => {
    if (modalType === 'backup') {
      void handleBackupSubmit();
      return;
    }
    if (modalType === 'csv') {
      setModalType(null);
      const query = selectedMonth ? `?month=${encodeURIComponent(selectedMonth)}` : '';
		void triggerDownload(`/api/export/transactions.csv${query}`, `transactions${selectedMonth ? `-${selectedMonth}` : ''}.csv`, true);
      return;
    }
    if (modalType === 'json') {
      setModalType(null);
		void triggerDownload('/api/export/full.json', 'ledger-two-full.json', true);
    }
  };

  const confirmation = modalType ? confirmationFor(modalType, selectedMonth) : null;
  const diagnosticsErrorText = diagnosticsError instanceof ApiError
    ? diagnosticsError.message
    : '诊断信息加载失败';

  return (
    <main className="settings-page">
      <header className="settings-page__header">
        <div>
          <span className="settings-page__eyebrow">账号、账本与数据安全</span>
          <h1>设置</h1>
          <p>按职责管理成员、元数据、自动化和数据安全。业务规则与服务端权限保持一致。</p>
        </div>
        <StatusChip tone={activeRole === 'owner' ? 'success' : 'neutral'} icon={<ShieldCheck size={14} />}>
          {activeRole ? roleLabels[activeRole] || activeRole : '未选择账本'}
        </StatusChip>
      </header>

      <div className="settings-identity" aria-label="当前账号信息">
        <User size={18} aria-hidden="true" />
        <div>
          <strong>{currentUser?.display_name || '未命名用户'}</strong>
          <span>@{currentUser?.username || '-'}</span>
        </div>
        <p>登录态由浏览器安全 Cookie 维护，服务端仍是最终权限边界。</p>
      </div>

      <nav className="settings-page__nav" aria-label="设置分区">
        <a href="#ledger">账本与成员</a>
        <a href="#metadata">分类与账户</a>
        <a href="#automation">规则与模板</a>
        <a href="#transfer">导入与导出</a>
        <a href="#safety">数据安全</a>
        <a href="#diagnostics">系统诊断</a>
      </nav>

      <div className="settings-page__messages" aria-live="polite">
        {errorMsg ? (
          <div className="settings-message settings-message--error">
            <AlertTriangle size={18} aria-hidden="true" />
            <span>{errorMsg}</span>
          </div>
        ) : null}
        {successMsg ? (
          <div className="settings-message settings-message--success">
            <CheckCircle2 size={18} aria-hidden="true" />
            <span>{successMsg}</span>
          </div>
        ) : null}
      </div>

      <SettingsSection
        id="ledger"
        eyebrow="01"
        title="账本与成员"
        description="查看当前账本成员。只有 Owner 可以创建账本、直接添加已有用户、调整角色或移除成员。"
      >
        <LedgerSettings />
      </SettingsSection>

      <SettingsSection
        id="metadata"
        eyebrow="02"
        title="分类、标签与支付账户"
        description="归档项不会进入新账单选择器，已经引用它们的历史账单仍保留原名称。"
      >
        <div className="settings-card-grid settings-card-grid--three">
          <SettingsActionCard
            icon={<Tags size={20} />}
            title="分类管理"
            description="维护支出和收入分类，支持排序、归档和恢复。"
          >
            <Link className="ui-button ui-button--secondary" to="/settings/categories">管理分类</Link>
          </SettingsActionCard>
          <SettingsActionCard
            icon={<Tags size={20} />}
            title="标签管理"
            description="维护场景、项目与报销标签，历史引用不会因归档丢失。"
          >
            <Link className="ui-button ui-button--secondary" to="/settings/tags">管理标签</Link>
          </SettingsActionCard>
          <SettingsActionCard
            icon={<CreditCard size={20} />}
            title="支付账户"
            description="管理现金、银行卡、支付宝和微信等支付来源，不计算账户余额。"
          >
            <Link className="ui-button ui-button--secondary" to="/settings/accounts">管理支付账户</Link>
          </SettingsActionCard>
        </div>
      </SettingsSection>

      <SettingsSection
        id="automation"
        eyebrow="03"
        title="周期规则与模板"
        description="周期规则只生成待确认提醒，用户确认后才会写入正式账单。"
      >
        <div className="settings-card-grid">
          <SettingsActionCard
            icon={<Clock size={20} />}
            title="周期账单规则"
            description="维护每周、每月或每年的待确认记账提醒。"
          >
            <Link className="ui-button ui-button--secondary" to="/recurring-rules">进入周期规则</Link>
          </SettingsActionCard>
          <SettingsActionCard
            icon={<FileJson size={20} />}
            title="账单模板"
            description="模板入口保留在记账抽屉中，不进入统计、结算或正式流水。"
            badge={<StatusChip>抽屉内管理</StatusChip>}
          />
        </div>
      </SettingsSection>

      <SettingsSection
        id="transfer"
        eyebrow="04"
        title="导入与导出"
        description="导入先进入预览工作区；导出产生明文财务文件。支付宝当前仅支持 CSV，微信支持 CSV/XLSX。"
      >
        <div className="settings-card-grid settings-card-grid--three">
          <SettingsActionCard
            icon={<FileSpreadsheet size={20} />}
            title="账单文件导入"
            description="上传文件后先核对状态与错误，preview 不会写入正式账单。"
          >
            {canImportData ? (
              <Link className="ui-button ui-button--secondary" to="/import">进入导入工作区</Link>
            ) : (
              <NoPermissionHint text="导入是批量写入操作，仅账本 Owner 可以使用。" />
            )}
          </SettingsActionCard>
          <SettingsActionCard
            icon={<FileSpreadsheet size={20} />}
            title="交易流水 CSV"
            description="按月份或全量导出当前角色可见流水。Owner 和 Editor 可以导出。"
            badge={<StatusChip tone="warning">明文文件</StatusChip>}
          >
            <PermissionGate allow={['owner', 'editor']} fallback={<NoPermissionHint text="Viewer 不能导出账本数据。" />}>
              <label className="settings-month-field">
                <span>月份范围</span>
                <input type="month" value={selectedMonth} onChange={(event) => setSelectedMonth(event.target.value)} />
              </label>
              <Button
                variant="primary"
                startIcon={<Download size={16} />}
                onClick={() => setModalType('csv')}
                disabled={actionLoading || !canExportData}
                fullWidth
              >
                导出 CSV
              </Button>
            </PermissionGate>
          </SettingsActionCard>
          <SettingsActionCard
            icon={<FileJson size={20} />}
            title="全量 JSON 数据包"
            description="包含当前角色可见数据的脱敏成员、元数据、交易分摊和结算记录。"
            badge={<StatusChip tone="warning">敏感归档</StatusChip>}
          >
            <PermissionGate allow={['owner', 'editor']} fallback={<NoPermissionHint text="Viewer 不能导出账本数据。" />}>
              <Button
                variant="secondary"
                startIcon={<Download size={16} />}
                onClick={() => setModalType('json')}
                disabled={actionLoading || !canExportData}
                fullWidth
              >
                导出 JSON
              </Button>
            </PermissionGate>
          </SettingsActionCard>
        </div>
      </SettingsSection>

      <SettingsSection
        id="safety"
        eyebrow="05"
        title="备份与恢复"
        description="物理备份包含整个 SQLite 数据库。恢复入口只准备安全前置备份与人工操作指引，不会在线替换运行中的数据库。"
      >
        <div className="settings-card-grid">
          <SettingsActionCard
            icon={<Database size={20} />}
            title="创建 SQLite 安全备份"
            description="生成当前数据库的只读镜像并写入审计日志，不修改任何现有账单。"
			badge={<StatusChip tone="danger">实例管理员高风险</StatusChip>}
            tone="danger"
          >
			{canManageSafety ? (
              <Button
                variant="primary"
                startIcon={<Database size={16} />}
                onClick={() => setModalType('backup')}
                disabled={actionLoading || !canManageSafety}
                fullWidth
              >
                创建安全备份
              </Button>
			) : <NoPermissionHint text="只有实例管理员可以创建、下载或准备恢复物理备份。" />}
          </SettingsActionCard>

			{canManageSafety ? (
            <SettingsActionCard
              icon={<HardDrive size={20} />}
              title="历史手动备份"
              description="下载会产生完整数据库文件；准备恢复前会再次要求输入确认短语。"
              tone="danger"
            >
              <div className="settings-card__toolbar">
                <strong>备份文件</strong>
                <Button
                  variant="ghost"
                  iconOnly
                  aria-label="刷新备份列表"
                  title="刷新备份列表"
                  onClick={() => void fetchBackups()}
                  disabled={loadingBackups}
                  startIcon={<RefreshCw className={loadingBackups ? 'animate-spin' : ''} size={17} />}
                />
              </div>
              {loadingBackups && backups.length === 0 ? (
                <div className="settings-loading"><RefreshCw className="animate-spin" size={18} />扫描备份文件中</div>
              ) : backups.length === 0 ? (
                <EmptyState title="暂无手动备份" description="正式记账或版本升级前，建议先创建一份安全备份。" />
              ) : (
                <div className="settings-backup-list">
                  {backups.map((backup) => (
                    <article key={backup.filename} className="settings-backup-item">
                      <div>
                        <strong>{backup.filename.replace('manual/', '')}</strong>
                        <span>{formatBytes(backup.size_bytes)} · {formatDate(backup.created_at)}</span>
                      </div>
                      <div className="settings-backup-item__actions">
                        <Button variant="danger" onClick={() => setSelectedBackup(backup)} disabled={actionLoading} startIcon={<RotateCcw size={14} />}>
                          准备恢复
                        </Button>
                        <Button
                          variant="secondary"
                          onClick={() => void triggerDownload(`/api/admin/backups/${encodeURIComponent(backup.filename)}`, backup.filename.split('/').pop() || 'backup.db')}
                          disabled={actionLoading}
                          startIcon={<Download size={14} />}
                        >
                          下载
                        </Button>
                      </div>
                    </article>
                  ))}
                </div>
              )}
            </SettingsActionCard>
			) : null}
        </div>
      </SettingsSection>

      <SettingsSection
        id="diagnostics"
        eyebrow="06"
        title="系统诊断"
        description="只展示脱敏运行状态，不展示密码、Cookie、密钥、DSN 或服务器绝对路径。"
      >
        <SettingsActionCard
          icon={<Activity size={20} />}
          title="运行与存储诊断"
          description="检查环境、schema、Cookie 策略、数据库和目录可写性。"
			badge={<StatusChip>实例管理员</StatusChip>}
          wide
        >
			{canManageSafety ? (
			<>
            <div className="settings-card__toolbar settings-card__toolbar--end">
              <Button
                variant="ghost"
                iconOnly
                aria-label="刷新系统诊断"
                title="刷新系统诊断"
                onClick={() => void refetchDiagnostics()}
                disabled={fetchingDiagnostics}
                startIcon={<RefreshCw className={fetchingDiagnostics ? 'animate-spin' : ''} size={17} />}
              />
            </div>
            {loadingDiagnostics ? (
              <div className="settings-loading"><RefreshCw className="animate-spin" size={18} />加载诊断信息中</div>
            ) : isDiagnosticsError ? (
              <div className="settings-permission-hint settings-permission-hint--error">
                <AlertTriangle size={16} />
                <span>{diagnosticsErrorText}</span>
              </div>
            ) : diagnostics ? (
              <div className="settings-diagnostics">
                <dl className="settings-diagnostics__summary">
                  <div><dt>运行环境</dt><dd>{getDeploymentChannelMeta(diagnostics.deployment_channel).label} · {diagnostics.env}</dd></div>
                  <div><dt>Cookie 策略</dt><dd>Secure {diagnostics.cookie_secure} · {diagnostics.cookie_samesite}</dd></div>
                  <div><dt>外部访问地址</dt><dd>{diagnostics.app_base_url_set ? '已配置' : '未配置'}</dd></div>
                </dl>
                <div className="settings-diagnostics__lines">
                  <DiagnosticLine item={diagnostics.database} />
                  {diagnostics.storage.map((item) => <DiagnosticLine key={item.key} item={item} />)}
                </div>
                <div className="settings-diagnostics__footer">
                  <span><CheckCircle2 size={15} />最近备份：{diagnostics.latest_backup ? `${diagnostics.latest_backup.filename.replace('manual/', '')} · ${formatBytes(diagnostics.latest_backup.size_bytes)}` : '暂无'}</span>
                  <span>诊断生成时间：{formatDate(diagnostics.generated_at)}</span>
                </div>
              </div>
            ) : null}
			</>
			) : <NoPermissionHint text="只有实例管理员可以查看系统诊断。" />}
        </SettingsActionCard>
      </SettingsSection>

      {selectedBackup ? (
        <RestoreBackupModal
          backup={selectedBackup}
          onClose={() => setSelectedBackup(null)}
          onSuccess={(instructions) => {
            setSuccessMsg(instructions);
            setSelectedBackup(null);
          }}
        />
      ) : null}

      <ConfirmDialog
        open={modalType !== null}
        title={confirmation?.title || ''}
        description={confirmation?.description || ''}
        confirmLabel={confirmation?.confirmLabel || '确认'}
        tone={modalType === 'backup' ? 'primary' : 'danger'}
        icon={modalType === 'backup' ? <Database /> : <AlertTriangle />}
        isConfirming={actionLoading}
        onClose={() => setModalType(null)}
        onConfirm={executeConfirmedAction}
      />
    </main>
  );
}
