import { api } from './client';
import type { MetadataItem, MetadataKind, MetadataUpsertPayload } from '../types/metadata';

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

  archive: (kind: MetadataKind, id: string) =>
    api.post<{ success: boolean }>(`/api/metadata/${kind}/${id}/archive`, {}),

  restore: (kind: MetadataKind, id: string) =>
    api.post<{ success: boolean }>(`/api/metadata/${kind}/${id}/restore`, {}),

  reorder: (kind: MetadataKind, orderedIds: string[]) =>
    api.post<{ success: boolean }>(`/api/metadata/${kind}/reorder`, { ordered_ids: orderedIds }),
};
