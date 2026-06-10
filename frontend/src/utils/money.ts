export function yuanToCents(value: string): number {
  const normalized = value.trim().replace(/,/g, '');
  if (!/^\d+(\.\d{0,2})?$/.test(normalized)) {
    throw new Error('金额格式错误');
  }
  return Math.round(Number(normalized) * 100);
}

export function centsToYuan(amountCents: number): string {
  return (amountCents / 100).toFixed(2);
}

export function formatCny(amountCents: number): string {
  return `¥${centsToYuan(amountCents)}`;
}
