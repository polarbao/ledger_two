import { formatCents } from '../../utils/format';
import { Transaction } from '../../types/transaction';
import { Layers, ShoppingBag, Utensils, Home, Car, Heart, Landmark, SplitSquareVertical } from 'lucide-react';
import './TransactionCard.css';

interface TransactionCardProps {
  tx: Transaction;
  currentUserId: string;
  onClick?: () => void;
}

const getCategoryIcon = (category: string) => {
  const map: Record<string, any> = {
    'food': Utensils,
    'housing': Home,
    'transport': Car,
    'shopping': ShoppingBag,
    'health': Heart,
    'entertainment': Layers,
  };
  const Icon = map[category] || Landmark;
  return <Icon size={20} />;
};

export default function TransactionCard({ tx, currentUserId, onClick }: TransactionCardProps) {
  const isPayer = tx.paid_by === currentUserId;
  // If no specific split exists yet (e.g. older demo records), just fall back to equal display logic or single-person.
  
  return (
    <div className="transaction-card glass-card" onClick={onClick}>
      <div className="tc-icon-wrapper">
        {getCategoryIcon(tx.category)}
      </div>
      
      <div className="tc-content">
        <div className="tc-header">
          <span className="tc-title">{tx.title}</span>
          <span className={`tc-amount ${isPayer ? 'amount-payer' : ''}`}>
            {formatCents(tx.amount_cents)}
          </span>
        </div>
        
        <div className="tc-footer">
          <span className="tc-category">{tx.category}</span>
          <span className="tc-date">{new Date(tx.occurred_at).toLocaleDateString()}</span>
          {tx.split_type !== 'equal' && (
            <span className="tc-split-type">
              <SplitSquareVertical size={12} />
              {tx.split_type}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
