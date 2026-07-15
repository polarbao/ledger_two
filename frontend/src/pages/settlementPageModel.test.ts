import { describe, expect, it, vi } from 'vitest';
import {
  buildSettlementCopyText,
  copyTextToClipboard,
  describeSettlementNet,
  formatSignedYuan,
} from './settlementPageModel';

describe('settlementPageModel', () => {
  it('formats the authoritative settlement explanation without floats', () => {
    expect(formatSignedYuan(12345)).toBe('+¥123.45');
    expect(formatSignedYuan(-7)).toBe('-¥0.07');
    expect(describeSettlementNet(0)).toBe('¥0.00 已结清');
  });

  it('builds scoped copy that explains settlement records do not mutate bills', () => {
    const text = buildSettlementCopyText({
      scope: 'month',
      month: '2026-07',
      transfer: { from_user_id: 'user-b', to_user_id: 'user-a', amount_cents: 6000 },
      details: [{
        userId: 'user-a',
        displayName: '我',
        isMe: true,
        paidCents: 20000,
        shareCents: 14000,
        rawNetCents: 6000,
        settlementNetCents: 0,
        finalNetCents: 6000,
      }],
      getUserDisplayName: (id) => id === 'user-a' ? '我' : '对方',
    });

    expect(text).toContain('2026-07 本月');
    expect(text).toContain('对方 需转给 我 ¥60.00');
    expect(text).toContain('实际支付 ¥200.00');
    expect(text).toContain('不修改历史共同支出');
  });

  it('uses the legacy copier only when the Clipboard API is unavailable', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    const legacyCopy = vi.fn().mockReturnValue(true);

    await copyTextToClipboard('结算文案', writeText, legacyCopy);
    expect(writeText).toHaveBeenCalledWith('结算文案');
    expect(legacyCopy).not.toHaveBeenCalled();

    await copyTextToClipboard('备用文案', undefined, legacyCopy);
    expect(legacyCopy).toHaveBeenCalledWith('备用文案');
  });

  it('surfaces clipboard failures so the page can expose manual copy text', async () => {
    await expect(copyTextToClipboard(
      '结算文案',
      vi.fn().mockRejectedValue(new Error('denied')),
      vi.fn(),
    )).rejects.toThrow('denied');

    await expect(copyTextToClipboard('结算文案', undefined, () => false))
      .rejects.toThrow('clipboard unavailable');
  });
});
