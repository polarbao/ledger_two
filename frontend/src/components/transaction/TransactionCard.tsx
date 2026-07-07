import { centsToYuan } from '../../utils/money';
import type { TransactionResponse } from '../../types/transaction';
import { Layers, ShoppingBag, Utensils, Home, Car, Heart, Landmark, SplitSquareVertical } from 'lucide-react';
import './TransactionCard.css';

interface TransactionCardProps {
  tx: TransactionResponse;
  currentUserId: string;
  categoryName?: string;
  onClick?: () => void;
}

const getCategoryIcon = (category: string) => {
  if (category.includes('餐') || category.includes('食') || category.includes('饮')) {
    return <Utensils size={20} />;
  }
  if (category.includes('住') || category.includes('房') || category.includes('居')) {
    return <Home size={20} />;
  }
  if (category.includes('交通') || category.includes('出行') || category.includes('车')) {
    return <Car size={20} />;
  }
  if (category.includes('购物') || category.includes('娱乐')) {
    return <ShoppingBag size={20} />;
  }
  if (category.includes('医') || category.includes('健康')) {
    return <Heart size={20} />;
  }

  const map: Record<string, React.ComponentType<{ size?: number }>> = {
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

export default function TransactionCard({ tx, currentUserId, categoryName, onClick }: TransactionCardProps) {
  const isPayer = tx.payer_user_id === currentUserId;
  const displayCategory = categoryName || (tx.category_id ? '已设分类' : '未分类');
  // If no specific split exists yet (e.g. older demo records), just fall back to equal display logic or single-person.
  
  return (
    <div className="transaction-card glass-card" onClick={onClick}>
      <div className="tc-icon-wrapper">
        {getCategoryIcon(displayCategory)}
      </div>
      
      <div className="tc-content">
        <div className="tc-header">
          <span className="tc-title">{tx.title}</span>
          <span className={`tc-amount ${isPayer ? 'amount-payer' : ''}`}>
            {centsToYuan(tx.amount_cents)}
          </span>
        </div>
        
        <div className="tc-footer">
          <span className="tc-category">{displayCategory}</span>
          <span className="tc-date">{new Date(tx.occurred_at).toLocaleDateString()}</span>
          {tx.split_method !== 'equal' && (
            <span className="tc-split-type">
              <SplitSquareVertical size={12} />
              {tx.split_method}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
