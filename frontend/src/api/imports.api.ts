import { api } from './client';
import type {
  ImportCommitResult,
  ImportPreviewBatch,
  ImportRule,
  ImportRuleUpsertPayload,
  ImportSourceType,
  UpdateImportRowPayload,
} from '../types/imports';

export const importsApi = {
  preview: (payload: { file: File; sourceType: ImportSourceType }) => {
    const formData = new FormData();
    formData.append('source_type', payload.sourceType);
    formData.append('file', payload.file);
    return api.post<ImportPreviewBatch>('/api/imports/preview', formData);
  },

  getBatch: (batchId: string) =>
    api.get<ImportPreviewBatch>(`/api/imports/${batchId}`),

  updateRow: (batchId: string, rowId: string, payload: UpdateImportRowPayload) =>
    api.patch<ImportPreviewBatch>(`/api/imports/${batchId}/rows/${rowId}`, payload),

  commit: (batchId: string) =>
    api.post<ImportCommitResult>(`/api/imports/${batchId}/commit`, {}),

  listRules: (status: 'active' | 'archived' | 'all' = 'all') =>
    api.get<ImportRule[]>(`/api/import-rules/?status=${status}`),

  createRule: (payload: ImportRuleUpsertPayload) =>
    api.post<ImportRule>('/api/import-rules/', payload),

  archiveRule: (ruleId: string) =>
    api.post<ImportRule>(`/api/import-rules/${ruleId}/archive`, {}),

  restoreRule: (ruleId: string) =>
    api.post<ImportRule>(`/api/import-rules/${ruleId}/restore`, {}),
};
