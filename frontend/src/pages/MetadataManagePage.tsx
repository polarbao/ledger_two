import { useMemo, useState, type FormEvent } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Archive,
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  FolderTree,
  PackagePlus,
  Pencil,
  RotateCcw,
  Save,
  Search,
  ShieldCheck,
  Tag,
  WalletCards,
  X,
} from 'lucide-react';
import { ApiError } from '../api/client';
import { metadataApi } from '../api/metadata.api';
import { queryKeys } from '../api/queryKeys';
import Button from '../components/ui/Button';
import ConfirmDialog from '../components/ui/ConfirmDialog';
import PageState from '../components/ui/PageState';
import SegmentedControl from '../components/ui/SegmentedControl';
import StatusChip from '../components/ui/StatusChip';
import { useHasLedgerRole } from '../components/ledger/useLedgerPermission';
import { useLedgerStore } from '../stores/ledger.store';
import type {
  MetadataItem,
  MetadataKind,
  MetadataProfileConflictResolution,
  MetadataProfilePreviewResult,
  MetadataUpsertPayload,
} from '../types/metadata';
import './MetadataManagePage.css';

interface KindConfig {
  title: string;
  singular: string;
  description: string;
  namePlaceholder: string;
  typeLabel?: string;
  typeOptions?: Array<{ label: string; value: string }>;
}

type StatusFilter = 'all' | 'active' | 'archived';

const KIND_CONFIG: Record<MetadataKind, KindConfig> = {
  categories: {
    title: '分类管理',
    singular: '分类',
    description: '维护支出和收入分类。归档项不再进入新账单选择器，历史账单仍保留原名称。',
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
    description: '维护现金、银行卡、支付宝、微信等支付来源，不进行余额校准。',
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

const typeLabels: Record<string, string> = {
  expense: '支出',
  income: '收入',
  cash: '现金',
  bank: '银行卡',
  alipay: '支付宝',
  wechat: '微信',
  other: '其他',
};

function parseKind(value: string | undefined): MetadataKind | null {
  return value === 'categories' || value === 'tags' || value === 'accounts' ? value : null;
}

function defaultForm(kind: MetadataKind) {
  if (kind === 'categories') return { name: '', type: 'expense', icon: '', color: '#10b981' };
  if (kind === 'accounts') return { name: '', type: 'cash', icon: '', color: '#10b981' };
  return { name: '', type: '', icon: '', color: '#10b981' };
}

function metadataMatchesSearch(item: MetadataItem, keyword: string) {
  if (!keyword) return true;
  return [item.name, item.type, item.icon, item.color]
    .filter(Boolean)
    .some((value) => value!.toLowerCase().includes(keyword));
}

function validColor(value: string | undefined) {
  return value && /^#[0-9a-f]{6}$/i.test(value) ? value : '#10b981';
}

function kindIcon(kind: MetadataKind) {
  if (kind === 'categories') return <FolderTree />;
  if (kind === 'accounts') return <WalletCards />;
  return <Tag />;
}

export default function MetadataManagePage() {
  const kind = parseKind(useParams().kind);
  return kind ? <MetadataManageContent key={kind} kind={kind} /> : <Navigate to="/settings" replace />;
}

function MetadataManageContent({ kind }: { kind: MetadataKind }) {
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const canManage = useHasLedgerRole(['owner']);
  const queryClient = useQueryClient();
  const [editingItem, setEditingItem] = useState<MetadataItem | null>(null);
  const [archiveTarget, setArchiveTarget] = useState<MetadataItem | null>(null);
  const [form, setForm] = useState(defaultForm(kind));
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [replacementCategoryId, setReplacementCategoryId] = useState('');
  const [profilePreview, setProfilePreview] = useState<MetadataProfilePreviewResult | null>(null);
  const [profileResolutions, setProfileResolutions] = useState<Record<string, MetadataProfileConflictResolution>>({});
  const config = KIND_CONFIG[kind];

  const itemsQuery = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, kind),
    queryFn: ({ signal }) => metadataApi.list(kind, true, signal),
    enabled: Boolean(activeLedgerId),
  });
  const items = useMemo(() => itemsQuery.data || [], [itemsQuery.data]);
  const activeItems = useMemo(() => items.filter((item) => !item.is_archived), [items]);
  const archivedItems = useMemo(() => items.filter((item) => item.is_archived), [items]);
  const normalizedSearchTerm = searchTerm.trim().toLowerCase();
  const visibleActiveItems = useMemo(
    () => statusFilter === 'archived' ? [] : activeItems.filter((item) => metadataMatchesSearch(item, normalizedSearchTerm)),
    [activeItems, normalizedSearchTerm, statusFilter],
  );
  const visibleArchivedItems = useMemo(
    () => statusFilter === 'active' ? [] : archivedItems.filter((item) => metadataMatchesSearch(item, normalizedSearchTerm)),
    [archivedItems, normalizedSearchTerm, statusFilter],
  );

  const resetForm = () => {
    setEditingItem(null);
    setForm(defaultForm(kind));
  };

  const invalidateMetadata = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.metadata.list(activeLedgerId, kind) });
    if (kind === 'categories') void queryClient.invalidateQueries({ queryKey: queryKeys.categories(activeLedgerId) });
    if (kind === 'accounts') void queryClient.invalidateQueries({ queryKey: queryKeys.accounts(activeLedgerId) });
  };

  const submitMutation = useMutation({
    mutationFn: (payload: MetadataUpsertPayload): Promise<unknown> => editingItem
      ? metadataApi.update(kind, editingItem.id, payload)
      : metadataApi.create(kind, payload),
    onSuccess: () => {
      setSuccessMsg(editingItem ? `${config.singular}已更新` : `${config.singular}已新增`);
      setErrorMsg(null);
      resetForm();
      invalidateMetadata();
    },
    onError: (error: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(error instanceof ApiError ? error.message : '保存失败，请稍后重试');
    },
  });

  const archiveMutation = useMutation({
    mutationFn: ({ item, replacementId }: { item: MetadataItem; replacementId?: string }): Promise<unknown> => item.is_archived
      ? metadataApi.restore(kind, item.id)
      : metadataApi.archive(kind, item.id, replacementId ? { replacement_category_id: replacementId } : {}),
    onSuccess: (_, { item }) => {
      setArchiveTarget(null);
      setReplacementCategoryId('');
      setSuccessMsg(item.is_archived ? `${config.singular}已恢复` : `${config.singular}已归档`);
      setErrorMsg(null);
      invalidateMetadata();
    },
    onError: (error: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(error instanceof ApiError ? error.message : '状态更新失败，请稍后重试');
    },
  });

  const profilePreviewMutation = useMutation({
    mutationFn: () => metadataApi.previewDefaultProfile('basic_cn_v1'),
    onSuccess: (preview) => {
      setProfilePreview(preview);
      setProfileResolutions({});
      setErrorMsg(null);
    },
    onError: (error: unknown) => {
      setErrorMsg(error instanceof ApiError ? error.message : '基础分类与标签预览失败');
    },
  });

  const profileApplyMutation = useMutation({
    mutationFn: () => metadataApi.applyDefaultProfile(
      'basic_cn_v1',
      Object.values(profileResolutions),
    ),
    onSuccess: (result) => {
      setProfilePreview(null);
      setProfileResolutions({});
      setSuccessMsg(`基础包已应用：新建 ${result.created_count} 项，复用 ${result.reused_count} 项，跳过 ${result.skipped_count} 项。`);
      setErrorMsg(null);
      void queryClient.invalidateQueries({ queryKey: queryKeys.metadata.root(activeLedgerId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.categories(activeLedgerId) });
    },
    onError: (error: unknown) => {
      setErrorMsg(error instanceof ApiError ? error.message : '应用基础分类与标签失败');
    },
  });

  const reorderMutation = useMutation({
    mutationFn: (orderedIds: string[]) => metadataApi.reorder(kind, orderedIds),
    onSuccess: () => {
      setSuccessMsg(`${config.singular}排序已更新`);
      setErrorMsg(null);
      invalidateMetadata();
    },
    onError: (error: unknown) => {
      setSuccessMsg(null);
      setErrorMsg(error instanceof ApiError ? error.message : '排序更新失败，请稍后重试');
    },
  });

  const handleEdit = (item: MetadataItem) => {
    setEditingItem(item);
    setForm({
      name: item.name,
      type: item.type || defaultForm(kind).type,
      icon: item.icon || '',
      color: validColor(item.color),
    });
    setErrorMsg(null);
    setSuccessMsg(null);
  };

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    submitMutation.mutate({
      name: form.name.trim(),
      type: form.type,
      icon: form.icon.trim(),
      color: form.color.trim(),
    });
  };

  const moveItem = (item: MetadataItem, direction: -1 | 1) => {
    const index = activeItems.findIndex((candidate) => candidate.id === item.id);
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= activeItems.length) return;
    const nextItems = [...activeItems];
    const [current] = nextItems.splice(index, 1);
    nextItems.splice(nextIndex, 0, current);
    reorderMutation.mutate(nextItems.map((candidate) => candidate.id));
  };

  const renderItem = (item: MetadataItem) => {
    const orderIndex = item.is_archived ? -1 : activeItems.findIndex((candidate) => candidate.id === item.id);
    return (
      <article key={item.id} className={`metadata-item${item.is_archived ? ' metadata-item--archived' : ''}`}>
        <div className="metadata-item__identity">
          <input
            className="metadata-color-swatch"
            type="color"
            value={validColor(item.color)}
            aria-label={`${item.name} 的颜色 ${item.color || '未设置'}`}
            disabled
          />
          <div>
            <div className="metadata-item__title-row">
              <strong>{item.name}</strong>
              {!item.is_archived ? <StatusChip>排序 {orderIndex + 1}</StatusChip> : null}
              {item.type ? <StatusChip tone="info">{typeLabels[item.type] || item.type}</StatusChip> : null}
              {item.system_key ? (
                <StatusChip tone={isFallbackCategory(item) ? 'warning' : 'success'} icon={<ShieldCheck size={13} />}>
                  {isFallbackCategory(item) ? '导入兜底' : '系统基础'}
                </StatusChip>
              ) : null}
              {item.is_archived ? <StatusChip tone="warning">已归档</StatusChip> : null}
            </div>
            <span>
              {item.icon ? `图标 ${item.icon} · ` : ''}
              历史引用 {item.usage_count || 0} 笔 · 规则引用 {item.rule_reference_count || 0} 条
            </span>
          </div>
        </div>

        {canManage ? (
          <div className="metadata-item__actions">
            {!item.is_archived && activeItems.length > 1 ? (
              <div className="metadata-item__order-actions">
                <Button
                  variant="ghost"
                  iconOnly
                  aria-label={`上移 ${item.name}`}
                  title="上移"
                  startIcon={<ArrowUp size={15} />}
                  onClick={() => moveItem(item, -1)}
                  disabled={orderIndex <= 0 || reorderMutation.isPending}
                />
                <Button
                  variant="ghost"
                  iconOnly
                  aria-label={`下移 ${item.name}`}
                  title="下移"
                  startIcon={<ArrowDown size={15} />}
                  onClick={() => moveItem(item, 1)}
                  disabled={orderIndex === activeItems.length - 1 || reorderMutation.isPending}
                />
              </div>
            ) : null}
            {!item.is_archived ? (
              <Button variant="secondary" startIcon={<Pencil size={14} />} onClick={() => handleEdit(item)}>编辑</Button>
            ) : null}
            <Button
              variant={item.is_archived ? 'secondary' : 'danger'}
              startIcon={item.is_archived ? <RotateCcw size={14} /> : <Archive size={14} />}
              onClick={() => {
                setReplacementCategoryId('');
                setArchiveTarget(item);
              }}
              disabled={archiveMutation.isPending}
            >
              {item.is_archived ? '恢复' : '归档'}
            </Button>
          </div>
        ) : null}
      </article>
    );
  };

  const archiveUsage = archiveTarget?.usage_count || 0;
  const archiveDescription = archiveTarget?.is_archived
    ? `恢复后，「${archiveTarget.name}」会重新进入新建账单选择器。历史账单不会改变。`
    : archiveTarget
      ? `归档后，「${archiveTarget.name}」不会再进入新建账单选择器，但 ${archiveUsage} 笔历史引用仍保留原名称。`
      : '';

  return (
    <main className="metadata-page">
      <header className="metadata-page__header">
        <div className="metadata-page__title">
          <span className="metadata-page__icon" aria-hidden="true">{kindIcon(kind)}</span>
          <div>
            <span className="metadata-page__eyebrow">设置 / 元数据</span>
            <h1>{config.title}</h1>
            <p>{config.description}</p>
          </div>
        </div>
        <Link to="/settings" className="ui-button ui-button--secondary"><ArrowLeft size={16} />返回设置</Link>
      </header>

      <div className="metadata-page__messages" aria-live="polite">
        {errorMsg ? <div className="metadata-message metadata-message--error">{errorMsg}</div> : null}
        {successMsg ? <div className="metadata-message metadata-message--success">{successMsg}</div> : null}
      </div>

      <div className="metadata-page__layout">
        <aside className="metadata-editor">
          {canManage ? (
            <>
              <header className="metadata-panel-heading">
                <div><h2>{editingItem ? `编辑${config.singular}` : `新增${config.singular}`}</h2><p>同一账本内名称不可重复。</p></div>
                {editingItem ? (
                  <Button variant="ghost" iconOnly aria-label="取消编辑" title="取消编辑" startIcon={<X size={17} />} onClick={resetForm} />
                ) : null}
              </header>
              <form onSubmit={handleSubmit}>
                <label>
                  <span>{config.singular}名称</span>
                  <input
                    value={form.name}
                    onChange={(event) => setForm((previous) => ({ ...previous, name: event.target.value }))}
                    placeholder={config.namePlaceholder}
                    maxLength={40}
                    required
                  />
                </label>
                {config.typeOptions ? (
                  <label>
                    <span>{config.typeLabel}</span>
                    <select value={form.type} onChange={(event) => setForm((previous) => ({ ...previous, type: event.target.value }))}>
                      {config.typeOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                    </select>
                  </label>
                ) : null}
                <label>
                  <span>图标文字（可选）</span>
                  <input
                    value={form.icon}
                    onChange={(event) => setForm((previous) => ({ ...previous, icon: event.target.value }))}
                    placeholder="例如：餐、行、卡"
                    maxLength={20}
                  />
                </label>
                <label>
                  <span>识别颜色</span>
                  <div className="metadata-color-field">
                    <input
                      type="color"
                      value={validColor(form.color)}
                      onChange={(event) => setForm((previous) => ({ ...previous, color: event.target.value }))}
                      aria-label="选择颜色"
                    />
                    <input
                      value={form.color}
                      onChange={(event) => setForm((previous) => ({ ...previous, color: event.target.value }))}
                      placeholder="#10b981"
                      maxLength={7}
                      pattern="#[0-9a-fA-F]{6}"
                    />
                  </div>
                </label>
                <Button
                  type="submit"
                  variant="primary"
                  fullWidth
                  isLoading={submitMutation.isPending}
                  disabled={!form.name.trim()}
                  startIcon={<Save size={16} />}
                >
                  {editingItem ? '保存修改' : `新增${config.singular}`}
                </Button>
              </form>
            </>
          ) : (
            <div className="metadata-readonly">
              <StatusChip tone="neutral">只读模式</StatusChip>
              <h2>当前角色仅可查看</h2>
              <p>只有 Owner 可以新增、编辑、排序、归档或恢复{config.singular}，后端会拒绝越权写入。</p>
            </div>
          )}
        </aside>

        <section className="metadata-list-panel">
          <header className="metadata-panel-heading metadata-panel-heading--list">
            <div><h2>当前{config.singular}</h2><p>活跃 {activeItems.length} 个，已归档 {archivedItems.length} 个。</p></div>
            <div className="metadata-panel-heading__actions">
              {canManage && kind !== 'accounts' ? (
                <Button
                  variant="secondary"
                  startIcon={<PackagePlus size={16} />}
                  onClick={() => profilePreviewMutation.mutate()}
                  isLoading={profilePreviewMutation.isPending}
                >
                  补充基础分类与标签
                </Button>
              ) : null}
              <Button
                variant="ghost"
                iconOnly
                aria-label="刷新列表"
                title="刷新列表"
                startIcon={<RotateCcw size={17} />}
                onClick={() => void itemsQuery.refetch()}
                disabled={itemsQuery.isFetching}
              />
            </div>
          </header>

          <div className="metadata-toolbar">
            <label className="metadata-search">
              <Search size={17} aria-hidden="true" />
              <span className="sr-only">搜索{config.singular}</span>
              <input value={searchTerm} onChange={(event) => setSearchTerm(event.target.value)} placeholder={`搜索${config.singular}`} maxLength={40} />
              {searchTerm ? (
                <Button variant="ghost" iconOnly aria-label="清空搜索" title="清空搜索" startIcon={<X size={15} />} onClick={() => setSearchTerm('')} />
              ) : null}
            </label>
            <SegmentedControl
              ariaLabel={`${config.singular}状态筛选`}
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { label: '全部', value: 'all', count: items.length },
                { label: '活跃', value: 'active', count: activeItems.length },
                { label: '归档', value: 'archived', count: archivedItems.length },
              ]}
            />
          </div>

          <PageState
            isLoading={itemsQuery.isLoading}
            isError={itemsQuery.isError}
            isEmpty={items.length === 0}
            emptyMessage={`暂无${config.singular}${canManage ? '，可以在左侧新增。' : '。'}`}
            errorMsg={itemsQuery.error instanceof ApiError ? itemsQuery.error.message : `获取${config.singular}失败`}
            skeletonType="table"
            onRetry={() => void itemsQuery.refetch()}
          >
            <div className="metadata-list">
              {statusFilter !== 'archived' ? (
                <section className="metadata-list__group">
                  <h3>活跃项</h3>
                  {visibleActiveItems.length ? visibleActiveItems.map(renderItem) : <p>没有匹配的活跃项。</p>}
                </section>
              ) : null}
              {statusFilter !== 'active' ? (
                <section className="metadata-list__group">
                  <h3>已归档项</h3>
                  {visibleArchivedItems.length ? visibleArchivedItems.map(renderItem) : <p>没有匹配的归档项。</p>}
                </section>
              ) : null}
            </div>
          </PageState>
        </section>
      </div>

      <ConfirmDialog
        open={archiveTarget !== null}
        title={archiveTarget?.is_archived ? `恢复${config.singular}「${archiveTarget.name}」？` : `归档${config.singular}「${archiveTarget?.name || ''}」？`}
        description={archiveDescription}
        confirmLabel={archiveTarget?.is_archived ? '确认恢复' : '确认归档'}
        tone={archiveTarget?.is_archived ? 'primary' : 'danger'}
        icon={archiveTarget?.is_archived ? <RotateCcw /> : <Archive />}
        isConfirming={archiveMutation.isPending}
        confirmDisabled={Boolean(archiveTarget && isFallbackCategory(archiveTarget) && !archiveTarget.is_archived && !replacementCategoryId)}
        onClose={() => setArchiveTarget(null)}
        onConfirm={() => archiveTarget && archiveMutation.mutate({
          item: archiveTarget,
          replacementId: replacementCategoryId || undefined,
        })}
      >
        {archiveTarget && isFallbackCategory(archiveTarget) && !archiveTarget.is_archived ? (
          <label className="metadata-replacement-field">
            <span>选择新的{archiveTarget.type === 'income' ? '收入' : '支出'}兜底分类</span>
            <select
              value={replacementCategoryId}
              onChange={(event) => setReplacementCategoryId(event.target.value)}
            >
              <option value="">请选择替代分类</option>
              {activeItems
                .filter((item) => item.id !== archiveTarget.id && item.type === archiveTarget.type && !item.system_key)
                .map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
            </select>
            <small>只转移兜底职责，不改写历史账单或已有规则。</small>
          </label>
        ) : null}
      </ConfirmDialog>

      <ConfirmDialog
        open={profilePreview !== null}
        title="补充基础分类与标签"
        description="先预览创建、复用和冲突项。冲突必须逐项选择复用或跳过，现有名称、颜色和排序不会被覆盖。"
        confirmLabel="确认应用基础包"
        cancelLabel="暂不应用"
        icon={<PackagePlus />}
        isConfirming={profileApplyMutation.isPending}
        confirmDisabled={!profilePreview || !profileConflictsResolved(profilePreview, profileResolutions)}
        onClose={() => {
          setProfilePreview(null);
          setProfileResolutions({});
        }}
        onConfirm={() => profileApplyMutation.mutate()}
      >
        {profilePreview ? (
          <div className="metadata-profile-preview">
            <dl>
              <div><dt>将创建</dt><dd>{profilePreview.create_count}</dd></div>
              <div><dt>将复用</dt><dd>{profilePreview.reuse_count}</dd></div>
              <div><dt>待处理冲突</dt><dd>{profilePreview.conflict_count}</dd></div>
            </dl>
            <div className="metadata-profile-preview__items">
              {profilePreview.profile.items.map((item) => (
                <article key={item.system_key}>
                  <div>
                    <strong>{item.name}</strong>
                    <span>{profileKindLabel(item.kind)} · {profileActionLabel(item.action)}</span>
                  </div>
                  {item.action === 'conflict' ? (
                    <select
                      aria-label={`处理 ${item.name} 冲突`}
                      value={serializeProfileResolution(profileResolutions[item.system_key])}
                      onChange={(event) => setProfileResolutions((current) => {
                        const next = { ...current };
                        if (!event.target.value) delete next[item.system_key];
                        else next[item.system_key] = parseProfileResolution(item.system_key, event.target.value);
                        return next;
                      })}
                    >
                      <option value="">请选择处理方式</option>
                      {item.existing_id ? (
                        <option value={`reuse:${item.existing_id}`}>复用现有“{item.name}”</option>
                      ) : null}
                      <option value="skip">跳过此项</option>
                    </select>
                  ) : (
                    <StatusChip tone={item.action === 'create' ? 'success' : 'neutral'}>
                      {profileActionLabel(item.action)}
                    </StatusChip>
                  )}
                </article>
              ))}
            </div>
          </div>
        ) : null}
      </ConfirmDialog>
    </main>
  );
}

function isFallbackCategory(item: MetadataItem) {
  return item.system_key === 'expense_other' || item.system_key === 'income_other';
}

function profileConflictsResolved(
  preview: MetadataProfilePreviewResult,
  resolutions: Record<string, MetadataProfileConflictResolution>,
) {
  return preview.profile.items
    .filter((item) => item.action === 'conflict')
    .every((item) => Boolean(resolutions[item.system_key]));
}

function parseProfileResolution(systemKey: string, value: string): MetadataProfileConflictResolution {
  if (value === 'skip') return { system_key: systemKey, action: 'skip' };
  return { system_key: systemKey, action: 'reuse', existing_id: value.replace(/^reuse:/, '') };
}

function serializeProfileResolution(resolution?: MetadataProfileConflictResolution) {
  if (!resolution) return '';
  return resolution.action === 'skip' ? 'skip' : `reuse:${resolution.existing_id}`;
}

function profileKindLabel(kind: MetadataProfilePreviewResult['profile']['items'][number]['kind']) {
  return {
    expense_category: '支出分类',
    income_category: '收入分类',
    tag: '标签',
  }[kind];
}

function profileActionLabel(action: MetadataProfilePreviewResult['profile']['items'][number]['action']) {
  return {
    create: '将创建',
    reuse: '将复用',
    skip: '将跳过',
    conflict: '需要处理',
    existing: '已存在',
  }[action];
}
