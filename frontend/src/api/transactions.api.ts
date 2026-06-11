import { api } from './client';
import type {
  TransactionResponse,
  Category,
  CreateTransactionPayload,
  CreateSharedExpensePayload,
  TransactionTemplateResponse,
  CreateTemplatePayload,
} from '../types/transaction';

export interface TransactionListFilter {
  month?: string;
  type?: string;
  category_id?: string;
  keyword?: string;
  page?: number;
  page_size?: number;
}

export const transactionsApi = {
  getCategories: () =>
    api.get<Category[]>('/api/categories'),

  /** GET /api/transactions — 分页流水列表 */
  list: (filter: TransactionListFilter = {}) => {
    const params = new URLSearchParams();
    if (filter.month)       params.set('month', filter.month);
    if (filter.type)        params.set('type', filter.type);
    if (filter.category_id) params.set('category_id', filter.category_id);
    if (filter.keyword)     params.set('keyword', filter.keyword);
    if (filter.page)        params.set('page', String(filter.page));
    if (filter.page_size)   params.set('page_size', String(filter.page_size));
    const qs = params.toString();
    return api.get<TransactionResponse[]>(`/api/transactions${qs ? `?${qs}` : ''}`);
  },

  createTransaction: (payload: CreateTransactionPayload) =>
    api.post<TransactionResponse>('/api/transactions', payload),

  createSharedExpense: (payload: CreateSharedExpensePayload) =>
    api.post<TransactionResponse>('/api/shared-expenses', payload),

  deleteTransaction: (id: string) =>
    api.delete<void>(`/api/transactions/${id}`),

  listTemplates: () =>
    api.get<TransactionTemplateResponse[]>('/api/transaction-templates'),

  createTemplate: (payload: CreateTemplatePayload) =>
    api.post<TransactionTemplateResponse>('/api/transaction-templates', payload),

  updateTemplate: (id: string, payload: CreateTemplatePayload) =>
    api.put<TransactionTemplateResponse>(`/api/transaction-templates/${id}`, payload),

  deleteTemplate: (id: string) =>
    api.delete<{ success: boolean }>(`/api/transaction-templates/${id}`),
};

