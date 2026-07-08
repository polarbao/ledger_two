import { useMemo, useRef, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import {
  AlertTriangle,
  Ban,
  CheckCircle2,
  CircleAlert,
  Clock3,
  FileSpreadsheet,
  FileWarning,
  Loader2,
  RefreshCw,
  ShieldCheck,
  Upload,
  X,
} from 'lucide-react';
import { ApiError } from '../api/client';
import { importsApi } from '../api/imports.api';
import { useLedgerStore } from '../stores/ledger.store';
import { centsToYuan } from '../utils/money';
import type {
  ImportDuplicateStatus,
  ImportPreviewBatch,
  ImportPreviewRow,
  ImportRowStatus,
  ImportSourceType,
} from '../types/imports';

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

export default function ImportPage() {
  const activeRole = useLedgerStore((state) => state.activeRole);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [sourceType, setSourceType] = useState<ImportSourceType>('wechat');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [batch, setBatch] = useState<ImportPreviewBatch | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const isOwner = activeRole === 'owner';

  const previewMutation = useMutation({
    mutationFn: (file: File) => importsApi.preview({ file, sourceType }),
    onSuccess: (data) => {
      setBatch(data);
      setErrorMsg(null);
    },
    onError: (err: unknown) => {
      setBatch(null);
      setErrorMsg(resolveErrorMessage(err, '生成导入预览失败，请检查来源和 CSV 文件格式'));
    },
  });

  const updateRowMutation = useMutation({
    mutationFn: ({ row, rowStatus }: { row: ImportPreviewRow; rowStatus: 'pending' | 'skipped' }) =>
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
    setErrorMsg(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="page-content animate-fade-in text-left import-workbench">
      <div className="glass-card header-banner import-workbench__hero">
        <FileSpreadsheet className="banner-icon" />
        <div>
          <h2>导入预览工作台</h2>
          <p>CSV 上传后先生成可审阅批次。Task47 阶段只做预览和行级状态处理，不写入正式账单。</p>
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
              预览阶段暂不可提交
            </button>
          </div>
        </div>

        <div className="glass-card import-preview-panel">
          <div className="import-section-title">
            <span>预览批次</span>
            <small>{batch ? `Batch ${batch.id.slice(0, 8)}` : '等待上传'}</small>
          </div>

          <div className="import-safe-banner">
            <ShieldCheck size={16} />
            <span>当前批次只保存预览数据，没有写入 transactions。</span>
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
            <button type="button" className="btn-primary" disabled>
              预览阶段暂不可提交
            </button>
          </div>

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
                  disabled={updateRowMutation.isPending}
                  onSkip={() => updateRowMutation.mutate({ row, rowStatus: 'skipped' })}
                  onRestore={() => updateRowMutation.mutate({ row, rowStatus: 'pending' })}
                />
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function ImportRowCard({
  row,
  disabled,
  onSkip,
  onRestore,
}: {
  row: ImportPreviewRow;
  disabled: boolean;
  onSkip: () => void;
  onRestore: () => void;
}) {
  const status = duplicateStatusCopy[row.duplicate_status];
  const isSkipped = row.row_status === 'skipped';
  const canRestore = isSkipped && row.duplicate_status !== 'invalid' && row.duplicate_status !== 'duplicate';
  const canSkip = !isSkipped && row.row_status !== 'failed';

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
      </div>
    </article>
  );
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

function resolveErrorMessage(err: unknown, fallback: string) {
  if (err instanceof ApiError) {
    return err.message;
  }
  return fallback;
}
