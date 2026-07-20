import { AlertTriangle, Brain, SlidersHorizontal } from 'lucide-react';
import { useState } from 'react';
import type {
  ImportLearnSourceScope,
  ImportPreviewRow,
  UpdateImportRowPayload,
} from '../../types/imports';
import type { MetadataItem } from '../../types/metadata';
import { centsToYuan } from '../../utils/money';
import BottomSheet from '../ui/BottomSheet';
import Button from '../ui/Button';
import StatusChip from '../ui/StatusChip';
import {
  buildImportRowUpdatePayload,
  canRememberImportMerchant,
  createImportRowEditorDraft,
  IMPORT_TAG_LIMIT,
  toggleImportTag,
  type ImportRowEditorDraft,
} from './importRowEditorModel';

interface ImportRowEditorProps {
  row: ImportPreviewRow;
  categories: MetadataItem[];
  accounts: MetadataItem[];
  tags: MetadataItem[];
  canLearnMerchant: boolean;
  saving: boolean;
  onSave: (
    payload: UpdateImportRowPayload,
    learning?: { remember: boolean; sourceScope: ImportLearnSourceScope },
  ) => void;
  onClose: () => void;
}

export default function ImportRowEditor({
  row,
  categories,
  accounts,
  tags,
  canLearnMerchant,
  saving,
  onSave,
  onClose,
}: ImportRowEditorProps) {
  const activeCategories = categories.filter((item) => !item.is_archived);
  const activeAccounts = accounts.filter((item) => !item.is_archived);
  const activeTags = tags.filter((item) => !item.is_archived);
  const [draft, setDraft] = useState<ImportRowEditorDraft>(() => (
    createImportRowEditorDraft(row, categories, accounts, tags)
  ));
  const [rememberMerchant, setRememberMerchant] = useState(false);
  const [sourceScope, setSourceScope] = useState<ImportLearnSourceScope>('current_source');
  const selectedTags = new Set(draft.tagIds);
  const canRemember = canLearnMerchant
    && canRememberImportMerchant(row)
    && draft.targetTransactionType !== 'skipped';

  const toggleTag = (tagId: string) => {
    setDraft((current) => ({
      ...current,
      tagIds: selectedTags.has(tagId)
        ? current.tagIds.filter((id) => id !== tagId)
        : toggleImportTag(current.tagIds, tagId),
    }));
  };

  return (
    <BottomSheet
      open
      title={`调整第 ${row.row_number} 行`}
      description={`${row.merchant || row.title || '未识别商户'} · ¥${centsToYuan(row.amount_cents)}`}
      closeOnBackdrop={!saving}
      onClose={onClose}
      footer={(
        <div className="import-row-editor__footer">
          <Button variant="secondary" onClick={onClose} disabled={saving}>取消</Button>
          <Button
            variant="primary"
            startIcon={<SlidersHorizontal size={17} />}
            isLoading={saving}
            onClick={() => onSave(
              buildImportRowUpdatePayload(draft),
              canLearnMerchant ? { remember: rememberMerchant, sourceScope } : undefined,
            )}
          >
            保存调整
          </Button>
        </div>
      )}
    >
      <div className="import-row-editor">
        <div className="import-row-editor__context">
          <div>
            <span>原始标题</span>
            <strong>{row.title || '未命名流水'}</strong>
          </div>
          <div>
            <span>交易时间</span>
            <strong>{formatImportTime(row.occurred_at)}</strong>
          </div>
          <div>
            <span>识别方向</span>
            <strong>{directionLabel(row.direction)}</strong>
          </div>
        </div>

        {row.suspicious_reason ? (
          <div className="import-row-editor__warning">
            <AlertTriangle size={17} />
            <span>{row.suspicious_reason}</span>
          </div>
        ) : null}

        <label className="import-field">
          <span>记账类型</span>
          <select
            value={draft.targetTransactionType}
            onChange={(event) => setDraft({
              ...draft,
              targetTransactionType: event.target.value as ImportRowEditorDraft['targetTransactionType'],
            })}
          >
            <option value="expense">支出</option>
            <option value="income">收入</option>
            <option value="skipped">跳过，不生成账单</option>
          </select>
        </label>

        {draft.targetTransactionType !== 'skipped' ? (
          <>
            <div className="import-row-editor__two-column">
              <label className="import-field">
                <span>分类</span>
                <select
                  value={draft.categoryId}
                  onChange={(event) => setDraft({ ...draft, categoryId: event.target.value })}
                >
                  <option value="">不指定分类</option>
                  {activeCategories.map((item) => (
                    <option key={item.id} value={item.id}>{item.name}</option>
                  ))}
                </select>
              </label>
              <label className="import-field">
                <span>支付账户</span>
                <select
                  value={draft.accountId}
                  onChange={(event) => setDraft({ ...draft, accountId: event.target.value })}
                >
                  <option value="">不指定账户</option>
                  {activeAccounts.map((item) => (
                    <option key={item.id} value={item.id}>{item.name}</option>
                  ))}
                </select>
              </label>
            </div>

            <fieldset className="import-row-editor__visibility">
              <legend>可见范围</legend>
              {([
                ['private', '仅自己'],
                ['partner_readable', '对方可读'],
              ] as const).map(([value, label]) => (
                <label key={value}>
                  <input
                    type="radio"
                    name={`import-visibility-${row.id}`}
                    value={value}
                    checked={draft.visibility === value}
                    onChange={() => setDraft({ ...draft, visibility: value })}
                  />
                  <span>{label}</span>
                </label>
              ))}
            </fieldset>

            <fieldset className="import-row-editor__tags">
              <legend>标签 <span>{draft.tagIds.length}/{IMPORT_TAG_LIMIT}</span></legend>
              {activeTags.length === 0 ? (
                <span className="import-row-editor__empty-tags">暂无可用标签</span>
              ) : activeTags.map((tag) => (
                <label key={tag.id} className={selectedTags.has(tag.id) ? 'is-selected' : ''}>
                  <input
                    type="checkbox"
                    checked={selectedTags.has(tag.id)}
                    disabled={!selectedTags.has(tag.id) && draft.tagIds.length >= IMPORT_TAG_LIMIT}
                    onChange={() => toggleTag(tag.id)}
                  />
                  <span>{tag.name}</span>
                </label>
              ))}
            </fieldset>

            {canRemember ? (
              <fieldset className="import-row-editor__remember">
                <legend>长期规则</legend>
                <label className={rememberMerchant ? 'is-selected' : ''}>
                  <input
                    type="checkbox"
                    checked={rememberMerchant}
                    onChange={(event) => setRememberMerchant(event.target.checked)}
                  />
                  <Brain size={17} aria-hidden="true" />
                  <span>记住此商户，下次导入继续使用本次分类与标签</span>
                </label>
                {rememberMerchant ? (
                  <div className="import-row-editor__learn-scope" role="group" aria-label="商户规则适用来源">
                    <button
                      type="button"
                      aria-pressed={sourceScope === 'current_source'}
                      onClick={() => setSourceScope('current_source')}
                    >
                      仅当前来源
                    </button>
                    <button
                      type="button"
                      aria-pressed={sourceScope === 'all_sources'}
                      onClick={() => setSourceScope('all_sources')}
                    >
                      所有账单来源
                    </button>
                  </div>
                ) : null}
              </fieldset>
            ) : null}
          </>
        ) : (
          <StatusChip tone="warning">保存后该行计入跳过，不写入正式账单</StatusChip>
        )}
      </div>
    </BottomSheet>
  );
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
  return value ? value.replace('T', ' ').slice(0, 16) : '未识别';
}
