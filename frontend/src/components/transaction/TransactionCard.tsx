import {
  Car,
  Heart,
  Home,
  Landmark,
  Layers,
  ShoppingBag,
  Utensils,
} from 'lucide-react';
import type { TransactionResponse } from '../../types/transaction';
import { formatDate } from '../../utils/date';
import { centsToYuan } from '../../utils/money';
import StatusChip from '../ui/StatusChip';
import { getTransactionPresentation } from './transactionsPageModel';
import './TransactionCard.css';

interface TransactionCardProps {
  tx: TransactionResponse;
  categoryName: string;
  payerName: string;
  selectable?: boolean;
  selected?: boolean;
  onSelectedChange?: () => void;
  onClick: () => void;
}

const getCategoryIcon = (category: string) => {
  if (category.includes('餐') || category.includes('食') || category.includes('饮')) return <Utensils size={19} />;
  if (category.includes('住') || category.includes('房') || category.includes('居')) return <Home size={19} />;
  if (category.includes('交通') || category.includes('出行') || category.includes('车')) return <Car size={19} />;
  if (category.includes('购物') || category.includes('娱乐')) return <ShoppingBag size={19} />;
  if (category.includes('医') || category.includes('健康')) return <Heart size={19} />;
  if (category.includes('其他')) return <Layers size={19} />;
  return <Landmark size={19} />;
};

export default function TransactionCard({
  tx,
  categoryName,
  payerName,
  selectable = false,
  selected = false,
  onSelectedChange,
  onClick,
}: TransactionCardProps) {
  const presentation = getTransactionPresentation(tx);

  return (
    <article className={`transaction-card${selected ? ' transaction-card--selected' : ''}`}>
      {selectable ? (
        <label className="transaction-card__select">
          <span className="sr-only">选择账单 {tx.title || '无标题'}</span>
          <input type="checkbox" checked={selected} onChange={onSelectedChange} />
        </label>
      ) : null}
      <button type="button" className="transaction-card__main" onClick={onClick}>
        <span className="transaction-card__icon" aria-hidden="true">
          {getCategoryIcon(categoryName)}
        </span>
        <span className="transaction-card__content">
          <span className="transaction-card__topline">
            <span className="transaction-card__title">{tx.title || '无标题'}</span>
            <strong className={`transaction-card__amount transaction-card__amount--${presentation.amountTone}`}>
              {presentation.amountPrefix}¥{centsToYuan(tx.amount_cents)}
            </strong>
          </span>
          <span className="transaction-card__meta">
            <StatusChip tone={presentation.typeTone}>{presentation.typeLabel}</StatusChip>
            <span>{categoryName}</span>
            <span>{payerName}付款</span>
            <time dateTime={tx.occurred_at}>{formatDate(tx.occurred_at).substring(0, 10)}</time>
          </span>
          <span className="transaction-card__scope">
            <span>{presentation.scopeLabel}</span>
            <span>{presentation.splitLabel}</span>
            {tx.tags?.slice(0, 2).map((tag) => <span key={tag}>#{tag}</span>)}
            {(tx.tags?.length || 0) > 2 ? <span>+{(tx.tags?.length || 0) - 2}</span> : null}
          </span>
        </span>
      </button>
    </article>
  );
}
