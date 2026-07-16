export interface User {
  id: string;
  username: string;
	display_name: string;
	avatar_url: string;
	instance_admin: boolean;
}

export interface AuthState {
  user: User | null;
  isInitialized: boolean;
  isLoading: boolean;
}
