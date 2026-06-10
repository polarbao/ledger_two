export interface User {
  id: string;
  username: string;
  displayName: string;
  createdAt: string;
}

export interface AuthState {
  user: User | null;
  isInitialized: boolean;
  isLoading: boolean;
}
