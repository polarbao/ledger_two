import type { SuggestedTransfer } from '../types/settlement';
import { centsToYuan } from '../utils/money';

export type SettlementScope = 'all' | 'month';

export interface SettlementBalanceDetail {
  userId: string;
  displayName: string;
  isMe: boolean;
  paidCents: number;
  shareCents: number;
  rawNetCents: number;
  settlementNetCents: number;
  finalNetCents: number;
}

interface SettlementCopyInput {
  scope: SettlementScope;
  month: string;
  transfer: SuggestedTransfer;
  details: SettlementBalanceDetail[];
  getUserDisplayName: (userId: string) => string;
}

export function formatSignedYuan(cents: number) {
  if (cents === 0) return '¥0.00';
  return `${cents > 0 ? '+' : '-'}¥${centsToYuan(Math.abs(cents))}`;
}

export function describeSettlementNet(cents: number) {
  if (cents > 0) return `${formatSignedYuan(cents)} 应收`;
  if (cents < 0) return `${formatSignedYuan(cents)} 应付`;
  return '¥0.00 已结清';
}

export function buildSettlementCopyText({
  scope,
  month,
  transfer,
  details,
  getUserDisplayName,
}: SettlementCopyInput) {
  const range = scope === 'month' ? `${month} 本月` : '全部未结账期';
  const lines = [
    `【LedgerTwo 结算】${range}`,
    `${getUserDisplayName(transfer.from_user_id)} 需转给 ${getUserDisplayName(transfer.to_user_id)} ¥${centsToYuan(transfer.amount_cents)}`,
    '',
    '对账拆解：',
    ...details.map((item) =>
      `${item.displayName}: 实际支付 ¥${centsToYuan(item.paidCents)} / 实际承担 ¥${centsToYuan(item.shareCents)} / 共同支出净额 ${formatSignedYuan(item.rawNetCents)} / 已登记结算 ${formatSignedYuan(item.settlementNetCents)} / 最终未结 ${describeSettlementNet(item.finalNetCents)}`
    ),
    '',
    '生成结算记录只登记本次转账，不修改历史共同支出。',
  ];
  return lines.join('\n');
}

export async function copyTextToClipboard(
  value: string,
  writeText: ((text: string) => Promise<void>) | undefined,
  legacyCopy: (text: string) => boolean,
) {
  if (writeText) {
    await writeText(value);
    return;
  }

  if (!legacyCopy(value)) {
    throw new Error('clipboard unavailable');
  }
}
