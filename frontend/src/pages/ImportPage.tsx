import { useMemo, useRef, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  CheckCircle2,
  FileSpreadsheet,
  FileWarning,
  RefreshCw,
  ShieldCheck,
  Upload,
  X,
} from 'lucide-react';
import { importsApi } from '../api/imports.api';
import { metadataApi } from '../api/metadata.api';
import { queryKeys } from '../api/queryKeys';
import { systemApi } from '../api/system.api';
import ImportPreviewRows from '../components/import/ImportPreviewRows';
import ImportRowEditor from '../components/import/ImportRowEditor';
import ImportRuleManager from '../components/import/ImportRuleManager';
import {
  buildImportRulePayload,
  createDefaultImportRuleForm,
  type ImportRuleForm,
  type ImportRuleStatusFilter,
} from '../components/import/importRuleModel';
import Button from '../components/ui/Button';
import ConfirmDialog from '../components/ui/ConfirmDialog';
import SegmentedControl from '../components/ui/SegmentedControl';
import StatePanel from '../components/ui/StatePanel';
import StatusChip, { type StatusChipTone } from '../components/ui/StatusChip';
import { useLedgerStore } from '../stores/ledger.store';
import type {
  ImportCommitResult,
  ImportPreviewBatch,
  ImportPreviewRow,
  ImportRule,
  ImportRuleUpsertPayload,
  ImportSourceType,
  UpdateImportRowPayload,
} from '../types/imports';
import {
  buildImportCommitSummary,
  defaultImportRowFilter,
  filterImportRows,
  getImportFileAccept,
  getImportSourceDescription,
  IMPORT_ROW_FILTER_LABELS,
  resolveImportErrorMessage,
  type ImportRowFilter,
  validateImportFile,
} from './importPageState';
import './ImportPage.css';

const sourceOptions: Array<{ value: ImportSourceType; label: string }> = [
  { value: 'wechat', label: '微信账单' },
  { value: 'alipay', label: '支付宝账单' },
  { value: 'generic', label: '通用模板' },
];

export default function ImportPage() {
  const activeRole = useLedgerStore((state) => state.activeRole);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);

  return (
    <ImportWorkspace
      key={activeLedgerId || 'no-active-ledger'}
      activeLedgerId={activeLedgerId}
      activeRole={activeRole}
    />
  );
}

function ImportWorkspace({
  activeLedgerId,
  activeRole,
}: {
  activeLedgerId: string | null;
  activeRole: string | null;
}) {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const rowListRef = useRef<HTMLDivElement | null>(null);
  const [sourceType, setSourceType] = useState<ImportSourceType>('wechat');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [batch, setBatch] = useState<ImportPreviewBatch | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [commitResult, setCommitResult] = useState<ImportCommitResult | null>(null);
  const [editingRow, setEditingRow] = useState<ImportPreviewRow | null>(null);
  const [ruleToArchive, setRuleToArchive] = useState<ImportRule | null>(null);
  const [ruleForm, setRuleForm] = useState<ImportRuleForm>(() => createDefaultImportRuleForm());
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null);
  const [ruleStatusFilter, setRuleStatusFilter] = useState<ImportRuleStatusFilter>('all');
  const [rowFilter, setRowFilter] = useState<ImportRowFilter>('all');
  const isOwner = activeRole === 'owner';

  const { data: health } = useQuery({
    queryKey: queryKeys.system.health,
    queryFn: ({ signal }) => systemApi.getHealth(signal),
    staleTime: Number.POSITIVE_INFINITY,
  });
  const xlsxEnabled = health?.import_xlsx_enabled ?? false;

  const { data: importRules = [] } = useQuery({
    queryKey: queryKeys.importRules(activeLedgerId),
    queryFn: ({ signal }) => importsApi.listRules('all', signal),
    enabled: isOwner && Boolean(activeLedgerId),
  });
  const { data: categories = [] } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, 'categories'),
    queryFn: ({ signal }) => metadataApi.list('categories', true, signal),
    enabled: isOwner && Boolean(activeLedgerId),
  });
  const { data: accounts = [] } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, 'accounts'),
    queryFn: ({ signal }) => metadataApi.list('accounts', true, signal),
    enabled: isOwner && Boolean(activeLedgerId),
  });
  const { data: tags = [] } = useQuery({
    queryKey: queryKeys.metadata.list(activeLedgerId, 'tags'),
    queryFn: ({ signal }) => metadataApi.list('tags', true, signal),
    enabled: isOwner && Boolean(activeLedgerId),
  });

  const summary = useMemo(() => buildSummary(batch), [batch]);
  const commitSummary = useMemo(() => buildImportCommitSummary(batch), [batch]);
  const visibleRows = useMemo(() => filterImportRows(batch?.rows || [], rowFilter), [batch?.rows, rowFilter]);
  const canOpenCommit = isOwner && Boolean(batch) && batch?.status === 'ready';

  const previewMutation = useMutation({
    mutationFn: (file: File) => importsApi.preview({ file, sourceType }),
    onSuccess: (data) => {
      setBatch(data);
      setRowFilter(defaultImportRowFilter(data.rows));
      setCommitResult(null);
      setErrorMsg(null);
      window.requestAnimationFrame(() => rowListRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }));
    },
    onError: (err: unknown) => {
      setBatch(null);
      setErrorMsg(resolveImportErrorMessage(err, '生成导入预览失败，请检查来源和账单文件格式'));
    },
  });

  const updateRowMutation = useMutation({
    mutationFn: ({ row, payload }: { row: ImportPreviewRow; payload: UpdateImportRowPayload }) => (
      importsApi.updateRow(batch?.id || '', row.id, payload)
    ),
    onSuccess: (data) => {
      setBatch(data);
      setEditingRow(null);
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setErrorMsg(resolveImportErrorMessage(err, '更新预览行失败'));
    },
  });

  const commitMutation = useMutation({
    mutationFn: async () => {
      if (!batch) throw new Error('missing batch');
      const result = await importsApi.commit(batch.id);
      const latestBatch = await importsApi.getBatch(batch.id);
      return { result, latestBatch };
    },
    onSuccess: ({ result, latestBatch }) => {
      setBatch(latestBatch);
      setCommitResult(result);
      setIsConfirmOpen(false);
      setErrorMsg(null);
      void queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(activeLedgerId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(activeLedgerId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.reports.root(activeLedgerId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.metadata.root(activeLedgerId) });
    },
    onError: (err: unknown) => {
      setIsConfirmOpen(false);
      setErrorMsg(resolveImportErrorMessage(err, '导入提交失败，当前批次未写入正式账单'));
      if (batch) void importsApi.getBatch(batch.id).then(setBatch).catch(() => undefined);
    },
  });

  const createRuleMutation = useMutation({
    mutationFn: (payload: ImportRuleUpsertPayload) => importsApi.createRule(payload),
    onSuccess: () => {
      setRuleForm(createDefaultImportRuleForm());
      void queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
      setErrorMsg(null);
    },
    onError: (err: unknown) => setErrorMsg(resolveImportErrorMessage(err, '创建导入规则失败')),
  });

  const updateRuleMutation = useMutation({
    mutationFn: ({ ruleId, payload }: { ruleId: string; payload: ImportRuleUpsertPayload }) => (
      importsApi.updateRule(ruleId, payload)
    ),
    onSuccess: () => {
      setRuleForm(createDefaultImportRuleForm());
      setEditingRuleId(null);
      void queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
      setErrorMsg(null);
    },
    onError: (err: unknown) => setErrorMsg(resolveImportErrorMessage(err, '更新导入规则失败')),
  });

  const archiveRuleMutation = useMutation({
    mutationFn: (ruleId: string) => importsApi.archiveRule(ruleId),
    onSuccess: () => {
      setRuleToArchive(null);
      void queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setRuleToArchive(null);
      setErrorMsg(resolveImportErrorMessage(err, '归档导入规则失败'));
    },
  });

  const restoreRuleMutation = useMutation({
    mutationFn: (ruleId: string) => importsApi.restoreRule(ruleId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.importRules(activeLedgerId) });
      setErrorMsg(null);
    },
    onError: (err: unknown) => setErrorMsg(resolveImportErrorMessage(err, '恢复导入规则失败')),
  });

  const handleFile = (file: File) => {
    const validationError = validateImportFile(sourceType, file.name, xlsxEnabled);
    if (validationError) {
      setErrorMsg(validationError);
      return;
    }
    setSelectedFile(file);
    setBatch(null);
    setRowFilter('all');
    previewMutation.mutate(file);
  };

  const handleReset = () => {
    setSelectedFile(null);
    setBatch(null);
    setCommitResult(null);
    setErrorMsg(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const handleSourceChange = (nextSource: ImportSourceType) => {
    setSourceType(nextSource);
    handleReset();
  };

  const handleOpenCommit = () => {
    if (!batch) return;
    if (commitSummary.blockingCount > 0) {
      setErrorMsg(`还有 ${commitSummary.blockingCount} 条错误或疑似重复流水需要处理`);
      setRowFilter('needs_attention');
      window.requestAnimationFrame(() => rowListRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }));
      return;
    }
    setIsConfirmOpen(true);
  };

  const handleSubmitRule = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = buildImportRulePayload(ruleForm);
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
    setRuleForm(createDefaultImportRuleForm());
  };

  return (
    <div className="page-content animate-fade-in import-workbench">
      <header className="import-page-header">
        <div className="import-page-header__icon"><FileSpreadsheet size={24} /></div>
        <div className="import-page-header__copy">
          <span className="import-panel__eyebrow">数据导入</span>
          <h1>账单导入工作台</h1>
          <p>先审阅预览批次，再确认写入当前账本。</p>
        </div>
        <div className="import-page-header__status">
          <StatusChip tone={isOwner ? 'success' : 'neutral'}>{isOwner ? 'Owner 可操作' : '当前角色只读'}</StatusChip>
          <StatusChip tone="info">Preview 不写正式账单</StatusChip>
        </div>
      </header>

      {errorMsg ? (
        <div className="import-workbench__notice" role="alert">
          <AlertTriangle size={18} />
          <span>{errorMsg}</span>
          <Button
            variant="ghost"
            iconOnly
            aria-label="关闭错误提示"
            title="关闭错误提示"
            onClick={() => setErrorMsg(null)}
          >
            <X size={18} />
          </Button>
        </div>
      ) : null}

      {!activeLedgerId ? (
        <section className="import-panel">
          <StatePanel
            tone="warning"
            icon={<FileWarning size={38} />}
            title="还没有可用账本"
            description="选择或创建账本后才能生成导入预览。"
          />
        </section>
      ) : !isOwner ? (
        <section className="import-panel">
          <StatePanel
            tone="warning"
            icon={<ShieldCheck size={38} />}
            title="当前角色不可导入"
            description="导入是批量写入操作，仅账本 Owner 可以创建、调整和提交预览批次。"
          />
        </section>
      ) : (
        <>
          <section className="import-panel import-entry" aria-labelledby="import-entry-title">
            <header className="import-panel__header">
              <div>
                <span className="import-panel__eyebrow">步骤 1</span>
                <h2 id="import-entry-title">选择来源与文件</h2>
                <p>{getImportSourceDescription(sourceType, xlsxEnabled)}</p>
              </div>
              {selectedFile ? <StatusChip tone="info">{selectedFile.name}</StatusChip> : <StatusChip>未选择文件</StatusChip>}
            </header>

            <div className="import-entry__grid">
              <div className="import-entry__source">
                <SegmentedControl
                  ariaLabel="账单来源"
                  value={sourceType}
                  options={sourceOptions}
                  onChange={handleSourceChange}
                  fullWidth
                />
                <dl className="import-format-matrix">
                  <div><dt>微信</dt><dd>{xlsxEnabled ? 'CSV / XLSX' : 'CSV'}</dd></div>
                  <div><dt>支付宝</dt><dd>CSV</dd></div>
                  <div><dt>通用模板</dt><dd>CSV</dd></div>
                </dl>
              </div>

              <label
                className={`import-upload-zone ${previewMutation.isPending ? 'is-loading' : ''}`}
                onDragOver={(event) => event.preventDefault()}
                onDrop={(event) => {
                  event.preventDefault();
                  const file = event.dataTransfer.files?.[0];
                  if (file) handleFile(file);
                }}
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  accept={getImportFileAccept(sourceType, xlsxEnabled)}
                  disabled={previewMutation.isPending}
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (file) handleFile(file);
                  }}
                />
                <span className="import-upload-zone__icon"><Upload size={25} /></span>
                <strong>{previewMutation.isPending ? '正在生成预览' : '选择或拖入账单文件'}</strong>
                <small>单批最多 2000 行；上传后只生成预览批次。</small>
              </label>
            </div>

            {selectedFile ? (
              <div className="import-entry__actions">
                <Button
                  variant="secondary"
                  startIcon={<RefreshCw size={16} />}
                  onClick={handleReset}
                  disabled={previewMutation.isPending}
                >
                  重新选择
                </Button>
              </div>
            ) : null}
          </section>

          <section ref={rowListRef} className="import-panel import-preview-panel" aria-labelledby="import-preview-title">
            <header className="import-panel__header">
              <div>
                <span className="import-panel__eyebrow">步骤 2</span>
                <h2 id="import-preview-title">审阅预览批次</h2>
                <p>{batch ? `批次 ${batch.id.slice(0, 8)} · ${batch.filename}` : '上传文件后显示解析摘要和行级状态。'}</p>
              </div>
              <StatusChip tone={batchStatusTone(batch?.status)}>{batchStatusLabel(batch?.status)}</StatusChip>
            </header>

            {!batch ? (
              <StatePanel
                icon={<FileWarning size={38} />}
                title="还没有预览批次"
                description="上传账单文件后会在这里看到行级状态和错误原因。"
              />
            ) : (
              <>
                <dl className="import-parser-summary">
                  <div><dt>文件格式</dt><dd>{batch.file_format.toUpperCase()}</dd></div>
                  {batch.parser_metadata.sheet_name ? <div><dt>工作表</dt><dd>{batch.parser_metadata.sheet_name}</dd></div> : null}
                  <div><dt>表头位置</dt><dd>第 {batch.parser_metadata.header_row_number} 行</dd></div>
                  <div><dt>识别流水</dt><dd>{batch.parser_metadata.parsed_rows} 条</dd></div>
                </dl>

                <div className={`import-safe-banner ${batch.status === 'failed' ? 'is-danger' : ''}`}>
                  <ShieldCheck size={17} />
                  <span>{batchSafetyMessage(batch.status)}</span>
                </div>

                <div className="import-summary-grid">
                  {summary.map((item) => (
                    <button
                      key={item.filter}
                      type="button"
                      className={`import-summary-card tone-${item.tone}`}
                      aria-label={`${item.label} ${item.value} 条`}
                      onClick={() => setRowFilter(item.filter)}
                    >
                      <span>{item.label}</span>
                      <strong>{item.value}</strong>
                    </button>
                  ))}
                </div>

                <SegmentedControl
                  className="import-row-filter"
                  ariaLabel="导入行状态筛选"
                  value={rowFilter}
                  onChange={setRowFilter}
                  options={buildRowFilterOptions(batch)}
                />

                {commitResult ? (
                  <div className="import-result-card" role="status">
                    <CheckCircle2 size={19} />
                    <div>
                      <strong>导入完成</strong>
                      <span>已导入 {commitResult.imported_rows} 条，跳过 {commitResult.skipped_rows} 条，失败 {commitResult.failed_rows} 条。</span>
                    </div>
                  </div>
                ) : null}

                <div className="import-row-results" aria-live="polite">
                  <div className="import-row-results__header">
                    <div>
                      <strong>{IMPORT_ROW_FILTER_LABELS[rowFilter]}</strong>
                      <span>显示 {visibleRows.length} / {batch.rows.length} 条</span>
                    </div>
                    {rowFilter !== 'all' ? (
                      <Button variant="ghost" onClick={() => setRowFilter('all')}>显示全部</Button>
                    ) : null}
                  </div>
                  {visibleRows.length === 0 ? (
                    <div className="import-filter-empty"><CheckCircle2 size={20} /><span>当前筛选下没有流水</span></div>
                  ) : (
                    <ImportPreviewRows
                      rows={visibleRows}
                      categories={categories}
                      accounts={accounts}
                      tags={tags}
                      disabled={updateRowMutation.isPending || batch.status === 'committed'}
                      onSkip={(row) => updateRowMutation.mutate({ row, payload: { row_status: 'skipped' } })}
                      onRestore={(row) => updateRowMutation.mutate({ row, payload: { row_status: 'pending' } })}
                      onConfirmImport={(row) => updateRowMutation.mutate({ row, payload: { row_status: 'adjusted' } })}
                      onEdit={setEditingRow}
                    />
                  )}
                </div>

                <div className="import-commit-bar">
                  <div>
                    <strong>{commitSummary.blockingCount > 0 ? `还有 ${commitSummary.blockingCount} 条需要处理` : '当前批次可以提交'}</strong>
                    <span>将导入 {commitSummary.importableCount} 条，跳过 {commitSummary.skippedCount} 条。</span>
                  </div>
                  <Button
                    variant={commitSummary.blockingCount > 0 ? 'secondary' : 'primary'}
                    onClick={handleOpenCommit}
                    isLoading={commitMutation.isPending}
                    disabled={!canOpenCommit || updateRowMutation.isPending}
                  >
                    {batch.status === 'committed'
                      ? '已完成导入'
                      : batch.status === 'failed'
                        ? '当前批次不可提交'
                        : commitSummary.blockingCount > 0
                          ? '查看待处理流水'
                          : '确认导入'}
                  </Button>
                </div>
              </>
            )}
          </section>

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
            onArchive={setRuleToArchive}
            onRestore={(ruleId) => restoreRuleMutation.mutate(ruleId)}
          />
        </>
      )}

      <ConfirmDialog
        open={isConfirmOpen && Boolean(batch)}
        title="确认导入账单？"
        description="系统将在单个事务中把当前批次写入正式账单；任一行失败都会回滚整批。"
        confirmLabel="确认导入"
        cancelLabel="返回预览"
        icon={<FileSpreadsheet size={22} />}
        isConfirming={commitMutation.isPending}
        confirmDisabled={commitSummary.blockingCount > 0}
        onConfirm={() => commitMutation.mutate()}
        onClose={() => setIsConfirmOpen(false)}
      >
        <div className="import-confirm-summary">
          <div><span>将导入</span><strong>{commitSummary.importableCount}</strong></div>
          <div><span>将跳过</span><strong>{commitSummary.skippedCount}</strong></div>
          <div><span>疑似未确认</span><strong>{commitSummary.unconfirmedSuspiciousCount}</strong></div>
          <div><span>错误未跳过</span><strong>{commitSummary.invalidOpenCount}</strong></div>
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={Boolean(ruleToArchive)}
        title={`归档规则“${ruleToArchive?.name || ruleToArchive?.pattern || ''}”？`}
        description="归档后该规则不再参与新预览推荐；已有预览、历史账单和手工调整不会改变。"
        confirmLabel="确认归档"
        tone="danger"
        icon={<AlertTriangle size={22} />}
        isConfirming={archiveRuleMutation.isPending}
        onConfirm={() => {
          if (ruleToArchive) archiveRuleMutation.mutate(ruleToArchive.id);
        }}
        onClose={() => setRuleToArchive(null)}
      />

      {editingRow ? (
        <ImportRowEditor
          key={editingRow.id}
          row={editingRow}
          categories={categories}
          accounts={accounts}
          tags={tags}
          saving={updateRowMutation.isPending}
          onSave={(payload) => updateRowMutation.mutate({ row: editingRow, payload })}
          onClose={() => setEditingRow(null)}
        />
      ) : null}
    </div>
  );
}

function buildSummary(batch: ImportPreviewBatch | null) {
  return [
    { label: '总行数', value: batch?.total_rows ?? 0, tone: 'neutral', filter: 'all' as const },
    { label: '新增', value: batch?.new_rows ?? 0, tone: 'success', filter: 'new' as const },
    { label: '重复', value: batch?.duplicate_rows ?? 0, tone: 'neutral', filter: 'duplicate' as const },
    { label: '疑似', value: batch?.suspicious_rows ?? 0, tone: 'warning', filter: 'suspicious' as const },
    { label: '错误', value: batch?.invalid_rows ?? 0, tone: 'danger', filter: 'invalid' as const },
    { label: '已跳过', value: batch?.skipped_rows ?? 0, tone: 'info', filter: 'skipped' as const },
  ];
}

function buildRowFilterOptions(batch: ImportPreviewBatch) {
  const needsAttention = filterImportRows(batch.rows, 'needs_attention').length;
  return [
    { value: 'all' as const, label: '全部', count: batch.rows.length },
    { value: 'needs_attention' as const, label: '需处理', count: needsAttention },
    { value: 'new' as const, label: '新增', count: batch.new_rows },
    { value: 'duplicate' as const, label: '重复', count: batch.duplicate_rows },
    { value: 'suspicious' as const, label: '疑似', count: batch.suspicious_rows },
    { value: 'invalid' as const, label: '错误', count: batch.invalid_rows },
    { value: 'skipped' as const, label: '跳过', count: batch.skipped_rows },
  ];
}

function batchStatusLabel(status?: ImportPreviewBatch['status']) {
  if (!status) return '等待上传';
  return {
    previewing: '解析中',
    ready: '待确认',
    committed: '已导入',
    failed: '失败',
    expired: '已过期',
  }[status];
}

function batchStatusTone(status?: ImportPreviewBatch['status']): StatusChipTone {
  if (status === 'ready') return 'warning';
  if (status === 'committed') return 'success';
  if (status === 'failed' || status === 'expired') return 'danger';
  return 'neutral';
}

function batchSafetyMessage(status: ImportPreviewBatch['status']) {
  if (status === 'committed') return '当前批次已写入正式账单，不能再次调整或提交。';
  if (status === 'failed') return '本次提交失败，未保留半批正式账单；请根据错误重新生成预览。';
  if (status === 'expired') return '当前预览已过期，不能提交；请重新上传文件。';
  return '当前只保存预览数据，点击确认导入前不会写入 transactions。';
}
