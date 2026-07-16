import { api } from './client';

export interface InitStatusResponse {
  initialized: boolean;
}

export interface SetupLedgerPayload {
  ledger_name: string;
  default_currency: string;
  user_a_username: string;
  user_a_display_name: string;
  user_a_password: string;
  user_b_username: string;
  user_b_display_name: string;
  user_b_password: string;
}

export const initApi = {
  getStatus: () =>
		api.get<InitStatusResponse>('/api/init/status', { ledgerScope: 'none' }),
  setup: (payload: SetupLedgerPayload) =>
		api.post<void>('/api/init/setup', payload, { ledgerScope: 'none' }),
};
