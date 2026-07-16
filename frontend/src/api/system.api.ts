import { api } from './client';

export type DeploymentChannel = 'development' | 'staging' | 'production' | 'unknown';

export interface HealthStatus {
  status: string;
  db: string;
  version: string;
  schema_version: number;
  deployment_channel: DeploymentChannel;
  import_xlsx_enabled: boolean;
}

export const systemApi = {
	getHealth: async (signal?: AbortSignal): Promise<HealthStatus> =>
    api.get('/api/healthz', { ledgerScope: 'none', signal }),
};
