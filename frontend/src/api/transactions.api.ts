import { api } from './client';
import type {
  TransactionResponse,
  Category,
  CreateTransactionPayload,
  CreateSharedExpensePayload,
  TransactionTemplateResponse,
  CreateTemplatePayload,
  RecurringRuleResponse,
  CreateRecurringRulePayload,
  RecurringReminderResponse,
  CSVParseResponse,
  AnalyzeImportPayload,
  AnalyzeImportResponse,
  CommitImportPayload,
  Account,
  ImportRuleResponse,
  CreateImportRulePayload,
  TransactionDefaultResponse,
  UpdateTransactionPayload,
} from '../types/transaction';

export interface TransactionListFilter {
  month?: string;
  type?: string;
  category_id?: string;
  keyword?: string;
  min_amount?: number; // 整数分
  max_amount?: number; // 整数分
  payer_user_id?: string;
  visibility?: string;
  tag?: string;
  page?: number;
  page_size?: number;
}

export interface BatchTagPayload {
  transaction_ids: string[];
  tag_names: string[];
}

export interface ListCategoryOptions {
  includeArchived?: boolean;
}

export const transactionsApi = {
  getCategories: (options: ListCategoryOptions = {}, signal?: AbortSignal) => {
    const qs = options.includeArchived ? '?include_archived=true' : '';
    return api.get<Category[]>(`/api/categories${qs}`, { signal });
  },

  /** GET /api/transactions — 分页流水列表 */
  list: (filter: TransactionListFilter = {}, signal?: AbortSignal) => {
    const params = new URLSearchParams();
    if (filter.month)          params.set('month', filter.month);
    if (filter.type)           params.set('type', filter.type);
    if (filter.category_id)    params.set('category_id', filter.category_id);
    if (filter.keyword)        params.set('keyword', filter.keyword);
    if (filter.min_amount !== undefined) params.set('min_amount', String(filter.min_amount));
    if (filter.max_amount !== undefined) params.set('max_amount', String(filter.max_amount));
    if (filter.payer_user_id)  params.set('payer_user_id', filter.payer_user_id);
    if (filter.visibility)     params.set('visibility', filter.visibility);
    if (filter.tag)            params.set('tag', filter.tag);
    if (filter.page)           params.set('page', String(filter.page));
    if (filter.page_size)      params.set('page_size', String(filter.page_size));
    const qs = params.toString();
    return api.get<TransactionResponse[]>(
      `/api/transactions${qs ? `?${qs}` : ''}`,
      { signal },
    );
  },

  createTransaction: (payload: CreateTransactionPayload) =>
    api.post<TransactionResponse>('/api/transactions', payload),

  createSharedExpense: (payload: CreateSharedExpensePayload) =>
    api.post<TransactionResponse>('/api/shared-expenses', payload),

  updateTransaction: (transaction: TransactionResponse, payload: UpdateTransactionPayload) =>
    api.patch<TransactionResponse>(
      transaction.type === 'shared_expense'
        ? `/api/shared-expenses/${transaction.id}`
        : `/api/transactions/${transaction.id}`,
      payload,
    ),

  deleteTransaction: (id: string) =>
    api.delete<void>(`/api/transactions/${id}`),

  listTemplates: (
    options: { includeArchived?: boolean } = {},
    signal?: AbortSignal,
  ) =>
    api.get<TransactionTemplateResponse[]>(
      `/api/transaction-templates${options.includeArchived ? '?include_archived=true' : ''}`,
      { signal },
    ),

  createTemplate: (payload: CreateTemplatePayload) =>
    api.post<TransactionTemplateResponse>('/api/transaction-templates', payload),

  updateTemplate: (id: string, payload: CreateTemplatePayload) =>
    api.put<TransactionTemplateResponse>(`/api/transaction-templates/${id}`, payload),

  deleteTemplate: (id: string) =>
    api.delete<{ success: boolean }>(`/api/transaction-templates/${id}`),

  archiveTemplate: (id: string) =>
    api.post<{ success: boolean }>(`/api/transaction-templates/${id}/archive`, {}),

  restoreTemplate: (id: string) =>
    api.post<{ success: boolean }>(`/api/transaction-templates/${id}/restore`, {}),

  listRecurringRules: (signal?: AbortSignal) =>
    api.get<RecurringRuleResponse[]>('/api/recurring-rules', { signal }),

  createRecurringRule: (payload: CreateRecurringRulePayload) =>
    api.post<RecurringRuleResponse>('/api/recurring-rules', payload),

  deleteRecurringRule: (id: string) =>
    api.delete<void>(`/api/recurring-rules/${id}`),

  listRecurringReminders: (signal?: AbortSignal) =>
    api.get<RecurringReminderResponse[]>('/api/recurring-reminders', { signal }),

  confirmReminder: (id: string) =>
    api.post<void>(`/api/recurring-reminders/${id}/confirm`, {}),

  skipReminder: (id: string) =>
    api.post<void>(`/api/recurring-reminders/${id}/skip`, {}),

  ignoreReminder: (id: string) =>
    api.post<void>(`/api/recurring-reminders/${id}/ignore`, {}),

  batchTag: (payload: BatchTagPayload) =>
    api.post<{ success: boolean }>('/api/transactions/batch-tag', payload),

  parseCSV: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post<CSVParseResponse>('/api/transactions/import/parse', formData);
  },

  uploadAttachment: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post<{ path: string }>('/api/attachments', formData);
  },

  analyzeImport: (payload: AnalyzeImportPayload) =>
    api.post<AnalyzeImportResponse>('/api/transactions/import/analyze', payload),

  commitImport: (payload: CommitImportPayload) =>
    api.post<{ status: string }>('/api/transactions/import/commit', payload),

  listAccounts: (signal?: AbortSignal) =>
    api.get<Account[]>('/api/accounts', { signal }),

  getTransactionDefaults: (signal?: AbortSignal) =>
    api.get<TransactionDefaultResponse>('/api/transaction-defaults', { signal }),

  listImportRules: (signal?: AbortSignal) =>
    api.get<ImportRuleResponse[]>('/api/import-rules', { signal }),

  createImportRule: (payload: CreateImportRulePayload) =>
    api.post<ImportRuleResponse>('/api/import-rules', payload),

  deleteImportRule: (id: string) =>
    api.delete<void>(`/api/import-rules/${id}`),
};

