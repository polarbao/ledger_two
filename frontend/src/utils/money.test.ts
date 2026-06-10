/* eslint-disable no-undef */
import { yuanToCents, centsToYuan, formatCny } from './money';

// 简单的高保真单元测试，可使用 vitest/jest 运行，也可作自测依据
describe('Money Utility Tests', () => {
  test('centsToYuan', () => {
    expect(centsToYuan(100)).toBe('1.00');
    expect(centsToYuan(150)).toBe('1.50');
    expect(centsToYuan(0)).toBe('0.00');
    expect(centsToYuan(99)).toBe('0.99');
    expect(centsToYuan(10050)).toBe('100.50');
  });

  test('yuanToCents', () => {
    expect(yuanToCents('1.00')).toBe(100);
    expect(yuanToCents('1.5')).toBe(150);
    expect(yuanToCents('0.99')).toBe(99);
    expect(yuanToCents('0')).toBe(0);
    expect(yuanToCents(' 100.50 ')).toBe(10050);
  });

  test('yuanToCents should throw on invalid format', () => {
    expect(() => yuanToCents('abc')).toThrow();
    expect(() => yuanToCents('-1.50')).toThrow();
    expect(() => yuanToCents('1.505')).toThrow();
  });

  test('formatCny', () => {
    expect(formatCny(100)).toBe('¥1.00');
    expect(formatCny(10050)).toBe('¥100.50');
  });
});
