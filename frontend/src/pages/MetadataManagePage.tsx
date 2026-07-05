import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Archive, RotateCcw, Save, Tags, X } from 'lucide-react';
import { metadataApi } from '../api/metadata.api';
import { queryKeys } from '../api/queryKeys';
import { useLedgerStore } from '../stores/ledger.store';
import type { MetadataItem, MetadataKind, MetadataUpsertPayload } from '../types/metadata';
import PageState from '../components/ui/PageState';
import PermissionGate, { useHasLedgerRole } from '../components/ledger/PermissionGate';
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

export default function MetadataManagePage() {
  const params = useParams();
  const kind = parseKind(params.kind);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const canManage = useHasLedgerRole(['owner']);
  const queryClient = useQueryClient();
  const [editingItem, setEditingItem] = useState<MetadataItem | null>(null);
  const [form, setForm] = useState(defaultForm(kind || 'categories'));
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const config = kind ? KIND_CONFIG[kind] : null;

  const { data: items = [], isLoading, isError, refetch } = useQuery({
    queryKey: kind ? queryKeys.metadata.list(activeLedgerId, kind) : queryKeys.metadata.root(activeLedgerId),
    queryFn: () => metadataApi.list(kind as MetadataKind, true),
    enabled: !!kind,
  });

  const activeItems = useMemo(() => items.filter((item) => !item.is_archived), [items]);
  const archivedItems = useMemo(() => items.filter((item) => item.is_archived), [items]);

  useEffect(() => {
    if (!kind) return;
    setEditingItem(null);
    setForm(defaultForm(kind));
    setErrorMsg(null);
    setSuccessMsg(null);
  }, [kind]);

  if (!kind || !config) {
    return <Navigate to="/settings" replace />;
  }

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
    mutationFn: (payload: MetadataUpsertPayload) => {
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
    if (!window.confirm(`确认${action}「${item.name}」吗？历史账单展示不会被删除。`)) {
      return;
    }
    archiveMutation.mutate(item);
  };

  const renderItem = (item: MetadataItem) => (
    <div key={item.id} style={{ background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '10px', padding: '12px 14px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px', flexWrap: 'wrap' }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          <strong style={{ fontSize: '14px' }}>{item.name}</strong>
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
        </div>
        {(item.icon || item.color) && (
          <span className="dimmed-desc" style={{ fontSize: '12px' }}>
            图标：{item.icon || '-'} / 颜色：{item.color || '-'}
          </span>
        )}
      </div>
      <PermissionGate allow={['owner']}>
        <div style={{ display: 'flex', gap: '8px' }}>
          {!item.is_archived && (
            <button className="btn-secondary" style={{ padding: '6px 12px', fontSize: '12px', borderRadius: '8px' }} onClick={() => handleEdit(item)}>
              编辑
            </button>
          )}
          <button
            className="btn-secondary"
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
        <Link to="/settings" className="btn-secondary" style={{ textDecoration: 'none', padding: '8px 14px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '6px' }}>
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

          <PageState
            isLoading={isLoading}
            isError={isError}
            isEmpty={items.length === 0}
            emptyMessage={`暂无${config.singular}，Owner 可以在左侧新增。`}
            skeletonType="table"
            onRetry={() => refetch()}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '18px' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <span className="dimmed-desc" style={{ fontSize: '12px' }}>活跃项</span>
                {activeItems.length === 0 ? (
                  <div className="dimmed-desc" style={{ fontSize: '12px' }}>暂无活跃项。</div>
                ) : activeItems.map(renderItem)}
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <span className="dimmed-desc" style={{ fontSize: '12px' }}>已归档项</span>
                {archivedItems.length === 0 ? (
                  <div className="dimmed-desc" style={{ fontSize: '12px' }}>暂无归档项。</div>
                ) : archivedItems.map(renderItem)}
              </div>
            </div>
          </PageState>
        </div>
      </div>
    </div>
  );
}
