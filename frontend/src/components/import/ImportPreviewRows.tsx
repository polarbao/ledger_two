import {
  Ban,
  CheckCircle2,
  CircleAlert,
  Clock3,
  Pencil,
  RefreshCw,
  Sparkles,
} from 'lucide-react';
import type { ImportDuplicateStatus, ImportPreviewRow, ImportRowStatus } from '../../types/imports';
import type { MetadataItem } from '../../types/metadata';
import { centsToYuan } from '../../utils/money';
import Button from '../ui/Button';
import ResponsiveDataList from '../ui/ResponsiveDataList';
import StatusChip, { type StatusChipTone } from '../ui/StatusChip';

const duplicateStatusCopy: Record<
  ImportDuplicateStatus,
  { label: string; tone: StatusChipTone; detail: string }
> = {
  new: { label: '新增', tone: 'success', detail: '可以导入' },
  duplicate: { label: '重复', tone: 'neutral', detail: '默认跳过' },
  suspicious: { label: '疑似重复', tone: 'warning', detail: '需要人工确认' },
  invalid: { label: '错误', tone: 'danger', detail: '需要跳过' },
};

const rowStatusCopy: Record<ImportRowStatus, { label: string; tone: StatusChipTone }> = {
  pending: { label: '待处理', tone: 'neutral' },
  adjusted: { label: '已调整', tone: 'info' },
  skipped: { label: '已跳过', tone: 'neutral' },
  imported: { label: '已导入', tone: 'success' },
  failed: { label: '不可用', tone: 'danger' },
};

interface ImportPreviewRowsProps {
  rows: ImportPreviewRow[];
  categories: MetadataItem[];
  accounts: MetadataItem[];
  tags: MetadataItem[];
  disabled: boolean;
  onSkip: (row: ImportPreviewRow) => void;
  onRestore: (row: ImportPreviewRow) => void;
  onConfirmImport: (row: ImportPreviewRow) => void;
  onEdit: (row: ImportPreviewRow) => void;
}

export default function ImportPreviewRows(props: ImportPreviewRowsProps) {
  return (
    <ResponsiveDataList
      className="import-preview-data"
      desktopLabel="导入预览桌面表格"
      mobileLabel="导入预览移动卡片"
      desktop={<ImportPreviewTable {...props} />}
      mobile={<ImportPreviewCards {...props} />}
    />
  );
}

function ImportPreviewTable(props: ImportPreviewRowsProps) {
  return (
    <div className="import-preview-table-shell">
      <table className="import-preview-table">
        <thead>
          <tr>
            <th>原始行</th>
            <th>流水</th>
            <th>状态</th>
            <th>规则与错误</th>
            <th className="import-preview-table__amount">金额</th>
            <th className="import-preview-table__actions">操作</th>
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row) => (
            <tr key={row.id} className={`tone-${row.duplicate_status}`}>
              <td>
                <strong>第 {row.row_number} 行</strong>
                <time dateTime={row.occurred_at}>{formatImportTime(row.occurred_at)}</time>
              </td>
              <td className="import-preview-table__title">
                <strong>{row.title || row.merchant || '未命名流水'}</strong>
                <span>{row.merchant || '未识别商户'} · {directionLabel(row.direction)}</span>
              </td>
              <td><ImportRowStatus row={row} /></td>
              <td><ImportRowExplanation row={row} {...props} /></td>
              <td className="import-preview-table__amount">¥{centsToYuan(row.amount_cents)}</td>
              <td><ImportRowActions row={row} {...props} compact /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ImportPreviewCards(props: ImportPreviewRowsProps) {
  return (
    <div className="import-row-list">
      {props.rows.map((row) => (
        <article key={row.id} className={`import-row-card tone-${row.duplicate_status}`}>
          <div className="import-row-card__top">
            <div>
              <span className="import-row-number">第 {row.row_number} 行 · {formatImportTime(row.occurred_at)}</span>
              <h3>{row.title || row.merchant || '未命名流水'}</h3>
              <p>{row.merchant || '未识别商户'} · {directionLabel(row.direction)}</p>
            </div>
            <strong className="import-row-amount">¥{centsToYuan(row.amount_cents)}</strong>
          </div>
          <ImportRowStatus row={row} />
          <ImportRowExplanation row={row} {...props} />
          <ImportRowActions row={row} {...props} />
        </article>
      ))}
    </div>
  );
}

function ImportRowStatus({ row }: { row: ImportPreviewRow }) {
  const duplicate = duplicateStatusCopy[row.duplicate_status];
  const rowStatus = rowStatusCopy[row.row_status];
  return (
    <div className="import-row-statuses">
      <StatusChip tone={duplicate.tone}>{duplicate.label}</StatusChip>
      <StatusChip tone={rowStatus.tone}>{rowStatus.label}</StatusChip>
      {row.duplicate_status === 'suspicious' && row.row_status === 'adjusted' ? (
        <StatusChip tone="success">已确认导入</StatusChip>
      ) : null}
    </div>
  );
}

function ImportRowExplanation({
  row,
  categories,
  accounts,
  tags,
}: Pick<ImportPreviewRowsProps, 'categories' | 'accounts' | 'tags'> & { row: ImportPreviewRow }) {
  const status = duplicateStatusCopy[row.duplicate_status];
  const suggestion = describeRowSuggestion(row, categories, accounts, tags);
  return (
    <div className="import-row-explanations">
      <div className={`import-row-message ${row.error ? 'is-danger' : ''}`}>
        {row.error ? <CircleAlert size={15} /> : row.suspicious_reason ? <Clock3 size={15} /> : <CheckCircle2 size={15} />}
        <span>{row.error ? `${row.error.code}：${row.error.message}` : row.suspicious_reason || status.detail}</span>
      </div>
      {row.suggestion_reason || suggestion ? (
        <div className="import-rule-suggestion">
          <Sparkles size={15} />
          <span>
            {row.suggestion_reason || '导入规则已命中'}
            {suggestion ? <small>{suggestion}</small> : null}
          </span>
        </div>
      ) : null}
    </div>
  );
}

function ImportRowActions({
  row,
  disabled,
  onSkip,
  onRestore,
  onConfirmImport,
  onEdit,
  compact = false,
}: Pick<ImportPreviewRowsProps, 'disabled' | 'onSkip' | 'onRestore' | 'onConfirmImport' | 'onEdit'> & {
  row: ImportPreviewRow;
  compact?: boolean;
}) {
  const skipped = row.row_status === 'skipped';
  const canRestore = skipped && row.duplicate_status !== 'invalid' && row.duplicate_status !== 'duplicate';
  const canSkip = !skipped && row.row_status !== 'imported';
  const canConfirm = row.duplicate_status === 'suspicious' && row.row_status === 'pending';
  const canEdit = row.row_status !== 'imported'
    && row.duplicate_status !== 'invalid'
    && row.duplicate_status !== 'duplicate'
    && !skipped;
  const iconOnly = compact;

  return (
    <div className="import-row-actions">
      {canEdit ? (
        <Button
          variant="ghost"
          iconOnly={iconOnly}
          startIcon={iconOnly ? undefined : <Pencil size={16} />}
          aria-label={`调整第 ${row.row_number} 行`}
          title="调整分类、账户、标签和可见性"
          onClick={() => onEdit(row)}
          disabled={disabled}
        >
          {iconOnly ? <Pencil size={17} /> : '调整'}
        </Button>
      ) : null}
      {canConfirm ? (
        <Button
          variant="secondary"
          iconOnly={iconOnly}
          startIcon={iconOnly ? undefined : <CheckCircle2 size={16} />}
          aria-label={`确认导入第 ${row.row_number} 行`}
          title="确认该疑似重复行仍需导入"
          onClick={() => onConfirmImport(row)}
          disabled={disabled}
        >
          {iconOnly ? <CheckCircle2 size={17} /> : '确认导入'}
        </Button>
      ) : null}
      {canSkip ? (
        <Button
          variant="ghost"
          iconOnly={iconOnly}
          startIcon={iconOnly ? undefined : <Ban size={16} />}
          aria-label={`跳过第 ${row.row_number} 行`}
          title={row.duplicate_status === 'invalid' ? '跳过此错误行' : '跳过该行'}
          onClick={() => onSkip(row)}
          disabled={disabled}
        >
          {iconOnly ? <Ban size={17} /> : row.duplicate_status === 'invalid' ? '跳过错误行' : '跳过'}
        </Button>
      ) : null}
      {canRestore ? (
        <Button
          variant="ghost"
          iconOnly={iconOnly}
          startIcon={iconOnly ? undefined : <RefreshCw size={16} />}
          aria-label={`恢复第 ${row.row_number} 行`}
          title="恢复为待处理"
          onClick={() => onRestore(row)}
          disabled={disabled}
        >
          {iconOnly ? <RefreshCw size={17} /> : '恢复'}
        </Button>
      ) : null}
    </div>
  );
}

function describeRowSuggestion(
  row: ImportPreviewRow,
  categories: MetadataItem[],
  accounts: MetadataItem[],
  tags: MetadataItem[],
) {
  const parts = [
    row.suggested_category_id ? `分类 ${metadataName(categories, row.suggested_category_id)}` : '',
    row.suggested_account_id ? `账户 ${metadataName(accounts, row.suggested_account_id)}` : '',
    row.suggested_tag_ids?.length
      ? `标签 ${row.suggested_tag_ids.map((id) => metadataName(tags, id)).join('、')}`
      : '',
  ].filter(Boolean);
  return parts.length > 0 ? `建议：${parts.join(' · ')}` : '';
}

function metadataName(items: MetadataItem[], id: string) {
  const item = items.find((candidate) => candidate.id === id);
  if (!item) return id.slice(0, 8);
  return item.is_archived ? `${item.name}（已归档）` : item.name;
}

function directionLabel(direction: ImportPreviewRow['direction']) {
  return {
    expense: '支出',
    income: '收入',
    refund: '退款',
    transfer: '转账',
    unknown: '待判断',
  }[direction];
}

function formatImportTime(value?: string) {
  return value ? value.replace('T', ' ').slice(0, 16) : '时间未识别';
}
