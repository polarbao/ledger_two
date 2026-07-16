import { useMemo, useRef, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  ArrowLeft,
  Crown,
  Download,
  FileJson,
  FileSpreadsheet,
  LogOut,
  Shield,
  Trash2,
  UserPlus,
  Users,
} from 'lucide-react';
import { useForm } from 'react-hook-form';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { z } from 'zod';
import {
  ledgerApi,
  type LedgerMember,
  type LedgerMemberList,
  type LedgerWithRole,
} from '../api/ledger.api';
import { queryKeys } from '../api/queryKeys';
import LedgerActionSurface from '../components/ledger/LedgerActionSurface';
import LedgerLifecycleActions from '../components/ledger/LedgerLifecycleActions';
import {
  getLedgerCapabilities,
  getLedgerErrorPresentation,
} from '../components/ledger/ledgerManagementModel';
import { switchActiveLedgerContext } from '../components/layout/ledgerContextModel';
import Button from '../components/ui/Button';
import PageState from '../components/ui/PageState';
import StatePanel from '../components/ui/StatePanel';
import StatusChip from '../components/ui/StatusChip';
import { useAuthStore } from '../stores/auth.store';
import { useLedgerStore } from '../stores/ledger.store';
import { useUIStore } from '../stores/ui.store';
import { formatDate } from '../utils/date';
import './LedgerManagementPage.css';

const addMemberSchema = z.object({
  username: z.string().trim().min(1, '请输入用户名').max(64, '用户名过长'),
  role: z.enum(['editor', 'viewer']),
});

type AddMemberValues = z.infer<typeof addMemberSchema>;

interface RoleChange {
  member: LedgerMember;
  nextRole: 'editor' | 'viewer';
}

const roleLabels = {
  owner: 'Owner',
  editor: 'Editor',
  viewer: 'Viewer',
} as const;

const roleDescriptions = {
  owner: '管理生命周期、成员、导入与元数据',
  editor: '可记账、结算和导出',
  viewer: '只读，不可导出',
} as const;

function upsertLedger(list: LedgerWithRole[] | undefined, ledger: LedgerWithRole, status: 'active' | 'archived') {
  const withoutLedger = (list ?? []).filter((item) => item.id !== ledger.id);
  return ledger.status === status ? [ledger, ...withoutLedger] : withoutLedger;
}

export default function LedgerDetailPage() {
  const { ledgerId = '' } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const setActiveLedger = useLedgerStore((state) => state.setActiveLedger);
  const reconcileActiveLedgers = useLedgerStore((state) => state.reconcileActiveLedgers);
  const isOffline = useUIStore((state) => state.isOffline);
  const memberSectionRef = useRef<HTMLElement>(null);
  const [acknowledgeHistory, setAcknowledgeHistory] = useState(false);
  const [roleChange, setRoleChange] = useState<RoleChange | null>(null);
  const [transferTarget, setTransferTarget] = useState<LedgerMember | null>(null);
  const [removeTarget, setRemoveTarget] = useState<LedgerMember | null>(null);
  const [leaveOpen, setLeaveOpen] = useState(false);
  const [pageMessage, setPageMessage] = useState<string | null>(null);
  const [exporting, setExporting] = useState<'csv' | 'json' | null>(null);

  const membersQuery = useQuery({
    queryKey: queryKeys.ledgers.members(ledgerId),
    queryFn: ({ signal }) => ledgerApi.getLedgerMembers(ledgerId, signal),
    enabled: Boolean(ledgerId),
  });
  const snapshot = membersQuery.data;
  const ledger = snapshot?.ledger;
  const capabilities = ledger ? getLedgerCapabilities(ledger) : null;
  const isActive = ledger?.status === 'active';
  const canManage = Boolean(capabilities?.canManageMembers);
  const membersById = useMemo(
    () => new Map((snapshot?.members ?? []).map((member) => [member.user_id, member])),
    [snapshot?.members],
  );
  const currentMember = currentUser ? membersById.get(currentUser.id) : undefined;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<AddMemberValues>({
    resolver: zodResolver(addMemberSchema),
    defaultValues: { username: '', role: 'viewer' },
  });

  const syncLedger = (updatedLedger: LedgerWithRole) => {
    const nextActive = upsertLedger(
      queryClient.getQueryData<LedgerWithRole[]>(queryKeys.ledgers.list('active')),
      updatedLedger,
      'active',
    );
    queryClient.setQueryData(queryKeys.ledgers.list('active'), nextActive);
    queryClient.setQueryData<LedgerWithRole[]>(
      queryKeys.ledgers.list('archived'),
      (current) => upsertLedger(current, updatedLedger, 'archived'),
    );
    queryClient.setQueryData<LedgerMemberList>(
      queryKeys.ledgers.members(updatedLedger.id),
      (current) => current ? { ...current, ledger: updatedLedger } : current,
    );
    if (updatedLedger.id === activeLedgerId) {
      if (updatedLedger.status === 'archived') reconcileActiveLedgers(nextActive);
      else setActiveLedger(updatedLedger.id, updatedLedger.role);
    }
  };

  const updateSnapshot = (nextSnapshot: LedgerMemberList) => {
    queryClient.setQueryData(queryKeys.ledgers.members(ledgerId), nextSnapshot);
    syncLedger(nextSnapshot.ledger);
  };

  const addMemberMutation = useMutation({
    mutationFn: (values: AddMemberValues) => {
      if (!ledger) throw new Error('账本信息尚未加载');
      return ledgerApi.addMember(ledger.id, ledger.version, {
        ...values,
        acknowledge_history_visibility: true,
      });
    },
    onSuccess: (nextSnapshot) => {
      updateSnapshot(nextSnapshot);
      reset();
      setAcknowledgeHistory(false);
      setPageMessage('成员已添加，历史可见性按账本规则立即生效。');
    },
  });
  const roleMutation = useMutation({
    mutationFn: (change: RoleChange) => {
      if (!ledger) throw new Error('账本信息尚未加载');
      return ledgerApi.updateMemberRole(ledger.id, ledger.version, change.member.user_id, {
        role: change.nextRole,
      });
    },
    onSuccess: (nextSnapshot, change) => {
      updateSnapshot(nextSnapshot);
      setRoleChange(null);
      setPageMessage(`${change.member.username} 已调整为 ${roleLabels[change.nextRole]}。`);
    },
  });
  const transferMutation = useMutation({
    mutationFn: (member: LedgerMember) => {
      if (!ledger) throw new Error('账本信息尚未加载');
      return ledgerApi.transferOwner(ledger.id, ledger.version, member.user_id, {
        acknowledge_permission_change: true,
      });
    },
    onSuccess: (nextSnapshot, member) => {
      updateSnapshot(nextSnapshot);
      setTransferTarget(null);
      setPageMessage(`所有权已原子移交给 ${member.username}，你现在是 Editor。`);
    },
  });
  const removeMutation = useMutation({
    mutationFn: (member: LedgerMember) => {
      if (!ledger) throw new Error('账本信息尚未加载');
      return ledgerApi.removeMember(ledger.id, ledger.version, member.user_id);
    },
    onSuccess: (nextSnapshot, member) => {
      updateSnapshot(nextSnapshot);
      setRemoveTarget(null);
      setPageMessage(`${member.username} 已被移出账本，历史数据未删除。`);
    },
  });
  const leaveMutation = useMutation({
    mutationFn: () => {
      if (!ledger) throw new Error('账本信息尚未加载');
      return ledgerApi.leaveLedger(ledger.id, ledger.version);
    },
    onSuccess: () => {
      const nextActive = (
        queryClient.getQueryData<LedgerWithRole[]>(queryKeys.ledgers.list('active')) ?? []
      ).filter((item) => item.id !== ledgerId);
      queryClient.setQueryData(queryKeys.ledgers.list('active'), nextActive);
      queryClient.setQueryData<LedgerWithRole[]>(
        queryKeys.ledgers.list('archived'),
        (current = []) => current.filter((item) => item.id !== ledgerId),
      );
      if (ledgerId === activeLedgerId) reconcileActiveLedgers(nextActive);
      setLeaveOpen(false);
      navigate('/settings/ledgers');
    },
  });

  const mutationError = addMemberMutation.error
    ?? roleMutation.error
    ?? transferMutation.error
    ?? removeMutation.error
    ?? leaveMutation.error;
  const mutationErrorPresentation = mutationError
    ? getLedgerErrorPresentation(mutationError)
    : null;

  const switchThenNavigate = async (targetLedger: LedgerWithRole, destination: string) => {
    if (targetLedger.id !== activeLedgerId) {
      await switchActiveLedgerContext({
        queryClient,
        currentLedgerId: activeLedgerId,
        nextLedger: targetLedger,
        commit: (nextLedger) => setActiveLedger(nextLedger.id, nextLedger.role),
      });
    }
    navigate(destination);
  };

  const downloadExport = async (kind: 'csv' | 'json') => {
    if (!ledger || !capabilities?.canExport) return;
    setExporting(kind);
    setPageMessage(null);
    const url = kind === 'csv' ? '/api/export/transactions.csv' : '/api/export/full.json';
    try {
      const response = await fetch(url, {
        credentials: 'include',
        headers: { 'X-Ledger-Id': ledger.id },
      });
      if (!response.ok) {
        let message = '导出失败，请稍后重试。';
        try {
          const body = await response.json();
          if (body?.error?.message) message = body.error.message;
        } catch {
          // Keep the stable fallback for non-JSON download errors.
        }
        throw new Error(message);
      }
      const blobUrl = window.URL.createObjectURL(await response.blob());
      const anchor = document.createElement('a');
      anchor.href = blobUrl;
      anchor.download = kind === 'csv'
        ? `ledger-${ledger.id}-transactions.csv`
        : `ledger-${ledger.id}-full.json`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      window.URL.revokeObjectURL(blobUrl);
      setPageMessage(`${kind.toUpperCase()} 已导出。`);
    } catch (error) {
      setPageMessage(error instanceof Error ? error.message : '导出失败，请稍后重试。');
    } finally {
      setExporting(null);
    }
  };

  return (
    <main className="ledger-detail-page">
      <PageState
        isLoading={membersQuery.isLoading}
        isError={membersQuery.isError}
        isEmpty={!membersQuery.isLoading && !snapshot}
        errorMsg={getLedgerErrorPresentation(membersQuery.error).message}
        emptyMessage="无法读取该账本。"
        skeletonType="card"
        onRetry={() => void membersQuery.refetch()}
      >
        {ledger && snapshot ? (
          <>
            <header className="ledger-detail-page__header">
              <div>
                <Link className="ledger-detail-page__back" to="/settings/ledgers">
                  <ArrowLeft size={16} aria-hidden="true" />
                  返回账本管理
                </Link>
                <span className="ledger-detail-page__eyebrow">账本资料与成员</span>
                <h1>{ledger.name}</h1>
                <p>成员关系、历史数据和所有权变化均由服务端权限与审计记录约束。</p>
              </div>
              <div className="ledger-detail-page__header-actions">
                <StatusChip tone={isActive ? 'success' : 'warning'}>
                  {isActive ? '活跃' : '已归档 · 只读'}
                </StatusChip>
                <StatusChip>{roleLabels[ledger.role]}</StatusChip>
              </div>
            </header>

            {isOffline ? (
              <div className="ledger-detail-page__notice" role="status">
                当前离线，成员和生命周期写操作已暂停。
              </div>
            ) : null}
            {pageMessage ? (
              <div className="ledger-detail-page__message" role="status">{pageMessage}</div>
            ) : null}
            {mutationErrorPresentation ? (
              <div className="ledger-detail-page__message ledger-detail-page__message--error" role="alert">
                {mutationErrorPresentation.message}
              </div>
            ) : null}

            <section className="ledger-detail-section" aria-labelledby="ledger-basic-title">
              <header>
                <div>
                  <Shield size={20} aria-hidden="true" />
                  <div>
                    <span>基本信息</span>
                    <h2 id="ledger-basic-title">状态与生命周期</h2>
                  </div>
                </div>
                <LedgerLifecycleActions
                  ledger={ledger}
                  mode="inline"
                  onUpdated={(updatedLedger) => {
                    syncLedger(updatedLedger);
                    setPageMessage(
                      updatedLedger.status === 'archived'
                        ? '账本已归档，全员进入只读状态。'
                        : '账本信息已更新。',
                    );
                  }}
                  onGoToImports={(target) => void switchThenNavigate(target, '/import')}
                />
              </header>
              <dl className="ledger-detail-facts">
                <div><dt>账本名称</dt><dd>{ledger.name}</dd></div>
                <div><dt>本人角色</dt><dd>{roleLabels[ledger.role]} · {roleDescriptions[ledger.role]}</dd></div>
                <div><dt>成员数量</dt><dd>{ledger.member_count}/2</dd></div>
                <div><dt>创建时间</dt><dd>{formatDate(ledger.created_at)}</dd></div>
                <div><dt>归档时间</dt><dd>{ledger.archived_at ? formatDate(ledger.archived_at) : '未归档'}</dd></div>
                <div><dt>并发版本</dt><dd>v{ledger.version}</dd></div>
              </dl>
            </section>

            <section ref={memberSectionRef} className="ledger-detail-section" aria-labelledby="ledger-members-title">
              <header>
                <div>
                  <Users size={20} aria-hidden="true" />
                  <div>
                    <span>成员</span>
                    <h2 id="ledger-members-title">访问角色与所有权</h2>
                  </div>
                </div>
                <StatusChip tone={snapshot.members.length >= 2 ? 'warning' : 'neutral'}>
                  {snapshot.members.length >= 2 ? '两人上限已满' : '可添加 1 人'}
                </StatusChip>
              </header>

              <div className="ledger-member-list">
                {snapshot.members.map((member) => {
                  const isSelf = member.user_id === currentUser?.id;
                  const canManageTarget = canManage && member.role !== 'owner';
                  return (
                    <article className="ledger-member-row" key={member.user_id}>
                      <div className="ledger-member-row__identity">
                        <span aria-hidden="true">{member.username.slice(0, 1).toUpperCase()}</span>
                        <div>
                          <strong>{member.username}{isSelf ? '（我）' : ''}</strong>
                          <small>加入于 {formatDate(member.joined_at)}</small>
                        </div>
                      </div>
                      <div className="ledger-member-row__role">
                        {canManageTarget ? (
                          <label>
                            <span className="sr-only">调整 {member.username} 的角色</span>
                            <select
                              value={member.role}
                              disabled={isOffline}
                              onChange={(event) => setRoleChange({
                                member,
                                nextRole: event.target.value as 'editor' | 'viewer',
                              })}
                            >
                              <option value="editor">Editor · 可记账、结算、导出</option>
                              <option value="viewer">Viewer · 只读，不可导出</option>
                            </select>
                          </label>
                        ) : (
                          <StatusChip tone={member.role === 'owner' ? 'success' : 'neutral'}>
                            {roleLabels[member.role]}
                          </StatusChip>
                        )}
                        <span>{roleDescriptions[member.role]}</span>
                      </div>
                      <div className="ledger-member-row__actions">
                        {canManageTarget ? (
                          <>
                            <Button
                              variant="secondary"
                              startIcon={<Crown size={16} />}
                              disabled={isOffline}
                              onClick={() => setTransferTarget(member)}
                            >
                              移交所有权
                            </Button>
                            <Button
                              variant="ghost"
                              iconOnly
                              aria-label={`移除 ${member.username}`}
                              title={`移除 ${member.username}`}
                              disabled={isOffline}
                              onClick={() => setRemoveTarget(member)}
                            >
                              <Trash2 size={17} aria-hidden="true" />
                            </Button>
                          </>
                        ) : null}
                        {isSelf && member.role !== 'owner' && isActive ? (
                          <Button
                            variant="danger"
                            startIcon={<LogOut size={16} />}
                            disabled={isOffline}
                            onClick={() => setLeaveOpen(true)}
                          >
                            离开账本
                          </Button>
                        ) : null}
                      </div>
                    </article>
                  );
                })}
              </div>

              {canManage && snapshot.members.length < 2 ? (
                <form
                  className="ledger-detail-form"
                  onSubmit={handleSubmit((values) => {
                    if (acknowledgeHistory) addMemberMutation.mutate(values);
                  })}
                >
                  <header>
                    <UserPlus size={18} aria-hidden="true" />
                    <div>
                      <h3>新成员可查看部分历史</h3>
                      <p>按精确 username 添加已存在账号，不发送邀请或通知。</p>
                    </div>
                  </header>
                  <div className="ledger-detail-form__grid">
                    <label>
                      <span>用户名</span>
                      <input
                        {...register('username')}
                        type="text"
                        maxLength={64}
                        placeholder="输入已注册用户名"
                        aria-invalid={Boolean(errors.username)}
                      />
                      {errors.username ? <small role="alert">{errors.username.message}</small> : null}
                    </label>
                    <label>
                      <span>角色</span>
                      <select {...register('role')}>
                        <option value="editor">Editor · 可记账、结算、导出</option>
                        <option value="viewer">Viewer · 只读，不可导出</option>
                      </select>
                    </label>
                  </div>
                  <div className="ledger-history-warning">
                    <AlertTriangle size={18} aria-hidden="true" />
                    <ul>
                      <li>新成员可读取既有 partner_readable 与 shared 历史。</li>
                      <li>private 账单仍不可见，历史所有者和分摊不会改变。</li>
                    </ul>
                  </div>
                  <label className="ledger-action-checkbox">
                    <input
                      type="checkbox"
                      checked={acknowledgeHistory}
                      onChange={(event) => setAcknowledgeHistory(event.target.checked)}
                    />
                    <span>我已确认历史可见性范围</span>
                  </label>
                  <Button
                    type="submit"
                    variant="primary"
                    startIcon={<UserPlus size={16} />}
                    isLoading={addMemberMutation.isPending}
                    disabled={!acknowledgeHistory || isOffline}
                  >
                    添加成员
                  </Button>
                </form>
              ) : null}

              {currentMember?.role === 'owner' && isActive ? (
                <div className="ledger-owner-leave-block">
                  <Crown size={18} aria-hidden="true" />
                  <span>Owner 不能直接离开。请先将所有权移交给另一名成员。</span>
                  <Button
                    variant="ghost"
                    disabled={snapshot.members.length < 2}
                    onClick={() => memberSectionRef.current?.scrollIntoView({ behavior: 'smooth' })}
                  >
                    前往移交
                  </Button>
                </div>
              ) : null}
            </section>

            <section className="ledger-detail-section" aria-labelledby="ledger-data-title">
              <header>
                <div>
                  <Download size={20} aria-hidden="true" />
                  <div>
                    <span>数据操作</span>
                    <h2 id="ledger-data-title">当前角色可见数据导出</h2>
                  </div>
                </div>
                {capabilities?.canExport ? <StatusChip tone="warning">明文财务文件</StatusChip> : null}
              </header>
              {capabilities?.canExport ? (
                <div className="ledger-export-actions">
                  <Button
                    variant="secondary"
                    startIcon={<FileSpreadsheet size={17} />}
                    isLoading={exporting === 'csv'}
                    disabled={exporting !== null}
                    onClick={() => void downloadExport('csv')}
                  >
                    导出账本 CSV
                  </Button>
                  <Button
                    variant="secondary"
                    startIcon={<FileJson size={17} />}
                    isLoading={exporting === 'json'}
                    disabled={exporting !== null}
                    onClick={() => void downloadExport('json')}
                  >
                    导出账本 JSON
                  </Button>
                </div>
              ) : (
                <StatePanel
                  tone="neutral"
                  icon={<Shield size={36} />}
                  title="当前角色不可导出"
                  description="Viewer 可查看账本历史，但不能生成明文财务文件。"
                />
              )}
            </section>
          </>
        ) : null}
      </PageState>

      <LedgerActionSurface
        open={roleChange !== null}
        title="调整成员角色"
        description={roleChange
          ? `${roleChange.member.username} 将从 ${roleLabels[roleChange.member.role]} 调整为 ${roleLabels[roleChange.nextRole]}，服务端权限立即生效。`
          : ''}
        confirmLabel="调整成员角色"
        icon={<Shield size={22} />}
        isConfirming={roleMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (roleMutation.isPending) return;
          roleMutation.reset();
          setRoleChange(null);
        }}
        onConfirm={() => roleChange && roleMutation.mutate(roleChange)}
      />

      <LedgerActionSurface
        open={transferTarget !== null}
        title="你将失去账本管理权限"
        description="目标成员将成为唯一 Owner，你会同时降为 Editor；历史账单、分摊和结算不会改变。"
        confirmLabel={`移交所有权给 ${transferTarget?.username ?? ''}`}
        tone="danger"
        icon={<Crown size={22} />}
        isConfirming={transferMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (transferMutation.isPending) return;
          transferMutation.reset();
          setTransferTarget(null);
        }}
        onConfirm={() => transferTarget && transferMutation.mutate(transferTarget)}
      />

      <LedgerActionSurface
        open={removeTarget !== null}
        title="对方将立即失去访问"
        description="移除不会删除对方创建、拥有、支付或参与的历史账单，也不会改写结算、附件和审计。"
        confirmLabel={`移除 ${removeTarget?.username ?? ''}`}
        tone="danger"
        icon={<Trash2 size={22} />}
        isConfirming={removeMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (removeMutation.isPending) return;
          removeMutation.reset();
          setRemoveTarget(null);
        }}
        onConfirm={() => removeTarget && removeMutation.mutate(removeTarget)}
      />

      <LedgerActionSurface
        open={leaveOpen}
        title="你将立即失去访问"
        description="离开不会删除你的历史账单或审计记录，完成后返回其他活跃账本或无账本状态。"
        confirmLabel="离开账本"
        tone="danger"
        icon={<LogOut size={22} />}
        isConfirming={leaveMutation.isPending}
        confirmDisabled={isOffline}
        onClose={() => {
          if (leaveMutation.isPending) return;
          leaveMutation.reset();
          setLeaveOpen(false);
        }}
        onConfirm={() => leaveMutation.mutate()}
      />
    </main>
  );
}
