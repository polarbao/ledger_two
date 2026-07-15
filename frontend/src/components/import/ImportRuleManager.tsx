import { Archive, Loader2, RotateCcw, Save, Sparkles, X } from 'lucide-react';
import type { FormEvent } from 'react';
import type { ImportRule, ImportRuleMatchType } from '../../types/imports';
import type { MetadataItem } from '../../types/metadata';
import Button from '../ui/Button';
import SegmentedControl from '../ui/SegmentedControl';
import StatusChip from '../ui/StatusChip';
import type { ImportRuleForm, ImportRuleStatusFilter } from './importRuleModel';

const matchTypeOptions: Array<{ value: ImportRuleMatchType; label: string }> = [
  { value: 'merchant_contains', label: '商户包含' },
  { value: 'description_contains', label: '描述包含' },
  { value: 'source_account', label: '来源账户' },
  { value: 'amount_range', label: '金额区间' },
];

interface ImportRuleManagerProps {
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
  statusFilter: ImportRuleStatusFilter;
  onFormChange: (form: ImportRuleForm) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onCancelEdit: () => void;
  onEdit: (rule: ImportRule) => void;
  onStatusFilterChange: (status: ImportRuleStatusFilter) => void;
  onArchive: (rule: ImportRule) => void;
  onRestore: (ruleId: string) => void;
}

export default function ImportRuleManager({
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
}: ImportRuleManagerProps) {
  const activeRules = rules.filter((rule) => rule.status === 'active');
  const archivedRules = rules.filter((rule) => rule.status === 'archived');
  const visibleRules = rules.filter((rule) => statusFilter === 'all' || rule.status === statusFilter);
  const activeCategories = categories.filter((item) => !item.is_archived);
  const activeAccounts = accounts.filter((item) => !item.is_archived);
  const activeTags = tags.filter((item) => !item.is_archived);
  const busy = creating || updating || archiving || restoring;
  const selectedTagIds = new Set(form.tag_ids);

  const toggleTag = (tagId: string) => {
    onFormChange({
      ...form,
      tag_ids: selectedTagIds.has(tagId)
        ? form.tag_ids.filter((id) => id !== tagId)
        : [...form.tag_ids, tagId],
    });
  };

  return (
    <section className="import-panel import-rule-manager" aria-labelledby="import-rule-title">
      <header className="import-panel__header">
        <div>
          <span className="import-panel__eyebrow">自动推荐</span>
          <h2 id="import-rule-title">导入规则</h2>
          <p>规则只填充分类、账户和标签建议，不会自动提交账单。</p>
        </div>
        <StatusChip tone="success">{activeRules.length} 条启用</StatusChip>
      </header>

      <SegmentedControl
        ariaLabel="导入规则状态"
        value={statusFilter}
        onChange={onStatusFilterChange}
        options={[
          { value: 'all', label: '全部', count: rules.length },
          { value: 'active', label: '启用', count: activeRules.length },
          { value: 'archived', label: '归档', count: archivedRules.length },
        ]}
      />

      <form className="import-rule-form" onSubmit={onSubmit}>
        <label className="import-field">
          <span>规则名称</span>
          <input
            value={form.name}
            onChange={(event) => onFormChange({ ...form, name: event.target.value })}
            placeholder="例如 咖啡消费"
          />
        </label>
        <label className="import-field">
          <span>匹配方式</span>
          <select
            value={form.match_type}
            onChange={(event) => onFormChange({
              ...form,
              match_type: event.target.value as ImportRuleMatchType,
            })}
          >
            {matchTypeOptions.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </label>
        <label className="import-field import-rule-form__pattern">
          <span>匹配内容</span>
          <input
            value={form.pattern}
            onChange={(event) => onFormChange({ ...form, pattern: event.target.value })}
            placeholder="例如 星巴克"
          />
        </label>
        <label className="import-field">
          <span>推荐分类</span>
          <select
            value={form.category_id}
            onChange={(event) => onFormChange({ ...form, category_id: event.target.value })}
          >
            <option value="">不推荐分类</option>
            {activeCategories.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </label>
        <label className="import-field">
          <span>推荐账户</span>
          <select
            value={form.account_id}
            onChange={(event) => onFormChange({ ...form, account_id: event.target.value })}
          >
            <option value="">不推荐账户</option>
            {activeAccounts.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </label>
        <label className="import-field">
          <span>优先级</span>
          <input
            type="number"
            min="0"
            value={form.priority}
            onChange={(event) => onFormChange({ ...form, priority: event.target.value })}
          />
        </label>
        <fieldset className="import-rule-tag-options">
          <legend>推荐标签</legend>
          {activeTags.length === 0 ? (
            <span>暂无可用标签</span>
          ) : activeTags.map((item) => (
            <label key={item.id} className={selectedTagIds.has(item.id) ? 'is-selected' : ''}>
              <input
                type="checkbox"
                checked={selectedTagIds.has(item.id)}
                onChange={() => toggleTag(item.id)}
              />
              <span>{item.name}</span>
            </label>
          ))}
        </fieldset>
        <div className="import-rule-form__actions">
          {editingRuleId ? (
            <Button
              variant="secondary"
              startIcon={<X size={16} />}
              onClick={onCancelEdit}
              disabled={busy}
            >
              取消编辑
            </Button>
          ) : null}
          <Button
            type="submit"
            variant="primary"
            startIcon={creating || updating ? <Loader2 size={16} className="spin" /> : <Save size={16} />}
            disabled={creating || updating}
          >
            {editingRuleId ? '保存规则' : '创建规则'}
          </Button>
        </div>
      </form>

      <div className="import-rule-list">
        {visibleRules.map((rule) => {
          const metadataWarning = describeRuleMetadataWarning(rule, categories, accounts, tags);
          return (
            <article key={rule.id} className={`import-rule-card ${rule.status === 'archived' ? 'is-archived' : ''}`}>
              <div className="import-rule-card__copy">
                <div className="import-rule-card__title">
                  <strong>{rule.name || rule.pattern}</strong>
                  <StatusChip tone={rule.status === 'active' ? 'success' : 'neutral'}>
                    {rule.status === 'active' ? '启用' : '已归档'}
                  </StatusChip>
                </div>
                <span>{matchTypeLabel(rule.match_type)}「{rule.pattern}」 · 优先级 {rule.priority}</span>
                <small>{describeRuleResult(rule, categories, accounts, tags)}</small>
                {metadataWarning ? <small className="import-rule-warning">需处理：{metadataWarning}</small> : null}
              </div>
              <div className="import-rule-card__actions">
                <Button variant="ghost" onClick={() => onEdit(rule)} disabled={busy}>编辑</Button>
                {rule.status === 'active' ? (
                  <Button
                    variant="ghost"
                    startIcon={<Archive size={16} />}
                    onClick={() => onArchive(rule)}
                    disabled={busy}
                  >
                    归档
                  </Button>
                ) : (
                  <Button
                    variant="ghost"
                    startIcon={<RotateCcw size={16} />}
                    onClick={() => onRestore(rule.id)}
                    disabled={busy}
                  >
                    恢复
                  </Button>
                )}
              </div>
            </article>
          );
        })}
        {visibleRules.length === 0 ? (
          <div className="import-rule-empty">
            <Sparkles size={19} />
            <span>当前状态下没有导入规则</span>
          </div>
        ) : null}
      </div>
    </section>
  );
}

function matchTypeLabel(matchType: ImportRuleMatchType) {
  return matchTypeOptions.find((option) => option.value === matchType)?.label || matchType;
}

function describeRuleResult(
  rule: ImportRule,
  categories: MetadataItem[],
  accounts: MetadataItem[],
  tags: MetadataItem[],
) {
  const parts = [
    rule.result.category_id ? `分类 ${metadataName(categories, rule.result.category_id)}` : '',
    rule.result.account_id ? `账户 ${metadataName(accounts, rule.result.account_id)}` : '',
    rule.result.tag_ids?.length
      ? `标签 ${rule.result.tag_ids.map((id) => metadataName(tags, id)).join('、')}`
      : '',
  ].filter(Boolean);
  return parts.length > 0 ? parts.join(' · ') : '仅记录命中解释';
}

function describeRuleMetadataWarning(
  rule: ImportRule,
  categories: MetadataItem[],
  accounts: MetadataItem[],
  tags: MetadataItem[],
) {
  return [
    rule.result.category_id ? metadataIssue(categories, rule.result.category_id, '分类') : '',
    rule.result.account_id ? metadataIssue(accounts, rule.result.account_id, '账户') : '',
    ...(rule.result.tag_ids || []).map((id) => metadataIssue(tags, id, '标签')),
  ].filter(Boolean).join('、');
}

function metadataName(items: MetadataItem[], id: string) {
  return items.find((item) => item.id === id)?.name || id.slice(0, 8);
}

function metadataIssue(items: MetadataItem[], id: string, label: string) {
  const item = items.find((candidate) => candidate.id === id);
  if (!item) return `${label} ${id.slice(0, 8)} 不可用`;
  return item.is_archived ? `${label} ${item.name} 已归档` : '';
}
