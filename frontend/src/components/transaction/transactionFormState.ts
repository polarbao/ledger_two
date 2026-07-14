import { yuanToCents } from '../../utils/money';

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

export type SharedExpensePreviewMember = {
  user_id: string;
  display_name: string;
};

export type SharedExpensePreviewItem = {
  userId: string;
  displayName: string;
  shareAmountCents: number;
  isPayer: boolean;
  isParticipating: boolean;
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

export function shouldOpenAdvancedFields(values: TransactionFormValueSnapshot) {
  return Boolean(
    values.title?.trim()
    || values.tag_names?.trim()
    || values.note?.trim()
    || values.visibility === 'private'
    || values.attachment_paths?.length,
  );
}

export function buildSharedExpensePreview(
  amount: string,
  members: SharedExpensePreviewMember[],
  payerUserId: string,
  splitMethod: 'equal' | 'payer_only',
): SharedExpensePreviewItem[] {
  let amountCents = 0;
  try {
    amountCents = amount ? yuanToCents(amount) : 0;
  } catch {
    amountCents = 0;
  }

  const memberCount = members.length;
  const equalBase = memberCount > 0 ? Math.floor(amountCents / memberCount) : 0;
  const equalRemainder = memberCount > 0 ? amountCents % memberCount : 0;

  return members.map((member) => {
    const isPayer = member.user_id === payerUserId;
    const isParticipating = splitMethod === 'equal' || isPayer;
    const shareAmountCents = splitMethod === 'payer_only'
      ? isPayer ? amountCents : 0
      : equalBase + (isPayer ? equalRemainder : 0);

    return {
      userId: member.user_id,
      displayName: member.display_name,
      shareAmountCents,
      isPayer,
      isParticipating,
    };
  });
}
