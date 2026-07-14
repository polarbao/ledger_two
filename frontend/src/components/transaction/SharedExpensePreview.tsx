import { Check, WalletCards } from 'lucide-react';
import { formatCny } from '../../utils/money';
import type { SharedExpensePreviewItem } from './transactionFormState';

interface SharedExpensePreviewProps {
  items: SharedExpensePreviewItem[];
  currentUserId: string | undefined;
}

export default function SharedExpensePreview({
  items,
  currentUserId,
}: SharedExpensePreviewProps) {
  return (
    <section className="lt-entry-preview" aria-labelledby="lt-entry-preview-title">
      <header className="lt-entry-preview__header">
        <div>
          <span className="lt-entry-section__eyebrow">共同支出</span>
          <h4 id="lt-entry-preview-title">承担预览</h4>
        </div>
        <WalletCards size={19} aria-hidden="true" />
      </header>

      <div className="lt-entry-preview__list">
        {items.map((item) => (
          <div className="lt-entry-preview__row" key={item.userId}>
            <span className={`lt-entry-preview__check ${item.isParticipating ? 'is-active' : ''}`} aria-hidden="true">
              {item.isParticipating ? <Check size={14} /> : null}
            </span>
            <div className="lt-entry-preview__member">
              <strong>
                {item.displayName}{item.userId === currentUserId ? '（我）' : ''}
              </strong>
              <span>{item.isPayer ? '付款人' : item.isParticipating ? '参与分摊' : '本次不承担'}</span>
            </div>
            <strong className="lt-entry-preview__amount">
              {formatCny(item.shareAmountCents)}
            </strong>
          </div>
        ))}
      </div>
      <p>这里是保存前预览，最终结果以服务端校验为准。</p>
    </section>
  );
}
