import { api } from './client';

export interface BackupInfo {
  filename: string;
  size_bytes: number;
  created_at: string;
}

export interface RestoreResponse {
  success: boolean;
  instructions: string;
}

export interface DiagnosticStatus {
  key: string;
  label: string;
  status: 'ok' | 'warning' | 'error' | string;
  configured: boolean;
  writable?: boolean;
  message?: string;
  version?: number;
}

export interface SystemDiagnostics {
  env: string;
  deployment_channel: string;
  app_base_url_set: boolean;
  cookie_secure: string;
  cookie_samesite: string;
  database: DiagnosticStatus;
  storage: DiagnosticStatus[];
  latest_backup?: BackupInfo;
  audit_action_count: Record<string, number>;
  generated_at: string;
}

export const safetyApi = {
  createBackup: async (): Promise<{ success: boolean; filename: string }> => {
    return api.post('/api/admin/backup');
  },

  getBackups: async (): Promise<BackupInfo[]> => {
    return api.get('/api/admin/backups');
  },

  restoreBackup: async (filename: string): Promise<RestoreResponse> => {
    return api.post('/api/admin/restore', { filename });
  },

  getDiagnostics: async (): Promise<SystemDiagnostics> => {
    return api.get('/api/admin/diagnostics');
  },
};
