import { api } from './client';

export type DeploymentChannel = 'development' | 'staging' | 'production' | 'unknown';

export interface HealthStatus {
  status: string;
  db: string;
  version: string;
  schema_version: number;
  deployment_channel: DeploymentChannel;
}

export const systemApi = {
  getHealth: async (): Promise<HealthStatus> => api.get('/api/healthz'),
};
