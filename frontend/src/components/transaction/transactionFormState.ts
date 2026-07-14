import { yuanToCents } from '../../utils/money';
import type { TransactionResponse, UpdateTransactionPayload } from '../../types/transaction';

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

const normalizeTagNames = (value: string) => Array.from(new Set(
  value.split(/[，, ；;]/).map((item) => item.trim()).filter(Boolean),
));

const sameStringSet = (left: string[], right: string[]) => {
  const leftSorted = [...new Set(left)].sort();
  const rightSorted = [...new Set(right)].sort();
  return leftSorted.length === rightSorted.length
    && leftSorted.every((item, index) => item === rightSorted[index]);
};

const sameStringList = (left: string[], right: string[]) => (
  left.length === right.length && left.every((item, index) => item === right[index])
);

export function transactionToFormValues(transaction: TransactionResponse): TransactionFormValueSnapshot {
  return {
    type: transaction.type === 'settlement' ? 'expense' : transaction.type,
    amount: (transaction.amount_cents / 100).toFixed(2),
    title: transaction.title || '',
    category_id: transaction.category_id || '',
    account_id: transaction.account_id || '',
    tag_names: transaction.tags?.join(', ') || '',
    payer_user_id: transaction.payer_user_id,
    split_method: transaction.split_method === 'payer_only' ? 'payer_only' : 'equal',
    occurred_at: transaction.occurred_at.substring(0, 10),
    note: transaction.note || '',
    visibility: transaction.visibility === 'private' ? 'private' : 'partner_readable',
    attachment_paths: transaction.attachment_paths || [],
  };
}

export function getTransactionEditBlockReason(
  transaction: TransactionResponse,
  currentUserId: string,
  canWrite: boolean,
  ledgerUserIds: string[],
  isOffline: boolean,
): string | null {
  if (!canWrite) return '当前账本角色没有编辑权限';
  if (transaction.created_by_user_id !== currentUserId) return '只能编辑自己创建的账单';
  if (transaction.type === 'settlement') return '结算记录不能在流水页编辑';
  if (isOffline) return '离线状态不能编辑已保存账单';
  if (transaction.type !== 'shared_expense') return null;
  if (transaction.split_method !== 'equal' && transaction.split_method !== 'payer_only') {
    return '自定义分摊账单暂不支持在快捷编辑器中修改';
  }

  const participantIds = transaction.participants?.map((item) => item.user_id) || [];
  if (ledgerUserIds.length === 0) return '账本成员信息尚未加载完成';
  if (!sameStringSet(participantIds, ledgerUserIds)) {
    return '历史参与人和当前账本成员不一致，请先保留原账单';
  }
  return null;
}

export function buildTransactionUpdatePayload(
  transaction: TransactionResponse,
  values: TransactionFormValueSnapshot,
): UpdateTransactionPayload {
  const payload: UpdateTransactionPayload = {};
  const amountCents = yuanToCents(values.amount);
  const title = values.title?.trim() || '';
  const note = values.note?.trim() || '';
  const categoryId = values.category_id || null;
  const accountId = values.account_id || null;
  const tags = normalizeTagNames(values.tag_names || '');
  const attachments = values.attachment_paths || [];

  if (amountCents !== transaction.amount_cents) payload.amount_cents = amountCents;
  if (title !== transaction.title) payload.title = title;
  if (values.occurred_at !== transaction.occurred_at.substring(0, 10)) {
    payload.occurred_at = new Date(values.occurred_at).toISOString();
  }
  if (values.payer_user_id !== transaction.payer_user_id) payload.payer_user_id = values.payer_user_id;
  if (categoryId !== (transaction.category_id || null)) payload.category_id = categoryId;
  if (note !== (transaction.note || '')) payload.note = note;
  if (!sameStringSet(tags, transaction.tags || [])) payload.tag_names = tags;

  if (transaction.type === 'shared_expense') {
    if (values.split_method !== transaction.split_method) payload.split_method = values.split_method;
  } else {
    if (accountId !== (transaction.account_id || null)) payload.account_id = accountId;
    if (values.visibility !== transaction.visibility) payload.visibility = values.visibility;
    if (!sameStringList(attachments, transaction.attachment_paths || [])) {
      payload.attachment_paths = attachments;
    }
  }

  return payload;
}

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
