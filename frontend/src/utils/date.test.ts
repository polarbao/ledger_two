/* eslint-disable no-undef */
import { formatDate, getMonthOnly } from './date';

describe('Date Utility Tests', () => {
  test('formatDate should format ISO string into YYYY-MM-DD HH:MM', () => {
    // 2026-06-10T15:30:00Z
    const testDate = new Date(2026, 5, 10, 15, 30, 0).toISOString();
    const result = formatDate(testDate);
    expect(result).toContain('2026-06-10');
  });

  test('formatDate fallback on invalid dates', () => {
    expect(formatDate('')).toBe('');
    expect(formatDate('invalid-date')).toBe('invalid-date');
  });

  test('getMonthOnly', () => {
    expect(getMonthOnly('2026-06-10T15:30:00Z')).toBe('2026-06');
    expect(getMonthOnly('2026-06')).toBe('2026-06');
    expect(getMonthOnly('')).toBe('');
  });
});
