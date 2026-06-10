import { api } from './client';
import type { DashboardResponse } from '../types/dashboard';

export const dashboardApi = {
  getDashboard: (month: string) =>
    api.get<DashboardResponse>(`/api/dashboard?month=${month}`),
};
