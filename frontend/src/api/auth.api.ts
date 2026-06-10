import { api } from './client';
import type { User } from '../types/auth';

export const authApi = {
  login: (username: string, password: string) =>
    api.post<User>('/api/auth/login', { username, password }),
  logout: () =>
    api.post<void>('/api/auth/logout'),
  getMe: () =>
    api.get<User>('/api/auth/me'),
};
