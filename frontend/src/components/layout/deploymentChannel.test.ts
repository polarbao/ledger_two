import { describe, expect, it } from 'vitest';
import { getDeploymentChannelMeta } from './deploymentChannel';

describe('getDeploymentChannelMeta', () => {
  it.each([
    ['development', '开发环境', 'development'],
    ['staging', '验收环境', 'staging'],
    ['production', '正式数据', 'production'],
    ['unknown', '环境未知', 'unknown'],
  ] as const)('maps %s to a visible environment label', (channel, label, tone) => {
    expect(getDeploymentChannelMeta(channel)).toEqual({ label, tone });
  });
});
