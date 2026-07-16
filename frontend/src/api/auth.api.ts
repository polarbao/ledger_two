import { api } from './client';
import type { User } from '../types/auth';

export const authApi = {
  login: (username: string, password: string) =>
		api.post<{ status: string }>('/api/auth/login', { username, password }, { ledgerScope: 'none' }),
  logout: () =>
		api.post<void>('/api/auth/logout', undefined, { ledgerScope: 'none' }),
  getMe: () =>
		api.get<User>('/api/auth/me', { ledgerScope: 'none' }),
};
