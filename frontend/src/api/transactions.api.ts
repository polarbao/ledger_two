import { api } from './client';
import type {
  TransactionResponse,
  Category,
  CreateTransactionPayload,
  CreateSharedExpensePayload,
} from '../types/transaction';

export const transactionsApi = {
  getCategories: () =>
    api.get<Category[]>('/api/categories'),
  createTransaction: (payload: CreateTransactionPayload) =>
    api.post<TransactionResponse>('/api/transactions', payload),
  createSharedExpense: (payload: CreateSharedExpensePayload) =>
    api.post<TransactionResponse>('/api/shared-expenses', payload),
};
