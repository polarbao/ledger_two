import {
  Ban,
  CheckCircle2,
  CircleAlert,
  Clock3,
  CopyCheck,
  Pencil,
  RefreshCw,
  Sparkles,
} from 'lucide-react';
import type {
  ImportClassificationStatus,
  ImportDuplicateStatus,
  ImportPreviewRow,
  ImportRowStatus,
} from '../../types/imports';
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

const classificationStatusCopy: Record<
  ImportClassificationStatus,
  { label: string; tone: StatusChipTone }
> = {
  auto_selected: { label: '已自动选择', tone: 'success' },
  suggested: { label: '待接受建议', tone: 'info' },
  fallback: { label: '兜底分类', tone: 'warning' },
  manual: { label: '手工调整', tone: 'accent' },
  bulk: { label: '批量调整', tone: 'accent' },
  conflict: { label: '规则冲突', tone: 'danger' },
  unresolved: { label: '未识别', tone: 'neutral' },
};

interface ImportPreviewRowsProps {
  rows: ImportPreviewRow[];
  categories: MetadataItem[];
  accounts: MetadataItem[];
  tags: MetadataItem[];
  disabled: boolean;
  selectedRowIds: ReadonlySet<string>;
  onToggleSelect: (row: ImportPreviewRow) => void;
  onAcceptSuggestion: (row: ImportPreviewRow) => void;
  onApplySameMerchant: (row: ImportPreviewRow) => void;
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
            <th className="import-preview-table__select"><span className="sr-only">选择</span></th>
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
              <td className="import-preview-table__select">
                <ImportRowSelector row={row} {...props} />
              </td>
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
              <span className="import-row-number">
                <ImportRowSelector row={row} {...props} />
                第 {row.row_number} 行 · {formatImportTime(row.occurred_at)}
              </span>
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
      {row.classification ? (
        <StatusChip
          tone={classificationStatusCopy[row.classification.status].tone}
          aria-label={`分类状态：${classificationStatusCopy[row.classification.status].label}`}
        >
          {classificationStatusCopy[row.classification.status].label}
        </StatusChip>
      ) : null}
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
  const finalResult = describeFinalClassification(row, categories, accounts, tags);
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
      {row.classification ? (
        <div className={`import-classification-explanation tone-${row.classification.status}`}>
          <strong>{finalResult || '尚未确定分类'}</strong>
          <span>{row.classification.reason_text || classificationReason(row)}</span>
          <small>{classificationSourceLabel(row)} · 置信度 {confidenceLabel(row.classification.confidence)}</small>
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
  onAcceptSuggestion,
  onApplySameMerchant,
  compact = false,
}: Pick<
  ImportPreviewRowsProps,
  'disabled' | 'onSkip' | 'onRestore' | 'onConfirmImport' | 'onEdit' | 'onAcceptSuggestion' | 'onApplySameMerchant'
> & {
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
      {row.classification?.status === 'suggested' && canEdit ? (
        <Button
          variant="secondary"
          iconOnly={iconOnly}
          startIcon={iconOnly ? undefined : <CopyCheck size={16} />}
          aria-label={`接受第 ${row.row_number} 行的分类建议`}
          title="接受该行已保存的分类和标签建议，不会提交批次"
          onClick={() => onAcceptSuggestion(row)}
          disabled={disabled}
        >
          {iconOnly ? <CopyCheck size={17} /> : '接受建议'}
        </Button>
      ) : null}
      {row.merchant.trim() && canEdit ? (
        <Button
          variant="ghost"
          iconOnly={iconOnly}
          aria-label={`应用第 ${row.row_number} 行设置到相同商户`}
          title="应用到本批次内相同商户"
          onClick={() => onApplySameMerchant(row)}
          disabled={disabled}
        >
          {iconOnly ? <Sparkles size={17} /> : '相同商户'}
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

function ImportRowSelector({
  row,
  selectedRowIds,
  onToggleSelect,
  disabled,
}: Pick<ImportPreviewRowsProps, 'selectedRowIds' | 'onToggleSelect' | 'disabled'> & { row: ImportPreviewRow }) {
  const selectable = row.row_status !== 'imported'
    && row.row_status !== 'skipped'
    && row.duplicate_status !== 'duplicate'
    && row.duplicate_status !== 'invalid';
  return (
    <input
      className="import-row-selector"
      type="checkbox"
      aria-label={`选择第 ${row.row_number} 行`}
      checked={selectedRowIds.has(row.id)}
      disabled={disabled || !selectable}
      onChange={() => onToggleSelect(row)}
    />
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

function describeFinalClassification(
  row: ImportPreviewRow,
  categories: MetadataItem[],
  accounts: MetadataItem[],
  tags: MetadataItem[],
) {
  const categoryId = row.selected_category_id || row.suggested_category_id;
  const accountId = row.selected_account_id || row.suggested_account_id;
  const tagIds = row.selected_tag_ids?.length ? row.selected_tag_ids : row.suggested_tag_ids;
  return [
    categoryId ? `分类 ${metadataName(categories, categoryId)}` : '',
    accountId ? `账户 ${metadataName(accounts, accountId)}` : '',
    tagIds?.length ? `标签 ${tagIds.map((id) => metadataName(tags, id)).join('、')}` : '',
  ].filter(Boolean).join(' · ');
}

function classificationReason(row: ImportPreviewRow) {
  if (row.classification.status === 'conflict') return '多条规则给出了不同结果，请手工确认。';
  if (row.classification.status === 'fallback') return '没有更可靠的匹配，暂时使用兜底分类。';
  if (row.classification.status === 'suggested') return '已有建议，接受后才会成为最终分类。';
  return '当前结果已保存在预览行中，重新分类不会覆盖手工或批量调整。';
}

function classificationSourceLabel(row: ImportPreviewRow) {
  return ({
    manual: '手工选择',
    bulk: '批量调整',
    user_rule: '用户规则',
    learned_rule: '已记住商户',
    builtin: '内置匹配',
    fallback: '兜底规则',
  } as const)[row.classification.source || 'fallback'];
}

function confidenceLabel(confidence: ImportPreviewRow['classification']['confidence']) {
  return { high: '高', medium: '中', low: '低', none: '无' }[confidence];
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
