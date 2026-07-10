import type { DeploymentChannel } from '../../api/system.api';

export interface DeploymentChannelMeta {
  label: string;
  tone: 'development' | 'staging' | 'production' | 'unknown';
}

export function getDeploymentChannelMeta(channel: DeploymentChannel | string): DeploymentChannelMeta {
  switch (channel) {
    case 'development':
      return { label: '开发环境', tone: 'development' };
    case 'staging':
      return { label: '验收环境', tone: 'staging' };
    case 'production':
      return { label: '正式数据', tone: 'production' };
    default:
      return { label: '环境未知', tone: 'unknown' };
  }
}
