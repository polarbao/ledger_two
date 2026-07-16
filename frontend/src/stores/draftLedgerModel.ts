export interface LedgerScopedDraft {
  ledgerId: string | null;
}

export function selectLedgerDrafts<TDraft extends LedgerScopedDraft>(
  drafts: TDraft[],
  ledgerId: string | null,
) {
  if (!ledgerId) return [];
  return drafts.filter((draft) => draft.ledgerId === ledgerId);
}

export function migrateLegacyDrafts<TDraft extends LedgerScopedDraft>(
  drafts: TDraft[],
  persistedLedgerId: string | null,
) {
  return drafts.map((draft) => (
    draft.ledgerId
      ? draft
      : { ...draft, ledgerId: persistedLedgerId }
  ));
}
