import { useMemo, useState, type FormEvent } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowDown, ArrowLeft, ArrowUp, Archive, RotateCcw, Save, Search, Tags, X } from 'lucide-react';
import { metadataApi } from '../api/metadata.api';
import { queryKeys } from '../api/queryKeys';
import { useLedgerStore } from '../stores/ledger.store';
import type { MetadataItem, MetadataKind, MetadataUpsertPayload } from '../types/metadata';
import PageState from '../components/ui/PageState';
import PermissionGate from '../components/ledger/PermissionGate';
import { useHasLedgerRole } from '../components/ledger/useLedgerPermission';
import { ApiError } from '../api/client';

interface KindConfig {
  title: string;
  singular: string;
  description: string;
  namePlaceholder: string;
  typeLabel?: string;
  typeOptions?: Array<{ label: string; value: string }>;
}

const KIND_CONFIG: Record<MetadataKind, KindConfig> = {
  categories: {
    title: '分类管理',
    singular: '分类',
    description: '维护支出和收入分类。已归档分类不会出现在新增账单默认选择器中，历史账单仍保留展示。',
    namePlaceholder: '例如：餐饮、交通、工资',
    typeLabel: '分类类型',
    typeOptions: [
      { label: '支出', value: 'expense' },
      { label: '收入', value: 'income' },
    ],
  },
  tags: {
    title: '标签管理',
    singular: '标签',
    description: '维护账单标签和自动补全数据源。标签适合表达项目、场景和报销状态。',
    namePlaceholder: '例如：报销、旅行、家庭',
  },
  accounts: {
    title: '支付账户管理',
    singular: '支付账户',
    description: '维护现金、银行卡、支付宝、微信等支付来源，服务导入和快捷记账。',
    namePlaceholder: '例如：招商银行卡、支付宝、现金',
    typeLabel: '账户类型',
    typeOptions: [
      { label: '现金', value: 'cash' },
      { label: '银行卡', value: 'bank' },
      { label: '支付宝', value: 'alipay' },
      { label: '微信', value: 'wechat' },
      { label: '其他', value: 'other' },
    ],
  },
};

function parseKind(value: string | undefined): MetadataKind | null {
  if (value === 'categories' || value === 'tags' || value === 'accounts') {
    return value;
  }
  return null;
}

function defaultForm(kind: MetadataKind) {
  if (kind === 'categories') {
    return { name: '', type: 'expense', icon: '', color: '' };
  }
  if (kind === 'accounts') {
    return { name: '', type: 'cash', icon: '', color: '' };
  }
  return { name: '', type: '', icon: '', color: '' };
}

function metadataMatchesSearch(item: MetadataItem, keyword: string) {
  if (!keyword) return true;
  return [item.name, item.type, item.icon, item.color]
    .filter(Boolean)
    .some((value) => value!.toLowerCase().includes(keyword));
}

export default function MetadataManagePage() {
  const params = useParams();
  const kind = parseKind(params.kind);
  if (!kind) {
    return <Navigate to="/settings" replace />;
  }
  return <MetadataManageContent key={kind} kind={kind} />;
}

function MetadataManageContent({ kind }: { kind: MetadataKind }) {
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const canManage = useHasLedgerRole(['owner']);
  const queryClient = useQueryClient();
  const [editingItem, setEditingItem] = useState<MetadataItem | null>(null);
  const [form, setForm] = useState(defaultForm(kind));
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'archived'>('all');

  const config = KIND_CONFIG[kind];

  const { data: items = [], isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, kind),
    queryFn: () => metadataApi.list(kind, true),
  });

  const activeItems = useMemo(() => items.filter((item) => !item.is_archived), [items]);
  const archivedItems = useMemo(() => items.filter((item) => item.is_archived), [items]);
  const normalizedSearchTerm = searchTerm.trim().toLowerCase();
  const visibleActiveItems = useMemo(
    () => statusFilter === 'archived' ? [] : activeItems.filter((item) => metadataMatchesSearch(item, normalizedSearchTerm)),
    [activeItems, normalizedSearchTerm, statusFilter]
  );
  const visibleArchivedItems = useMemo(
    () => statusFilter === 'active' ? [] : archivedItems.filter((item) => metadataMatchesSearch(item, normalizedSearchTerm)),
    [archivedItems, normalizedSearchTerm, statusFilter]
  );

  const resetForm = () => {
    setEditingItem(null);
    setForm(defaultForm(kind));
  };

  const invalidateMetadata = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.metadata.list(activeLedgerId, kind) });
    if (kind === 'categories') {
      queryClient.invalidateQueries({ queryKey: queryKeys.categories(activeLedgerId) });
    }
    if (kind === 'accounts') {
      queryClient.invalidateQueries({ queryKey: queryKeys.accounts(activeLedgerId) });
    }
  };

  const submitMutation = useMutation({
    mutationFn: async (payload: MetadataUpsertPayload): Promise<unknown> => {
      if (editingItem) {
        return metadataApi.update(kind, editingItem.id, payload);
      }
      return metadataApi.create(kind, payload);
    },
    onSuccess: () => {
      setSuccessMsg(editingItem ? `${config.singular}已更新` : `${config.singular}已新增`);
      setErrorMsg(null);
      resetForm();
      invalidateMetadata();
    },
    onError: (err: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(err instanceof ApiError ? err.message : '保存失败，请稍后重试');
    },
  });

  const archiveMutation = useMutation({
    mutationFn: (item: MetadataItem) =>
      item.is_archived ? metadataApi.restore(kind, item.id) : metadataApi.archive(kind, item.id),
    onSuccess: (_, item) => {
      setSuccessMsg(item.is_archived ? `${config.singular}已恢复` : `${config.singular}已归档`);
      setErrorMsg(null);
      invalidateMetadata();
    },
    onError: (err: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(err instanceof ApiError ? err.message : '状态更新失败，请稍后重试');
    },
  });

  const reorderMutation = useMutation({
    mutationFn: (orderedIds: string[]) => metadataApi.reorder(kind, orderedIds),
    onSuccess: () => {
      setSuccessMsg(`${config.singular}排序已更新`);
      setErrorMsg(null);
      invalidateMetadata();
    },
    onError: (err: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(err instanceof ApiError ? err.message : '排序更新失败，请稍后重试');
    },
  });

  const handleEdit = (item: MetadataItem) => {
    setEditingItem(item);
    setForm({
      name: item.name,
      type: item.type || defaultForm(kind).type,
      icon: item.icon || '',
      color: item.color || '',
    });
    setErrorMsg(null);
    setSuccessMsg(null);
  };

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const payload: MetadataUpsertPayload = {
      name: form.name.trim(),
      type: form.type,
      icon: form.icon.trim(),
      color: form.color.trim(),
    };
    submitMutation.mutate(payload);
  };

  const handleArchiveToggle = (item: MetadataItem) => {
    const action = item.is_archived ? '恢复' : '归档';
    const usageCount = item.usage_count || 0;
    const usageHint = item.is_archived
      ? `恢复后，「${item.name}」会重新出现在新建账单选择器中。`
      : usageCount > 0
        ? `该${config.singular}已被 ${usageCount} 笔历史账单使用。归档后不会出现在新建账单选择器中，但历史账单仍会显示原名称。`
        : `该${config.singular}尚未被历史账单使用。归档后不会出现在新建账单选择器中。`;
    if (!window.confirm(`确认${action}「${item.name}」吗？\n\n${usageHint}`)) {
      return;
    }
    archiveMutation.mutate(item);
  };

  const moveItem = (item: MetadataItem, direction: -1 | 1) => {
    const index = activeItems.findIndex((activeItem) => activeItem.id === item.id);
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= activeItems.length) {
      return;
    }
    const nextItems = [...activeItems];
    const [current] = nextItems.splice(index, 1);
    nextItems.splice(nextIndex, 0, current);
    reorderMutation.mutate(nextItems.map((item) => item.id));
  };

  const renderItem = (item: MetadataItem, displayIndex: number, sortable: boolean) => {
    const orderIndex = item.is_archived ? displayIndex : activeItems.findIndex((activeItem) => activeItem.id === item.id);
    return (
    <div key={item.id} className="metadata-item-card">
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          <strong style={{ fontSize: '14px' }}>{item.name}</strong>
          <span style={{ fontSize: '11px', color: 'var(--text-muted)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: '999px', padding: '1px 8px' }}>
            #{orderIndex + 1}
          </span>
          {item.type && (
            <span style={{ fontSize: '11px', color: '#c084fc', border: '1px solid rgba(168,85,247,0.18)', borderRadius: '999px', padding: '1px 8px' }}>
              {item.type}
            </span>
          )}
          {item.is_archived && (
            <span style={{ fontSize: '11px', color: '#fca5a5', border: '1px solid rgba(239,68,68,0.18)', borderRadius: '999px', padding: '1px 8px' }}>
              已归档
            </span>
          )}
          <span style={{ fontSize: '11px', color: 'var(--text-muted)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: '999px', padding: '1px 8px' }}>
            引用 {item.usage_count || 0} 笔
          </span>
        </div>
        {(item.icon || item.color) && (
          <span className="dimmed-desc" style={{ fontSize: '12px' }}>
            图标：{item.icon || '-'} / 颜色：{item.color || '-'}
          </span>
        )}
      </div>
      <PermissionGate allow={['owner']}>
        <div className="metadata-item-actions">
          {sortable && (
            <>
              <button
                className="btn-secondary metadata-order-button"
                style={{ padding: '6px 8px', fontSize: '12px', borderRadius: '8px', display: 'inline-flex', alignItems: 'center' }}
                onClick={() => moveItem(item, -1)}
                disabled={orderIndex <= 0 || reorderMutation.isPending}
                title="上移"
              >
                <ArrowUp size={13} />
              </button>
              <button
                className="btn-secondary metadata-order-button"
                style={{ padding: '6px 8px', fontSize: '12px', borderRadius: '8px', display: 'inline-flex', alignItems: 'center' }}
                onClick={() => moveItem(item, 1)}
                disabled={orderIndex === activeItems.length - 1 || reorderMutation.isPending}
                title="下移"
              >
                <ArrowDown size={13} />
              </button>
            </>
          )}
          {!item.is_archived && (
            <button className="btn-secondary metadata-text-action" style={{ padding: '6px 12px', fontSize: '12px', borderRadius: '8px' }} onClick={() => handleEdit(item)}>
              编辑
            </button>
          )}
          <button
            className="btn-secondary metadata-text-action"
            style={{ padding: '6px 12px', fontSize: '12px', borderRadius: '8px', color: item.is_archived ? 'var(--accent-green)' : '#fca5a5' }}
            onClick={() => handleArchiveToggle(item)}
            disabled={archiveMutation.isPending}
          >
            {item.is_archived ? <RotateCcw size={13} /> : <Archive size={13} />}
            <span style={{ marginLeft: '4px' }}>{item.is_archived ? '恢复' : '归档'}</span>
          </button>
        </div>
      </PermissionGate>
    </div>
    );
  };

  return (
    <div className="page-content animate-fade-in text-left">
      <div className="glass-card header-banner" style={{ justifyContent: 'space-between', gap: '12px', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Tags className="banner-icon" />
          <div>
            <h2>{config.title}</h2>
            <p>{config.description}</p>
          </div>
        </div>
        <Link to="/settings" className="btn-secondary mobile-full" style={{ textDecoration: 'none', padding: '8px 14px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}>
          <ArrowLeft size={14} /> 返回设置
        </Link>
      </div>

      {errorMsg && (
        <div className="error-banner animate-fade-in" style={{ margin: '0 0 16px 0', borderRadius: '12px' }}>
          {errorMsg}
        </div>
      )}
      {successMsg && (
        <div className="glass-card text-green animate-fade-in" style={{ padding: '12px 20px', margin: '0 0 16px 0', borderRadius: '12px', background: 'rgba(16, 185, 129, 0.06)', border: '1px solid rgba(16, 185, 129, 0.2)' }}>
          <span>{successMsg}</span>
        </div>
      )}

      <div className="form-row-2">
        <PermissionGate
          allow={['owner']}
          fallback={(
            <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              <strong>只读模式</strong>
              <p className="dimmed-desc" style={{ margin: 0 }}>
                只有 Owner 可以新增、编辑、归档或恢复{config.singular}。当前角色可以查看列表。
              </p>
            </div>
          )}
        >
          <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '10px' }}>
              <strong>{editingItem ? `编辑${config.singular}` : `新增${config.singular}`}</strong>
              {editingItem && (
                <button className="btn-close-drawer" onClick={resetForm} title="取消编辑">
                  <X size={16} />
                </button>
              )}
            </div>
            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <div className="form-group" style={{ margin: 0 }}>
                <label>{config.singular}名称</label>
                <input
                  className="form-input"
                  value={form.name}
                  onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))}
                  placeholder={config.namePlaceholder}
                  maxLength={40}
                  required
                />
              </div>
              {config.typeOptions && (
                <div className="form-group" style={{ margin: 0 }}>
                  <label>{config.typeLabel}</label>
                  <select
                    className="form-select"
                    value={form.type}
                    onChange={(event) => setForm((prev) => ({ ...prev, type: event.target.value }))}
                  >
                    {config.typeOptions.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>
              )}
              <div className="form-row-2" style={{ gap: '10px' }}>
                <div className="form-group" style={{ margin: 0 }}>
                  <label>图标</label>
                  <input
                    className="form-input"
                    value={form.icon}
                    onChange={(event) => setForm((prev) => ({ ...prev, icon: event.target.value }))}
                    placeholder="可选"
                    maxLength={20}
                  />
                </div>
                <div className="form-group" style={{ margin: 0 }}>
                  <label>颜色</label>
                  <input
                    className="form-input"
                    value={form.color}
                    onChange={(event) => setForm((prev) => ({ ...prev, color: event.target.value }))}
                    placeholder="#a855f7"
                    maxLength={24}
                  />
                </div>
              </div>
              <button
                type="submit"
                className="btn-primary"
                style={{ padding: '10px 16px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px' }}
                disabled={submitMutation.isPending || !form.name.trim() || !canManage}
              >
                <Save size={15} />
                {submitMutation.isPending ? '保存中...' : editingItem ? '保存修改' : `新增${config.singular}`}
              </button>
            </form>
          </div>
        </PermissionGate>

        <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px' }}>
            <div>
              <strong>当前{config.singular}</strong>
              <p className="dimmed-desc" style={{ margin: '4px 0 0 0', fontSize: '12px' }}>
                活跃 {activeItems.length} 个，已归档 {archivedItems.length} 个。
              </p>
            </div>
            <button className="btn-close-drawer" onClick={() => refetch()} title="刷新列表">
              <RotateCcw size={16} />
            </button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            <div className="input-wrapper">
              <Search className="input-icon" />
              <input
                value={searchTerm}
                onChange={(event) => setSearchTerm(event.target.value)}
                placeholder={`搜索${config.singular}`}
                maxLength={40}
              />
              {searchTerm && (
                <button
                  className="btn-close-drawer"
                  onClick={() => setSearchTerm('')}
                  title="清空搜索"
                  style={{ position: 'absolute', right: '10px' }}
                >
                  <X size={14} />
                </button>
              )}
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '8px' }}>
              {[
                { label: '全部', value: 'all' as const },
                { label: '活跃', value: 'active' as const },
                { label: '归档', value: 'archived' as const },
              ].map((option) => (
                <button
                  key={option.value}
                  className="btn-secondary"
                  onClick={() => setStatusFilter(option.value)}
                  style={{
                    padding: '8px 10px',
                    borderRadius: '8px',
                    fontSize: '12px',
                    color: statusFilter === option.value ? 'var(--text-primary)' : 'var(--text-secondary)',
                    borderColor: statusFilter === option.value ? 'rgba(168,85,247,0.36)' : 'rgba(255,255,255,0.08)',
                  }}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>

          <PageState
            isLoading={isLoading}
            isError={isError}
            isEmpty={items.length === 0}
            emptyMessage={`暂无${config.singular}，Owner 可以在左侧新增。`}
            skeletonType="table"
            onRetry={() => refetch()}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
              {statusFilter !== 'archived' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  <span className="dimmed-desc" style={{ fontSize: '12px' }}>活跃项</span>
                  {visibleActiveItems.length === 0 ? (
                    <div className="dimmed-desc" style={{ fontSize: '12px' }}>暂无匹配的活跃项。</div>
                  ) : visibleActiveItems.map((item, index) => renderItem(item, index, activeItems.length > 1))}
                </div>
              )}

              {statusFilter !== 'active' && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                  <span className="dimmed-desc" style={{ fontSize: '12px' }}>已归档项</span>
                  {visibleArchivedItems.length === 0 ? (
                    <div className="dimmed-desc" style={{ fontSize: '12px' }}>暂无匹配的归档项。</div>
                  ) : visibleArchivedItems.map((item, index) => renderItem(item, index, false))}
                </div>
              )}
            </div>
          </PageState>
        </div>
      </div>
    </div>
  );
}
