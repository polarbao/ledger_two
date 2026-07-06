export type TransactionFormValueSnapshot = {
  type: 'expense' | 'income' | 'shared_expense';
  amount: string;
  title?: string;
  category_id?: string | null;
  account_id?: string | null;
  tag_names?: string;
  payer_user_id: string;
  split_method: 'equal' | 'payer_only';
  occurred_at: string;
  note?: string;
  visibility: 'private' | 'partner_readable';
  attachment_paths?: string[];
};

export function buildContinueTransactionFormValues(
  values: TransactionFormValueSnapshot,
  fallbackDate: string,
): TransactionFormValueSnapshot {
  return {
    type: values.type,
    amount: '',
    title: '',
    category_id: values.category_id || '',
    account_id: values.account_id || '',
    tag_names: values.tag_names || '',
    payer_user_id: values.payer_user_id,
    split_method: values.split_method || 'equal',
    occurred_at: values.occurred_at ? values.occurred_at.substring(0, 10) : fallbackDate,
    note: '',
    visibility: values.visibility || 'partner_readable',
    attachment_paths: [],
  };
}
