import { client } from './client';

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
    return client.post('/api/admin/backup');
  },

  getBackups: async (): Promise<BackupInfo[]> => {
    return client.get('/api/admin/backups');
  },

  restoreBackup: async (filename: string): Promise<RestoreResponse> => {
    return client.post('/api/admin/restore', { filename });
  },
};
