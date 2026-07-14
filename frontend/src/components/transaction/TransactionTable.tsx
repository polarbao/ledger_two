import { BookmarkPlus, Copy, Eye, Pencil, Trash2 } from 'lucide-react';
import type { TransactionResponse } from '../../types/transaction';
import { formatDate } from '../../utils/date';
import { centsToYuan } from '../../utils/money';
import Button from '../ui/Button';
import StatusChip from '../ui/StatusChip';
import { getTransactionPresentation } from './transactionsPageModel';

interface TransactionTableProps {
  transactions: TransactionResponse[];
  currentUserId: string;
  canWrite: boolean;
  batchMode: boolean;
  selectedIds: string[];
  categoryLabel: (transaction: TransactionResponse) => string;
  payerName: (payerId: string) => string;
  onSelectAll: (checked: boolean) => void;
  onSelect: (id: string) => void;
  onView: (transaction: TransactionResponse) => void;
  onCopy: (transaction: TransactionResponse, saveAsTemplate: boolean) => void;
  onEdit: (transaction: TransactionResponse) => void;
  onDelete: (transaction: TransactionResponse) => void;
  editBlockReason: (transaction: TransactionResponse) => string | null;
}

export default function TransactionTable({
  transactions,
  currentUserId,
  canWrite,
  batchMode,
  selectedIds,
  categoryLabel,
  payerName,
  onSelectAll,
  onSelect,
  onView,
  onCopy,
  onEdit,
  onDelete,
  editBlockReason,
}: TransactionTableProps) {
  const selectable = transactions.filter((tx) => tx.created_by_user_id === currentUserId);
  const allSelected = selectable.length > 0 && selectable.every((tx) => selectedIds.includes(tx.id));

  return (
    <div className="transactions-table-shell">
      <table className="transactions-table">
        <thead>
          <tr>
            {batchMode ? (
              <th className="transactions-table__select-column">
                <label>
                  <span className="sr-only">选择本页可操作账单</span>
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={(event) => onSelectAll(event.target.checked)}
                  />
                </label>
              </th>
            ) : null}
            <th>日期</th>
            <th>类型</th>
            <th>分类与标题</th>
            <th>付款人</th>
            <th>范围与分摊</th>
            <th>标签</th>
            <th className="transactions-table__amount-column">金额</th>
            <th className="transactions-table__actions-column">操作</th>
          </tr>
        </thead>
        <tbody>
          {transactions.map((tx) => {
            const presentation = getTransactionPresentation(tx);
            const isCreator = tx.created_by_user_id === currentUserId;
            const canMutate = canWrite && isCreator && tx.type !== 'settlement';
            const editReason = canMutate ? editBlockReason(tx) : null;
            return (
              <tr key={tx.id}>
                {batchMode ? (
                  <td className="transactions-table__select-column">
                    <label>
                      <span className="sr-only">选择账单 {tx.title || '无标题'}</span>
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(tx.id)}
                        disabled={!isCreator}
                        onChange={() => onSelect(tx.id)}
                      />
                    </label>
                  </td>
                ) : null}
                <td>
                  <time dateTime={tx.occurred_at}>{formatDate(tx.occurred_at).substring(5, 10)}</time>
                </td>
                <td><StatusChip tone={presentation.typeTone}>{presentation.typeLabel}</StatusChip></td>
                <td className="transactions-table__title-cell">
                  <button type="button" onClick={() => onView(tx)}>{tx.title || '无标题'}</button>
                  <span>{categoryLabel(tx)}</span>
                  {tx.note ? <small>{tx.note}</small> : null}
                </td>
                <td>{payerName(tx.payer_user_id)}</td>
                <td className="transactions-table__scope-cell">
                  <span>{presentation.scopeLabel}</span>
                  <small>{presentation.splitLabel}</small>
                </td>
                <td>
                  <div className="transactions-table__tags">
                    {tx.tags?.length ? tx.tags.slice(0, 2).map((tag) => <span key={tag}>#{tag}</span>) : <span>无标签</span>}
                    {(tx.tags?.length || 0) > 2 ? <span>+{(tx.tags?.length || 0) - 2}</span> : null}
                  </div>
                </td>
                <td className={`transactions-table__amount transactions-table__amount--${presentation.amountTone}`}>
                  {presentation.amountPrefix}¥{centsToYuan(tx.amount_cents)}
                </td>
                <td>
                  <div className="transactions-table__actions">
                    <Button variant="ghost" iconOnly aria-label={`查看${tx.title || '账单'}`} title="查看详情" onClick={() => onView(tx)}>
                      <Eye size={17} />
                    </Button>
                    {canWrite && tx.type !== 'settlement' ? (
                      <>
                        <Button variant="ghost" iconOnly aria-label={`复制${tx.title || '账单'}`} title="复制一笔" onClick={() => onCopy(tx, false)}>
                          <Copy size={17} />
                        </Button>
                        <Button variant="ghost" iconOnly aria-label={`将${tx.title || '账单'}存为模板`} title="存为模板" onClick={() => onCopy(tx, true)}>
                          <BookmarkPlus size={17} />
                        </Button>
                      </>
                    ) : null}
                    {canMutate ? (
                      <>
                        <span title={editReason || '编辑账单'}>
                          <Button
                            variant="ghost"
                            iconOnly
                            aria-label={editReason ? `无法编辑${tx.title || '账单'}：${editReason}` : `编辑${tx.title || '账单'}`}
                            disabled={Boolean(editReason)}
                            onClick={() => onEdit(tx)}
                          >
                            <Pencil size={17} />
                          </Button>
                        </span>
                        <Button variant="ghost" iconOnly aria-label={`删除${tx.title || '账单'}`} title="删除账单" onClick={() => onDelete(tx)}>
                          <Trash2 size={17} />
                        </Button>
                      </>
                    ) : null}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
