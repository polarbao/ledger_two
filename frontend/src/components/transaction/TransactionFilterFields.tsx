import type { FormEvent } from 'react';
import type { Category } from '../../types/transaction';
import Button from '../ui/Button';

export interface TransactionFilterDraft {
  categoryId: string;
  payerUserId: string;
  visibility: string;
  tag: string;
  minAmount: string;
  maxAmount: string;
}

interface LedgerUserOption {
  userId: string;
  label: string;
}

interface TransactionFilterFieldsProps {
  categories: Category[];
  users: LedgerUserOption[];
  values: TransactionFilterDraft;
  showActions?: boolean;
  onApply: (filters: TransactionFilterDraft) => void;
  onReset: () => void;
}

export default function TransactionFilterFields({
  categories,
  users,
  values,
  showActions = true,
  onApply,
  onReset,
}: TransactionFilterFieldsProps) {
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    onApply({
      categoryId: String(data.get('category_id') || ''),
      payerUserId: String(data.get('payer_user_id') || ''),
      visibility: String(data.get('visibility') || ''),
      tag: String(data.get('tag') || '').trim(),
      minAmount: String(data.get('min_amount') || '').trim(),
      maxAmount: String(data.get('max_amount') || '').trim(),
    });
  };

  return (
    <form className="transaction-filter-form" onSubmit={handleSubmit}>
      <div className="transaction-filter-form__grid">
        <label>
          <span>分类</span>
          <select name="category_id" defaultValue={values.categoryId}>
            <option value="">全部分类</option>
            {categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}{category.is_archived ? '（已归档）' : ''}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>付款人</span>
          <select name="payer_user_id" defaultValue={values.payerUserId}>
            <option value="">全部付款人</option>
            {users.map((user) => <option key={user.userId} value={user.userId}>{user.label}</option>)}
          </select>
        </label>
        <label>
          <span>可见范围</span>
          <select name="visibility" defaultValue={values.visibility}>
            <option value="">全部范围</option>
            <option value="private">仅自己可见</option>
            <option value="partner_readable">对方可见，只读</option>
          </select>
        </label>
        <label>
          <span>标签</span>
          <input name="tag" type="search" defaultValue={values.tag} placeholder="输入标签名称" />
        </label>
        <label>
          <span>最低金额</span>
          <input name="min_amount" type="number" min="0" step="0.01" inputMode="decimal" defaultValue={values.minAmount} placeholder="元" />
        </label>
        <label>
          <span>最高金额</span>
          <input name="max_amount" type="number" min="0" step="0.01" inputMode="decimal" defaultValue={values.maxAmount} placeholder="元" />
        </label>
      </div>
      {showActions ? (
        <div className="transaction-filter-form__actions">
          <Button variant="ghost" onClick={onReset}>重置</Button>
          <Button variant="primary" type="submit">应用筛选</Button>
        </div>
      ) : <button type="submit" className="sr-only">应用筛选</button>}
    </form>
  );
}
