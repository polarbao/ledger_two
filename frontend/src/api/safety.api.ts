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
};
