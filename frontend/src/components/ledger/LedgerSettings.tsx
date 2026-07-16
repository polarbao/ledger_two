import { useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, BookOpen, Shield, Trash2, UserPlus, Users } from 'lucide-react';
import { ApiError } from '../../api/client';
import { ledgerApi, type LedgerMember } from '../../api/ledger.api';
import { queryKeys } from '../../api/queryKeys';
import { useLedgerStore } from '../../stores/ledger.store';
import Button from '../ui/Button';
import ConfirmDialog from '../ui/ConfirmDialog';
import PageState from '../ui/PageState';
import StatusChip from '../ui/StatusChip';
import './LedgerSettings.css';

interface RoleChange {
  member: LedgerMember;
  nextRole: 'editor' | 'viewer';
}

const roleLabels: Record<string, string> = {
  owner: 'Owner',
  editor: 'Editor',
  viewer: 'Viewer',
};

export default function LedgerSettings() {
  const queryClient = useQueryClient();
  const { activeLedgerId, activeRole, setActiveLedger } = useLedgerStore();
  const canManage = activeRole === 'owner';
  const [memberUsername, setMemberUsername] = useState('');
  const [memberRole, setMemberRole] = useState<'editor' | 'viewer'>('viewer');
  const [acknowledgeHistoryVisibility, setAcknowledgeHistoryVisibility] = useState(false);
  const [newLedgerName, setNewLedgerName] = useState('');
  const [roleChange, setRoleChange] = useState<RoleChange | null>(null);
  const [memberToRemove, setMemberToRemove] = useState<LedgerMember | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const membersQuery = useQuery({
    queryKey: queryKeys.ledgers.members(activeLedgerId),
    queryFn: () => ledgerApi.getLedgerMembers(activeLedgerId || ''),
    enabled: Boolean(activeLedgerId),
  });

  const createLedgerMutation = useMutation({
    mutationFn: (name: string) => ledgerApi.createLedger({ name }),
    onSuccess: (ledger) => {
      setNewLedgerName('');
      setErrorMsg(null);
      setSuccessMsg(`已创建并切换到账本「${ledger.name}」`);
      setActiveLedger(ledger.id, 'owner');
      void queryClient.invalidateQueries({ queryKey: queryKeys.ledgers.all });
      void queryClient.invalidateQueries();
    },
    onError: (error: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(error instanceof ApiError ? error.message : '创建账本失败，请稍后重试');
    },
  });

  const addMemberMutation = useMutation({
    mutationFn: (payload: { username: string; role: 'editor' | 'viewer' }) => {
      if (!activeLedgerId || !membersQuery.data) {
        throw new Error('账本成员信息尚未加载');
      }
      return ledgerApi.addMember(activeLedgerId, membersQuery.data.ledger.version, {
        ...payload,
        acknowledge_history_visibility: true,
      });
    },
    onSuccess: (data) => {
      setMemberUsername('');
      setAcknowledgeHistoryVisibility(false);
      setErrorMsg(null);
      setSuccessMsg('成员已添加');
      queryClient.setQueryData(queryKeys.ledgers.members(activeLedgerId), data);
    },
    onError: (error: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(error instanceof ApiError ? error.message : '添加成员失败，请稍后重试');
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: (change: RoleChange) => {
      if (!activeLedgerId || !membersQuery.data) {
        throw new Error('账本成员信息尚未加载');
      }
      return ledgerApi.updateMemberRole(
        activeLedgerId,
        membersQuery.data.ledger.version,
        change.member.user_id,
        { role: change.nextRole },
      );
    },
    onSuccess: (data, change) => {
      setRoleChange(null);
      setErrorMsg(null);
      setSuccessMsg(`${change.member.username} 的角色已调整为 ${roleLabels[change.nextRole]}`);
      queryClient.setQueryData(queryKeys.ledgers.members(activeLedgerId), data);
    },
    onError: (error: unknown) => {
      setErrorMsg(error instanceof ApiError ? error.message : '角色调整失败，请稍后重试');
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: (member: LedgerMember) => {
      if (!activeLedgerId || !membersQuery.data) {
        throw new Error('账本成员信息尚未加载');
      }
      return ledgerApi.removeMember(activeLedgerId, membersQuery.data.ledger.version, member.user_id);
    },
    onSuccess: (data, member) => {
      setMemberToRemove(null);
      setErrorMsg(null);
      setSuccessMsg(`已将 ${member.username} 移出当前账本`);
      queryClient.setQueryData(queryKeys.ledgers.members(activeLedgerId), data);
    },
    onError: (error: unknown) => {
      setErrorMsg(error instanceof ApiError ? error.message : '移除成员失败，请稍后重试');
    },
  });

  const handleCreateLedger = (event: FormEvent) => {
    event.preventDefault();
    const name = newLedgerName.trim();
    if (name) createLedgerMutation.mutate(name);
  };

  const handleAddMember = (event: FormEvent) => {
    event.preventDefault();
    const username = memberUsername.trim();
    if (username && acknowledgeHistoryVisibility) {
      addMemberMutation.mutate({ username, role: memberRole });
    }
  };

  return (
    <div className="ledger-settings">
      <div className="ledger-settings__messages" aria-live="polite">
        {errorMsg ? <div className="ledger-settings__message ledger-settings__message--error"><AlertTriangle size={16} />{errorMsg}</div> : null}
        {successMsg ? <div className="ledger-settings__message ledger-settings__message--success">{successMsg}</div> : null}
      </div>

      <div className="ledger-settings__member-panel">
        <header className="ledger-settings__panel-header">
          <div>
            <Users size={20} aria-hidden="true" />
            <div>
              <h3>当前账本成员</h3>
              <p>所有成员可以查看名单；成员管理操作只对 Owner 开放。</p>
            </div>
          </div>
          <StatusChip tone={canManage ? 'success' : 'neutral'}>{roleLabels[activeRole || ''] || '未选择'}</StatusChip>
        </header>

        <PageState
          isLoading={membersQuery.isLoading}
          isError={membersQuery.isError}
          isEmpty={!membersQuery.isLoading && (membersQuery.data?.members.length || 0) === 0}
          emptyMessage="当前账本暂无可展示成员。"
          errorMsg={membersQuery.error instanceof ApiError ? membersQuery.error.message : '成员列表加载失败'}
          onRetry={() => void membersQuery.refetch()}
          skeletonType="card"
        >
          <div className="ledger-settings__member-list">
            {(membersQuery.data?.members || []).map((member) => (
              <article key={member.user_id} className="ledger-settings__member">
                <div className="ledger-settings__member-identity">
                  <span aria-hidden="true">{member.username.slice(0, 1).toUpperCase()}</span>
                  <div>
                    <strong>{member.username}</strong>
                    <small>{member.role === 'owner' ? '账本管理员' : member.role === 'editor' ? '可记账与结算' : '只读查看'}</small>
                  </div>
                </div>
                <div className="ledger-settings__member-actions">
                  {canManage && member.role !== 'owner' ? (
                    <>
                      <label>
                        <span className="sr-only">调整 {member.username} 的角色</span>
                        <select
                          value={member.role}
                          onChange={(event) => setRoleChange({
                            member,
                            nextRole: event.target.value as 'editor' | 'viewer',
                          })}
                        >
                          <option value="editor">Editor</option>
                          <option value="viewer">Viewer</option>
                        </select>
                      </label>
                      <Button
                        variant="danger"
                        startIcon={<Trash2 size={15} />}
                        onClick={() => setMemberToRemove(member)}
                      >
                        移除
                      </Button>
                    </>
                  ) : (
                    <StatusChip tone={member.role === 'owner' ? 'success' : 'neutral'}>{roleLabels[member.role] || member.role}</StatusChip>
                  )}
                </div>
              </article>
            ))}
          </div>
        </PageState>

        {!canManage ? (
          <div className="ledger-settings__readonly">
            <Shield size={17} aria-hidden="true" />
            当前角色可查看成员，但不能添加成员、调整角色或移除成员。
          </div>
        ) : null}
      </div>

      {canManage ? (
        <div className="ledger-settings__management-grid">
          <section className="ledger-settings__form-panel">
            <header>
              <UserPlus size={19} aria-hidden="true" />
              <div><h3>添加已有用户</h3><p>这是直接添加账号，不会发送邀请或通知。</p></div>
            </header>
            <form onSubmit={handleAddMember}>
              <label>
                <span>用户名</span>
                <input
                  type="text"
                  value={memberUsername}
                  onChange={(event) => setMemberUsername(event.target.value)}
                  placeholder="输入已注册用户名"
                  maxLength={64}
                  required
                />
              </label>
              <label>
                <span>角色</span>
                <select value={memberRole} onChange={(event) => setMemberRole(event.target.value as 'editor' | 'viewer')}>
                  <option value="editor">Editor 编辑者</option>
                  <option value="viewer">Viewer 观察者</option>
                </select>
              </label>
              <label className="ledger-settings__acknowledgement">
                <input
                  type="checkbox"
                  checked={acknowledgeHistoryVisibility}
                  onChange={(event) => setAcknowledgeHistoryVisibility(event.target.checked)}
                />
                <span>新成员将按可见性规则读取当前账本的既有历史</span>
              </label>
              <Button
                type="submit"
                variant="primary"
                isLoading={addMemberMutation.isPending}
                disabled={!acknowledgeHistoryVisibility}
                fullWidth
                startIcon={<UserPlus size={16} />}
              >
                添加成员
              </Button>
            </form>
          </section>

          <section className="ledger-settings__form-panel">
            <header>
              <BookOpen size={19} aria-hidden="true" />
              <div><h3>创建独立账本</h3><p>创建后会立即切换。归档与恢复将在 Task50 实现。</p></div>
            </header>
            <form onSubmit={handleCreateLedger}>
              <label>
                <span>账本名称</span>
                <input
                  type="text"
                  value={newLedgerName}
                  onChange={(event) => setNewLedgerName(event.target.value)}
                  placeholder="例如：两人生活账本"
                  maxLength={80}
                  required
                />
              </label>
              <Button type="submit" variant="secondary" isLoading={createLedgerMutation.isPending} fullWidth startIcon={<BookOpen size={16} />}>
                创建并切换
              </Button>
            </form>
          </section>
        </div>
      ) : null}

      <ConfirmDialog
        open={roleChange !== null}
        title="确认调整成员角色？"
        description={roleChange ? `${roleChange.member.username} 将从 ${roleLabels[roleChange.member.role]} 调整为 ${roleLabels[roleChange.nextRole]}。新权限会在服务端立即生效。` : ''}
        confirmLabel="确认调整角色"
        icon={<Shield />}
        isConfirming={updateRoleMutation.isPending}
        onClose={() => setRoleChange(null)}
        onConfirm={() => roleChange && updateRoleMutation.mutate(roleChange)}
      />

      <ConfirmDialog
        open={memberToRemove !== null}
        title="将成员移出当前账本？"
        description={memberToRemove ? `${memberToRemove.username} 将立即失去当前账本及其账单、附件和统计的访问权限。历史账单不会被删除。` : ''}
        confirmLabel="确认移除成员"
        tone="danger"
        icon={<Trash2 />}
        isConfirming={removeMemberMutation.isPending}
        onClose={() => setMemberToRemove(null)}
        onConfirm={() => memberToRemove && removeMemberMutation.mutate(memberToRemove)}
      />
    </div>
  );
}
