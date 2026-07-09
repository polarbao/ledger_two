import { useMemo, useRef, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  Archive,
  Ban,
  CheckCircle2,
  CircleAlert,
  Clock3,
  FileSpreadsheet,
  FileWarning,
  Loader2,
  RefreshCw,
  RotateCcw,
  Save,
  ShieldCheck,
  Sparkles,
  Upload,
  X,
} from 'lucide-react';
import { ApiError } from '../api/client';
import { importsApi } from '../api/imports.api';
import { metadataApi } from '../api/metadata.api';
import { queryKeys } from '../api/queryKeys';
import { useLedgerStore } from '../stores/ledger.store';
import { centsToYuan } from '../utils/money';
import type {
  ImportDuplicateStatus,
  ImportCommitResult,
  ImportPreviewBatch,
  ImportPreviewRow,
  ImportRule,
  ImportRuleMatchType,
  ImportRuleUpsertPayload,
  ImportRowStatus,
  ImportSourceType,
} from '../types/imports';
import type { MetadataItem } from '../types/metadata';

const sourceOptions: Array<{ value: ImportSourceType; label: string; description: string }> = [
  { value: 'wechat', label: '微信账单', description: '微信支付导出的 CSV' },
  { value: 'alipay', label: '支付宝账单', description: '支付宝流水明细 CSV' },
  { value: 'generic', label: '通用模板', description: 'LedgerTwo 标准 CSV' },
];

const duplicateStatusCopy: Record<
  ImportDuplicateStatus,
  { label: string; tone: 'new' | 'duplicate' | 'suspicious' | 'invalid'; detail: string }
> = {
  new: { label: '新增', tone: 'new', detail: '可进入后续导入确认' },
  duplicate: { label: '重复', tone: 'duplicate', detail: '默认跳过' },
  suspicious: { label: '疑似重复', tone: 'suspicious', detail: '后续需要人工确认' },
  invalid: { label: '错误', tone: 'invalid', detail: '需修正或跳过' },
};

const rowStatusCopy: Record<ImportRowStatus, string> = {
  pending: '待处理',
  adjusted: '已调整',
  skipped: '已跳过',
  imported: '已导入',
  failed: '不可用',
};

const matchTypeOptions: Array<{ value: ImportRuleMatchType; label: string }> = [
  { value: 'merchant_contains', label: '商户包含' },
  { value: 'description_contains', label: '描述包含' },
  { value: 'source_account', label: '来源账户' },
  { value: 'amount_range', label: '金额区间' },
];

type ImportRuleForm = {
  name: string;
  match_type: ImportRuleMatchType;
  pattern: string;
  category_id: string;
  account_id: string;
  tag_ids: string[];
  priority: string;
};

const createDefaultRuleForm = (): ImportRuleForm => ({
  name: '',
  match_type: 'merchant_contains',
  pattern: '',
  category_id: '',
  account_id: '',
  tag_ids: [],
  priority: '100',
});

export default function ImportPage() {
  const activeRole = useLedgerStore((state) => state.activeRole);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [sourceType, setSourceType] = useState<ImportSourceType>('wechat');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [batch, setBatch] = useState<ImportPreviewBatch | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [commitResult, setCommitResult] = useState<ImportCommitResult | null>(null);
  const [ruleForm, setRuleForm] = useState<ImportRuleForm>(() => createDefaultRuleForm());
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [ruleStatusFilter, setRuleStatusFilter] = useState<'all' | 'active' | 'archived'>('all');

  const isOwner = activeRole === 'owner';

  const { data: importRules = [] } = useQuery({
    queryKey: queryKeys.importRules(activeLedgerId),
    queryFn: () => importsApi.listRules('all'),
    enabled: isOwner,
  });
  const { data: categories = [] } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, 'categories'),
    queryFn: () => metadataApi.list('categories', true),
    enabled: isOwner,
  });
  const { data: accounts = [] } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, 'accounts'),
    queryFn: () => metadataApi.list('accounts', true),
    enabled: isOwner,
  });
  const { data: tags = [] } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, 'tags'),
    queryFn: () => metadataApi.list('tags', true),
    enabled: isOwner,
  });

  const previewMutation = useMutation({
    mutationFn: (file: File) => importsApi.preview({ file, sourceType }),
    onSuccess: (data) => {
      setBatch(data);
      setCommitResult(null);
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setBatch(null);
      setErrorMsg(resolveErrorMessage(err, '生成导入预览失败，请检查来源和 CSV 文件格式'));
    },
  });

  const updateRowMutation = useMutation({
    mutationFn: ({ row, rowStatus }: { row: ImportPreviewRow; rowStatus: 'pending' | 'adjusted' | 'skipped' }) =>
      importsApi.updateRow(batch?.id || '', row.id, { row_status: rowStatus }),
    onSuccess: (data) => {
      setBatch(data);
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setErrorMsg(resolveErrorMessage(err, '更新预览行失败'));
    },
  });

  const summary = useMemo(() => buildSummary(batch), [batch]);
  const commitSummary = useMemo(() => buildCommitSummary(batch), [batch]);
  const canOpenCommit = isOwner && !!batch && batch.status === 'ready' && !previewMutation.isPending && !updateRowMutation.isPending;

  const commitMutation = useMutation({
    mutationFn: async () => {
      if (!batch) {
        throw new Error('missing batch');
      }
      const result = await importsApi.commit(batch.id);
      const latestBatch = await importsApi.getBatch(batch.id);
      return { result, latestBatch };
    },
    onSuccess: ({ result, latestBatch }) => {
      setBatch(latestBatch);
      setCommitResult(result);
      setIsConfirmOpen(false);
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setIsConfirmOpen(false);
      setErrorMsg(resolveErrorMessage(err, '导入提交失败，当前批次未写入正式账单'));
    },
  });

  const createRuleMutation = useMutation({
    mutationFn: (payload: ImportRuleUpsertPayload) => importsApi.createRule(payload),
    onSuccess: () => {
      setRuleForm(createDefaultRuleForm());
      queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setErrorMsg(resolveErrorMessage(err, '创建导入规则失败'));
    },
  });

  const updateRuleMutation = useMutation({
    mutationFn: ({ ruleId, payload }: { ruleId: string; payload: ImportRuleUpsertPayload }) => importsApi.updateRule(ruleId, payload),
    onSuccess: () => {
      setRuleForm(createDefaultRuleForm());
      setEditingRuleId(null);
      queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setErrorMsg(resolveErrorMessage(err, '更新导入规则失败'));
    },
  });

  const archiveRuleMutation = useMutation({
    mutationFn: (ruleId: string) => importsApi.archiveRule(ruleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
    },
    onError: (err: unknown) => {
      setErrorMsg(resolveErrorMessage(err, '归档导入规则失败'));
    },
  });

  const restoreRuleMutation = useMutation({
    mutationFn: (ruleId: string) => importsApi.restoreRule(ruleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
    },
    onError: (err: unknown) => {
      setErrorMsg(resolveErrorMessage(err, '恢复导入规则失败'));
    },
  });

  const handleFile = (file: File) => {
    if (!file.name.toLowerCase().endsWith('.csv')) {
      setErrorMsg('当前仅支持 CSV 文件');
      return;
    }
    setSelectedFile(file);
    setBatch(null);
    previewMutation.mutate(file);
  };

  const handleReset = () => {
    setSelectedFile(null);
    setBatch(null);
    setCommitResult(null);
    setErrorMsg(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleOpenCommit = () => {
    if (!batch) {
      return;
    }
    if (commitSummary.blockingCount > 0) {
      setErrorMsg('仍有错误或未确认的疑似重复行，请先跳过或确认后再提交');
      return;
    }
    setIsConfirmOpen(true);
  };

  const handleSubmitRule = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = buildRulePayload(ruleForm);
    if (!payload) {
      setErrorMsg('规则需要填写匹配内容，并至少选择分类、账户或标签');
      return;
    }
    if (editingRuleId) {
      updateRuleMutation.mutate({ ruleId: editingRuleId, payload });
      return;
    }
    createRuleMutation.mutate(payload);
  };

  const handleEditRule = (rule: ImportRule) => {
    setEditingRuleId(rule.id);
    setRuleForm({
      name: rule.name,
      match_type: rule.match_type,
      pattern: rule.pattern,
      category_id: rule.result.category_id || '',
      account_id: rule.result.account_id || '',
      tag_ids: rule.result.tag_ids || [],
      priority: String(rule.priority),
    });
  };

  const handleCancelRuleEdit = () => {
    setEditingRuleId(null);
    setRuleForm(createDefaultRuleForm());
  };

  return (
    <div className="page-content animate-fade-in text-left import-workbench">
      <div className="glass-card header-banner import-workbench__hero">
        <FileSpreadsheet className="banner-icon" />
        <div>
          <h2>导入预览工作台</h2>
          <p>CSV 上传后先生成可审阅批次，确认无误后再写入正式账单。</p>
        </div>
      </div>

      {errorMsg && (
        <div className="error-banner import-workbench__notice" role="alert">
          <AlertTriangle size={18} />
          <span>{errorMsg}</span>
          <button type="button" className="btn-close-drawer" onClick={() => setErrorMsg(null)} aria-label="关闭错误提示">
            <X size={16} />
          </button>
        </div>
      )}

      {!isOwner && (
        <div className="glass-card import-permission-card">
          <ShieldCheck size={22} />
          <div>
            <h3>当前角色不可导入</h3>
            <p>导入会影响账本数据质量，当前仅账本 Owner 可以创建预览批次。你仍可查看已有账单和报表。</p>
          </div>
        </div>
      )}

      <section className={`import-layout ${batch ? 'has-preview' : ''}`} aria-disabled={!isOwner}>
        <div className="glass-card import-entry">
          <div className="import-section-title">
            <span>来源</span>
            <small>{selectedFile ? selectedFile.name : '未选择文件'}</small>
          </div>

          <div className="import-source-tabs" role="tablist" aria-label="导入来源">
            {sourceOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                className={`import-source-tab ${sourceType === option.value ? 'is-active' : ''}`}
                onClick={() => {
                  setSourceType(option.value);
                  setBatch(null);
                }}
                disabled={!isOwner || previewMutation.isPending}
              >
                <strong>{option.label}</strong>
                <span>{option.description}</span>
              </button>
            ))}
          </div>

          <label
            className={`import-upload-zone ${!isOwner ? 'is-disabled' : ''}`}
            onDragOver={(event) => {
              event.preventDefault();
            }}
            onDrop={(event) => {
              event.preventDefault();
              const file = event.dataTransfer.files?.[0];
              if (file && isOwner) {
                handleFile(file);
              }
            }}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv"
              disabled={!isOwner || previewMutation.isPending}
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) {
                  handleFile(file);
                }
              }}
            />
            <span className="import-upload-zone__icon">
              {previewMutation.isPending ? <Loader2 size={26} className="spin" /> : <Upload size={26} />}
            </span>
            <strong>{previewMutation.isPending ? '正在生成预览' : '选择 CSV 文件'}</strong>
            <small>微信、支付宝或通用模板 CSV，单批最多 500 行。</small>
          </label>

          <div className="import-entry__actions">
            <button type="button" className="btn-secondary" onClick={handleReset} disabled={previewMutation.isPending}>
              <RefreshCw size={14} />
              重置
            </button>
            <button type="button" className="btn-primary" disabled>
              上传后在预览区提交
            </button>
          </div>
        </div>

        {isOwner && (
          <ImportRuleManager
            rules={importRules}
            categories={categories}
            accounts={accounts}
            tags={tags}
            form={ruleForm}
            creating={createRuleMutation.isPending}
            updating={updateRuleMutation.isPending}
            archiving={archiveRuleMutation.isPending}
            restoring={restoreRuleMutation.isPending}
            editingRuleId={editingRuleId}
            statusFilter={ruleStatusFilter}
            onFormChange={setRuleForm}
            onSubmit={handleSubmitRule}
            onCancelEdit={handleCancelRuleEdit}
            onEdit={handleEditRule}
            onStatusFilterChange={setRuleStatusFilter}
            onArchive={(ruleId) => archiveRuleMutation.mutate(ruleId)}
            onRestore={(ruleId) => restoreRuleMutation.mutate(ruleId)}
          />
        )}

        <div className="glass-card import-preview-panel">
          <div className="import-section-title">
            <span>预览批次</span>
            <small>{batch ? `Batch ${batch.id.slice(0, 8)}` : '等待上传'}</small>
          </div>

          <div className="import-safe-banner">
            <ShieldCheck size={16} />
            <span>{batch?.status === 'committed' ? '当前批次已写入正式账单，可在流水中查看。' : '当前批次只保存预览数据，提交前不会写入 transactions。'}</span>
          </div>

          <div className="import-summary-grid">
            {summary.map((item) => (
              <div key={item.label} className={`import-summary-card ${item.tone}`}>
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </div>
            ))}
          </div>

          <div className="import-preview-actions">
            <button type="button" className="btn-primary" disabled={!canOpenCommit || commitMutation.isPending} onClick={handleOpenCommit}>
              {commitMutation.isPending ? (
                <>
                  <Loader2 size={14} className="spin" />
                  正在提交
                </>
              ) : batch?.status === 'committed' ? '已完成导入' : '确认导入'}
            </button>
          </div>

          {commitResult && (
            <div className="import-result-card" role="status">
              <CheckCircle2 size={18} />
              <div>
                <strong>导入完成</strong>
                <span>
                  已导入 {commitResult.imported_rows} 条，跳过 {commitResult.skipped_rows} 条，失败 {commitResult.failed_rows} 条。
                </span>
              </div>
            </div>
          )}

          {!batch ? (
            <div className="import-empty-state">
              <FileWarning size={34} />
              <strong>还没有预览批次</strong>
              <span>上传 CSV 后会在这里看到行级状态和错误原因。</span>
            </div>
          ) : (
            <div className="import-row-list" aria-label="导入预览行">
              {batch.rows.map((row) => (
                <ImportRowCard
                  key={row.id}
                  row={row}
                  categories={categories}
                  accounts={accounts}
                  tags={tags}
                  disabled={updateRowMutation.isPending}
                  onSkip={() => updateRowMutation.mutate({ row, rowStatus: 'skipped' })}
                  onRestore={() => updateRowMutation.mutate({ row, rowStatus: 'pending' })}
                  onConfirmImport={() => updateRowMutation.mutate({ row, rowStatus: 'adjusted' })}
                />
              ))}
            </div>
          )}
        </div>
      </section>

      {isConfirmOpen && batch && (
        <div className="modal-overlay" onClick={() => setIsConfirmOpen(false)}>
          <div className="confirm-modal-box animate-fade-in import-commit-modal" onClick={(event) => event.stopPropagation()}>
            <div className="modal-header">
              <h3>确认导入账单？</h3>
              <button type="button" className="btn-close-drawer" onClick={() => setIsConfirmOpen(false)} aria-label="关闭">
                <X size={18} />
              </button>
            </div>
            <div className="modal-body-padding">
              <p className="modal-alert-text">
                系统将把当前预览批次写入正式账单。导入过程在单个事务中完成，失败时不会保留半批数据。
              </p>
              <div className="import-summary-grid">
                <div className="import-summary-card new">
                  <span>将导入</span>
                  <strong>{commitSummary.importableCount}</strong>
                </div>
                <div className="import-summary-card duplicate">
                  <span>将跳过</span>
                  <strong>{commitSummary.skippedCount}</strong>
                </div>
                <div className="import-summary-card suspicious">
                  <span>疑似未确认</span>
                  <strong>{commitSummary.unconfirmedSuspiciousCount}</strong>
                </div>
                <div className="import-summary-card invalid">
                  <span>错误未跳过</span>
                  <strong>{commitSummary.invalidOpenCount}</strong>
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="btn-secondary mobile-full" onClick={() => setIsConfirmOpen(false)}>
                  返回预览
                </button>
                <button
                  type="button"
                  className="btn-danger mobile-full"
                  onClick={() => commitMutation.mutate()}
                  disabled={commitMutation.isPending || commitSummary.blockingCount > 0}
                >
                  {commitMutation.isPending ? '正在导入' : '确认导入'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ImportRuleManager({
  rules,
  categories,
  accounts,
  tags,
  form,
  creating,
  updating,
  archiving,
  restoring,
  editingRuleId,
  statusFilter,
  onFormChange,
  onSubmit,
  onCancelEdit,
  onEdit,
  onStatusFilterChange,
  onArchive,
  onRestore,
}: {
  rules: ImportRule[];
  categories: MetadataItem[];
  accounts: MetadataItem[];
  tags: MetadataItem[];
  form: ImportRuleForm;
  creating: boolean;
  updating: boolean;
  archiving: boolean;
  restoring: boolean;
  editingRuleId: string | null;
  statusFilter: 'all' | 'active' | 'archived';
  onFormChange: (form: ImportRuleForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onCancelEdit: () => void;
  onEdit: (rule: ImportRule) => void;
  onStatusFilterChange: (status: 'all' | 'active' | 'archived') => void;
  onArchive: (ruleId: string) => void;
  onRestore: (ruleId: string) => void;
}) {
  const activeRules = rules.filter((rule) => rule.status === 'active');
  const visibleRules = rules.filter((rule) => statusFilter === 'all' || rule.status === statusFilter);
  const activeCategories = categories.filter((item) => !item.is_archived);
  const activeAccounts = accounts.filter((item) => !item.is_archived);
  const activeTags = tags.filter((item) => !item.is_archived);
  const busy = creating || updating || archiving || restoring;
  const selectedTagIds = new Set(form.tag_ids);

  const toggleTag = (tagId: string) => {
    onFormChange({
      ...form,
      tag_ids: selectedTagIds.has(tagId) ? form.tag_ids.filter((id) => id !== tagId) : [...form.tag_ids, tagId],
    });
  };

  return (
    <div className="glass-card import-rule-manager">
      <div className="import-section-title">
        <span>导入规则</span>
        <small>{activeRules.length} 条启用</small>
      </div>

      <div className="import-rule-filter" role="tablist" aria-label="导入规则状态">
        {(['all', 'active', 'archived'] as const).map((status) => (
          <button
            key={status}
            type="button"
            className={statusFilter === status ? 'is-active' : ''}
            onClick={() => onStatusFilterChange(status)}
          >
            {status === 'all' ? '全部' : status === 'active' ? '启用' : '归档'}
          </button>
        ))}
      </div>

      <form className="import-rule-form" onSubmit={onSubmit}>
        <input
          value={form.name}
          onChange={(event) => onFormChange({ ...form, name: event.target.value })}
          placeholder="规则名称"
        />
        <select
          value={form.match_type}
          onChange={(event) => onFormChange({ ...form, match_type: event.target.value as ImportRuleMatchType })}
        >
          {matchTypeOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
        <input
          value={form.pattern}
          onChange={(event) => onFormChange({ ...form, pattern: event.target.value })}
          placeholder="匹配内容，例如 星巴克"
        />
        <select
          value={form.category_id}
          onChange={(event) => onFormChange({ ...form, category_id: event.target.value })}
        >
          <option value="">不推荐分类</option>
          {activeCategories.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
        </select>
        <select
          value={form.account_id}
          onChange={(event) => onFormChange({ ...form, account_id: event.target.value })}
        >
          <option value="">不推荐账户</option>
          {activeAccounts.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
        </select>
        <div className="import-rule-tag-options" aria-label="推荐标签">
          {activeTags.length === 0 ? (
            <span>暂无可用标签</span>
          ) : activeTags.map((item) => (
            <label key={item.id} className={`import-rule-tag-option ${selectedTagIds.has(item.id) ? 'is-selected' : ''}`}>
              <input
                type="checkbox"
                checked={selectedTagIds.has(item.id)}
                onChange={() => toggleTag(item.id)}
              />
              <span>{item.name}</span>
            </label>
          ))}
        </div>
        <input
          type="number"
          min="0"
          value={form.priority}
          onChange={(event) => onFormChange({ ...form, priority: event.target.value })}
          placeholder="优先级"
        />
        <button type="submit" className="btn-primary" disabled={creating || updating}>
          {creating || updating ? <Loader2 size={14} className="spin" /> : <Save size={14} />}
          {editingRuleId ? '保存规则' : '创建规则'}
        </button>
        {editingRuleId && (
          <button type="button" className="btn-secondary" onClick={onCancelEdit} disabled={busy}>
            <X size={14} />
            取消编辑
          </button>
        )}
      </form>

      <div className="import-rule-list">
        {visibleRules.map((rule) => {
          const metadataWarning = describeRuleMetadataWarning(rule, categories, accounts, tags);
          return (
            <article key={rule.id} className={`import-rule-card ${rule.status === 'archived' ? 'is-archived' : ''}`}>
              <div>
                <strong>{rule.name || rule.pattern}</strong>
                <span>{matchTypeLabel(rule.match_type)}「{rule.pattern}」 · 优先级 {rule.priority}</span>
                <small>{describeRuleResult(rule, categories, accounts, tags)}</small>
                {metadataWarning && <small className="import-rule-warning">{metadataWarning}</small>}
              </div>
              <div className="import-rule-card__actions">
                <button type="button" className="btn-secondary" disabled={busy} onClick={() => onEdit(rule)}>
                  编辑
                </button>
                {rule.status === 'active' ? (
                  <button type="button" className="btn-secondary" disabled={busy} onClick={() => onArchive(rule.id)}>
                    <Archive size={14} />
                    归档
                  </button>
                ) : (
                  <button type="button" className="btn-secondary" disabled={busy} onClick={() => onRestore(rule.id)}>
                    <RotateCcw size={14} />
                    恢复
                  </button>
                )}
              </div>
            </article>
          );
        })}
        {visibleRules.length === 0 && (
          <div className="import-rule-empty">
            <Sparkles size={18} />
            <span>还没有导入规则</span>
          </div>
        )}
      </div>
    </div>
  );
}

function ImportRowCard({
  row,
  categories,
  accounts,
  tags,
  disabled,
  onSkip,
  onRestore,
  onConfirmImport,
}: {
  row: ImportPreviewRow;
  categories: MetadataItem[];
  accounts: MetadataItem[];
  tags: MetadataItem[];
  disabled: boolean;
  onSkip: () => void;
  onRestore: () => void;
  onConfirmImport: () => void;
}) {
  const status = duplicateStatusCopy[row.duplicate_status];
  const isSkipped = row.row_status === 'skipped';
  const isConfirmedSuspicious = row.duplicate_status === 'suspicious' && row.row_status === 'adjusted';
  const canRestore = isSkipped && row.duplicate_status !== 'invalid' && row.duplicate_status !== 'duplicate';
  const canSkip = !isSkipped && row.row_status !== 'failed';
  const canConfirmImport = row.duplicate_status === 'suspicious' && row.row_status === 'pending';
  const suggestionDetail = describeRowSuggestion(row, categories, accounts, tags);

  return (
    <article className={`import-row-card tone-${status.tone}`}>
      <div className="import-row-card__top">
        <div>
          <span className="import-row-number">第 {row.row_number} 行</span>
          <h3>{row.title || row.merchant || '未命名流水'}</h3>
          <p>{row.merchant || '未识别商户'}</p>
        </div>
        <strong className="import-row-amount">¥{centsToYuan(row.amount_cents)}</strong>
      </div>

      <div className="import-row-meta">
        <StatusPill status={row.duplicate_status} />
        <span className="import-row-status">{rowStatusCopy[row.row_status]}</span>
        {isConfirmedSuspicious && <span>已确认导入</span>}
        <span>{row.direction}</span>
        {row.occurred_at && <span>{row.occurred_at.replace('T', ' ').slice(0, 16)}</span>}
      </div>

      {(row.error || row.suspicious_reason || status.detail) && (
        <div className="import-row-message">
          {row.error ? (
            <>
              <CircleAlert size={14} />
              <span>
                {row.error.code}：{row.error.message}
              </span>
            </>
          ) : row.suspicious_reason ? (
            <>
              <Clock3 size={14} />
              <span>{row.suspicious_reason}</span>
            </>
          ) : (
            <>
              <CheckCircle2 size={14} />
              <span>{status.detail}</span>
            </>
          )}
        </div>
      )}

      {(row.suggestion_reason || suggestionDetail) && (
        <div className="import-rule-suggestion">
          <Sparkles size={14} />
          <span>
            {row.suggestion_reason || '导入规则已命中'}
            {suggestionDetail && <small>{suggestionDetail}</small>}
          </span>
        </div>
      )}

      <div className="import-row-actions">
        {canSkip && (
          <button type="button" className="btn-secondary" onClick={onSkip} disabled={disabled}>
            <Ban size={14} />
            跳过
          </button>
        )}
        {canRestore && (
          <button type="button" className="btn-secondary" onClick={onRestore} disabled={disabled}>
            <RefreshCw size={14} />
            恢复
          </button>
        )}
        {canConfirmImport && (
          <button type="button" className="btn-secondary" onClick={onConfirmImport} disabled={disabled}>
            <CheckCircle2 size={14} />
            确认导入
          </button>
        )}
      </div>
    </article>
  );
}

function buildRulePayload(form: ImportRuleForm): ImportRuleUpsertPayload | null {
  const pattern = form.pattern.trim();
  const tagIds = Array.from(new Set(form.tag_ids.filter(Boolean)));
  if (!pattern || (!form.category_id && !form.account_id && tagIds.length === 0)) {
    return null;
  }
  const priority = Number.parseInt(form.priority || '100', 10);
  return {
    name: form.name.trim() || pattern,
    match_type: form.match_type,
    pattern,
    priority: Number.isFinite(priority) ? priority : 100,
    result: {
      category_id: form.category_id || undefined,
      account_id: form.account_id || undefined,
      tag_ids: tagIds,
      visibility: 'private',
    },
  };
}

function matchTypeLabel(matchType: ImportRuleMatchType) {
  return matchTypeOptions.find((option) => option.value === matchType)?.label || matchType;
}

function describeRuleResult(rule: ImportRule, categories: MetadataItem[], accounts: MetadataItem[], tags: MetadataItem[]) {
  const parts = [
    rule.result.category_id ? `分类 ${metadataName(categories, rule.result.category_id)}` : '',
    rule.result.account_id ? `账户 ${metadataName(accounts, rule.result.account_id)}` : '',
    rule.result.tag_ids?.length ? `标签 ${rule.result.tag_ids.map((id) => metadataName(tags, id)).join('、')}` : '',
  ].filter(Boolean);
  return parts.length > 0 ? parts.join(' · ') : '仅记录命中解释';
}

function describeRuleMetadataWarning(rule: ImportRule, categories: MetadataItem[], accounts: MetadataItem[], tags: MetadataItem[]) {
  const issues = [
    rule.result.category_id ? metadataIssue(categories, rule.result.category_id, '分类') : '',
    rule.result.account_id ? metadataIssue(accounts, rule.result.account_id, '账户') : '',
    ...(rule.result.tag_ids || []).map((id) => metadataIssue(tags, id, '标签')),
  ].filter(Boolean);
  return issues.length > 0 ? `需处理：${issues.join('、')}` : '';
}

function describeRowSuggestion(row: ImportPreviewRow, categories: MetadataItem[], accounts: MetadataItem[], tags: MetadataItem[]) {
  const parts = [
    row.suggested_category_id ? `分类 ${metadataName(categories, row.suggested_category_id)}` : '',
    row.suggested_account_id ? `账户 ${metadataName(accounts, row.suggested_account_id)}` : '',
    row.suggested_tag_ids?.length ? `标签 ${row.suggested_tag_ids.map((id) => metadataName(tags, id)).join('、')}` : '',
  ].filter(Boolean);
  return parts.length > 0 ? `建议：${parts.join(' · ')}` : '';
}

function metadataName(items: MetadataItem[], id: string) {
  return items.find((item) => item.id === id)?.name || id.slice(0, 8);
}

function metadataIssue(items: MetadataItem[], id: string, label: string) {
  const item = items.find((candidate) => candidate.id === id);
  if (!item) {
    return `${label} ${id.slice(0, 8)} 不可用`;
  }
  if (item.is_archived) {
    return `${label} ${item.name} 已归档`;
  }
  return '';
}

function StatusPill({ status }: { status: ImportDuplicateStatus }) {
  const copy = duplicateStatusCopy[status];
  return <span className={`import-status-pill tone-${copy.tone}`}>{copy.label}</span>;
}

function buildSummary(batch: ImportPreviewBatch | null) {
  return [
    { label: '总行数', value: batch?.total_rows ?? 0, tone: 'neutral' },
    { label: '新增', value: batch?.new_rows ?? 0, tone: 'new' },
    { label: '疑似', value: batch?.suspicious_rows ?? 0, tone: 'suspicious' },
    { label: '错误', value: batch?.invalid_rows ?? 0, tone: 'invalid' },
    { label: '跳过', value: batch?.skipped_rows ?? 0, tone: 'duplicate' },
  ];
}

function buildCommitSummary(batch: ImportPreviewBatch | null) {
  if (!batch) {
    return {
      importableCount: 0,
      skippedCount: 0,
      unconfirmedSuspiciousCount: 0,
      invalidOpenCount: 0,
      blockingCount: 0,
    };
  }

  const importableCount = batch.rows.filter((row) =>
    row.row_status !== 'skipped' &&
    row.row_status !== 'failed' &&
    row.target_transaction_type !== 'skipped' &&
    row.duplicate_status !== 'duplicate' &&
    row.duplicate_status !== 'invalid'
  ).length;
  const skippedCount = batch.rows.filter((row) => row.row_status === 'skipped' || row.target_transaction_type === 'skipped').length;
  const unconfirmedSuspiciousCount = batch.rows.filter((row) => row.duplicate_status === 'suspicious' && row.row_status === 'pending').length;
  const invalidOpenCount = batch.rows.filter((row) => row.duplicate_status === 'invalid' && row.row_status !== 'skipped').length;

  return {
    importableCount,
    skippedCount,
    unconfirmedSuspiciousCount,
    invalidOpenCount,
    blockingCount: unconfirmedSuspiciousCount + invalidOpenCount,
  };
}

function resolveErrorMessage(err: unknown, fallback: string) {
  if (err instanceof ApiError) {
    return err.message;
  }
  return fallback;
}
