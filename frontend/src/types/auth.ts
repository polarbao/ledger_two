export interface User {
  id: string;
  username: string;
  display_name: string;   // 后端 MeData.DisplayName -> json:"display_name"
  avatar_url: string;     // 后端 MeData.AvatarURL   -> json:"avatar_url"
  ledger_id: string;      // 后端 MeData.LedgerID    -> json:"ledger_id"
}

export interface AuthState {
  user: User | null;
  isInitialized: boolean;
  isLoading: boolean;
}
