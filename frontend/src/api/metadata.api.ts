import { api } from './client';
import type {
  MetadataArchivePayload,
  MetadataArchiveResult,
  MetadataItem,
  MetadataKind,
  MetadataProfileApplyResult,
  MetadataProfileConflictResolution,
  MetadataProfileKey,
  MetadataProfilePreviewResult,
  MetadataUpsertPayload,
} from '../types/metadata';

export const metadataApi = {
  list: (kind: MetadataKind, includeArchived = true, signal?: AbortSignal) =>
    api.get<MetadataItem[]>(
      `/api/metadata/${kind}/?include_archived=${includeArchived ? 'true' : 'false'}`,
      { signal },
    ),

  create: (kind: MetadataKind, payload: MetadataUpsertPayload) =>
    api.post<MetadataItem>(`/api/metadata/${kind}/`, payload),

  update: (kind: MetadataKind, id: string, payload: MetadataUpsertPayload) =>
    api.patch<{ success: boolean }>(`/api/metadata/${kind}/${id}`, payload),

  archive: (kind: MetadataKind, id: string, payload: MetadataArchivePayload = {}) =>
    api.post<MetadataArchiveResult>(`/api/metadata/${kind}/${id}/archive`, payload),

  restore: (kind: MetadataKind, id: string) =>
    api.post<{ success: boolean }>(`/api/metadata/${kind}/${id}/restore`, {}),

  reorder: (kind: MetadataKind, orderedIds: string[]) =>
    api.post<{ success: boolean }>(`/api/metadata/${kind}/reorder`, { ordered_ids: orderedIds }),

  previewDefaultProfile: (profile: MetadataProfileKey = 'basic_cn_v1') =>
    api.post<MetadataProfilePreviewResult>('/api/metadata/default-profile/preview', { profile }),

  applyDefaultProfile: (
    profile: MetadataProfileKey,
    resolutions: MetadataProfileConflictResolution[],
  ) => api.post<MetadataProfileApplyResult>('/api/metadata/default-profile/apply', {
    profile,
    resolutions,
  }),
};
