import type { TransactionListFilter } from './transactions.api';
import type { MetadataKind } from '../types/metadata';

export const UNSELECTED_LEDGER_ID = 'no-active-ledger';

const ledgerScope = (ledgerId: string | null | undefined) => ledgerId || UNSELECTED_LEDGER_ID;

export const queryKeys = {
  ledgers: {
    all: ['ledgers'] as const,
  },
  dashboard: {
    root: (ledgerId?: string | null) => ['dashboard', ledgerScope(ledgerId)] as const,
    month: (ledgerId: string | null | undefined, month: string) =>
      ['dashboard', ledgerScope(ledgerId), month] as const,
  },
  transactions: {
    root: (ledgerId?: string | null) => ['transactions', ledgerScope(ledgerId)] as const,
    list: (ledgerId: string | null | undefined, filter: TransactionListFilter) =>
      ['transactions', ledgerScope(ledgerId), filter] as const,
  },
  categories: (ledgerId?: string | null) => ['categories', ledgerScope(ledgerId)] as const,
  accounts: (ledgerId?: string | null) => ['accounts', ledgerScope(ledgerId)] as const,
  transactionDefaults: (ledgerId?: string | null) =>
    ['transaction-defaults', ledgerScope(ledgerId)] as const,
  metadata: {
    root: (ledgerId?: string | null) => ['metadata', ledgerScope(ledgerId)] as const,
    list: (ledgerId: string | null | undefined, kind: MetadataKind) =>
      ['metadata', ledgerScope(ledgerId), kind] as const,
  },
  importRules: (ledgerId?: string | null) => ['importRules', ledgerScope(ledgerId)] as const,
  templates: (ledgerId?: string | null) => ['transaction-templates', ledgerScope(ledgerId)] as const,
  recurringRules: (ledgerId?: string | null) => ['recurring-rules', ledgerScope(ledgerId)] as const,
  recurringReminders: (ledgerId?: string | null) => ['recurring-reminders', ledgerScope(ledgerId)] as const,
  safety: {
    diagnostics: (ledgerId?: string | null) => ['safety', ledgerScope(ledgerId), 'diagnostics'] as const,
  },
  settlements: {
    root: (ledgerId?: string | null) => ['settlements', ledgerScope(ledgerId)] as const,
    balance: (ledgerId?: string | null) => ['settlement-balance', ledgerScope(ledgerId)] as const,
    list: (ledgerId: string | null | undefined, month: string) =>
      ['settlements', ledgerScope(ledgerId), month] as const,
  },
  reports: {
    root: (ledgerId?: string | null) => ['reports', ledgerScope(ledgerId)] as const,
    monthly: (ledgerId: string | null | undefined, month: string) =>
      ['reports', ledgerScope(ledgerId), 'monthly', month] as const,
    category: (ledgerId: string | null | undefined, month: string) =>
      ['reports', ledgerScope(ledgerId), 'category', month] as const,
    tag: (ledgerId: string | null | undefined, month: string) =>
      ['reports', ledgerScope(ledgerId), 'tag', month] as const,
    member: (ledgerId: string | null | undefined, month: string) =>
      ['reports', ledgerScope(ledgerId), 'member', month] as const,
  },
} as const;
