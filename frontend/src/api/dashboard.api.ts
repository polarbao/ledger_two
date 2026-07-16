import { api } from './client';
import type { DashboardResponse } from '../types/dashboard';

export const dashboardApi = {
  getDashboard: (month: string, signal?: AbortSignal) =>
    api.get<DashboardResponse>(`/api/dashboard?month=${month}`, { signal }),
};
