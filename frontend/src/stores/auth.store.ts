import { create } from 'zustand';
import type { User } from '../types/auth';

interface AuthStore {
  user: User | null;
  isInitialized: boolean;
  setUser: (user: User | null) => void;
  setIsInitialized: (initialized: boolean) => void;
  clear: () => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isInitialized: false,
  setUser: (user) => set({ user }),
  setIsInitialized: (isInitialized) => set({ isInitialized }),
  clear: () => set({ user: null }),
}));
