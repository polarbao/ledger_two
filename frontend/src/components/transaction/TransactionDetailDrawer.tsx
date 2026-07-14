import { useCallback, useEffect, useId, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react';
import { createPortal } from 'react-dom';
import { BookmarkPlus, Copy, Image, Pencil, ReceiptText, Trash2, X } from 'lucide-react';
import type { TransactionResponse } from '../../types/transaction';
import { formatDate } from '../../utils/date';
import { centsToYuan } from '../../utils/money';
import Button from '../ui/Button';
import StatusChip from '../ui/StatusChip';
import useModalSurface from '../ui/useModalSurface';
import { getTransactionPresentation } from './transactionsPageModel';

interface TransactionDetailDrawerProps {
  open: boolean;
  transaction: TransactionResponse | null;
  currentUserId: string;
  canWrite: boolean;
  categoryLabel: string;
  payerName: string;
  userName: (userId: string) => string;
  onClose: () => void;
  onCopy: (transaction: TransactionResponse, saveAsTemplate: boolean) => void;
  onEdit: (transaction: TransactionResponse) => void;
  onDelete: (transaction: TransactionResponse) => void;
  editBlockReason: string | null;
}

export default function TransactionDetailDrawer({
  open,
  transaction,
  currentUserId,
  canWrite,
  categoryLabel,
  payerName,
  userName,
  onClose,
  onCopy,
  onEdit,
  onDelete,
  editBlockReason,
}: TransactionDetailDrawerProps) {
  const titleId = useId();
  const surfaceRef = useRef<HTMLElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const lightboxCloseRef = useRef<HTMLButtonElement>(null);
  const [lightboxImage, setLightboxImage] = useState<string | null>(null);
  const lightboxImageRef = useRef(lightboxImage);
  const closeDrawerOrLightbox = useCallback(() => {
    if (lightboxImageRef.current) setLightboxImage(null);
    else onClose();
  }, [onClose]);
  useModalSurface({ open, onClose: closeDrawerOrLightbox, surfaceRef, initialFocusRef: closeRef });

  useEffect(() => {
    lightboxImageRef.current = lightboxImage;
    if (lightboxImage) lightboxCloseRef.current?.focus();
  }, [lightboxImage]);

  if (!open || !transaction) return null;

  const tx = transaction;
  const presentation = getTransactionPresentation(tx);
  const canDelete = canWrite && tx.created_by_user_id === currentUserId && tx.type !== 'settlement';
  const canCopy = canWrite && tx.type !== 'settlement';
  const closeFromBackdrop = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) onClose();
  };
  const closeLightboxFromKeyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Escape') return;
    event.preventDefault();
    event.stopPropagation();
    setLightboxImage(null);
  };

  const drawer = (
    <>
      <div className="transaction-detail-overlay" onMouseDown={closeFromBackdrop}>
        <section
          ref={surfaceRef}
          className="transaction-detail-drawer"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          tabIndex={-1}
        >
          <header className="transaction-detail-drawer__header">
            <div>
              <ReceiptText size={20} aria-hidden="true" />
              <h2 id={titleId}>账单详情</h2>
            </div>
            <Button ref={closeRef} variant="ghost" iconOnly aria-label="关闭账单详情" title="关闭" onClick={onClose}>
              <X size={20} />
            </Button>
          </header>

          <div className="transaction-detail-drawer__body">
            <section className="transaction-detail-drawer__amount" aria-label="交易金额">
              <StatusChip tone={presentation.typeTone}>{presentation.typeLabel}</StatusChip>
              <strong className={`transaction-detail-drawer__amount-value transaction-detail-drawer__amount-value--${presentation.amountTone}`}>
                {presentation.amountPrefix}¥{centsToYuan(tx.amount_cents)}
              </strong>
              <span>{tx.title || '无标题'}</span>
            </section>

            <dl className="transaction-detail-list">
              <div><dt>分类</dt><dd>{categoryLabel}</dd></div>
              <div><dt>发生时间</dt><dd>{formatDate(tx.occurred_at).substring(0, 16)}</dd></div>
              <div><dt>付款人</dt><dd>{payerName}</dd></div>
              <div><dt>可见范围</dt><dd>{presentation.scopeLabel}</dd></div>
              <div><dt>账单归属</dt><dd>{presentation.splitLabel}</dd></div>
            </dl>

            {tx.type === 'shared_expense' ? (
              <section className="transaction-detail-section">
                <h3>承担明细</h3>
                <dl className="transaction-detail-splits">
                  {tx.participants?.length ? tx.participants.map((participant) => (
                    <div key={participant.user_id}>
                      <dt>{userName(participant.user_id)}</dt>
                      <dd>¥{centsToYuan(participant.share_amount_cents)}</dd>
                    </div>
                  )) : <div><dt>暂无明细</dt><dd>以服务端记录为准</dd></div>}
                </dl>
              </section>
            ) : null}

            {tx.tags?.length ? (
              <section className="transaction-detail-section">
                <h3>标签</h3>
                <div className="transaction-detail-tags">
                  {tx.tags.map((tag) => <StatusChip key={tag} tone="neutral">#{tag}</StatusChip>)}
                </div>
              </section>
            ) : null}

            {tx.note ? (
              <section className="transaction-detail-section">
                <h3>备注</h3>
                <p>{tx.note}</p>
              </section>
            ) : null}

            {tx.attachment_paths?.length ? (
              <section className="transaction-detail-section">
                <h3>附件</h3>
                <div className="transaction-detail-attachments">
                  {tx.attachment_paths.map((path, index) => (
                    <button key={path} type="button" onClick={() => setLightboxImage(path)}>
                      <img src={path} alt={`账单附件 ${index + 1}`} />
                    </button>
                  ))}
                </div>
              </section>
            ) : (
              <section className="transaction-detail-section transaction-detail-section--muted">
                <Image size={17} aria-hidden="true" />
                <span>无附件</span>
              </section>
            )}
          </div>

          <footer className="transaction-detail-drawer__footer">
            {canDelete ? (
              <Button variant="ghost" startIcon={<Trash2 size={17} />} onClick={() => onDelete(tx)}>
                删除账单
              </Button>
            ) : null}
            <div>
              {canDelete ? (
                <span title={editBlockReason || '编辑账单'}>
                  <Button
                    variant="secondary"
                    startIcon={<Pencil size={17} />}
                    disabled={Boolean(editBlockReason)}
                    onClick={() => onEdit(tx)}
                  >
                    {editBlockReason ? '暂不可编辑' : '编辑账单'}
                  </Button>
                </span>
              ) : null}
              {canCopy ? (
                <>
                  <Button variant="secondary" startIcon={<BookmarkPlus size={17} />} onClick={() => onCopy(tx, true)}>
                    存为模板
                  </Button>
                  <Button variant="primary" startIcon={<Copy size={17} />} onClick={() => onCopy(tx, false)}>
                    复制一笔
                  </Button>
                </>
              ) : (
                <Button variant="secondary" onClick={onClose}>关闭</Button>
              )}
            </div>
          </footer>

          {lightboxImage ? (
            <div
              className="transaction-lightbox"
              role="dialog"
              aria-modal="true"
              aria-label="查看账单附件"
              onKeyDown={closeLightboxFromKeyboard}
              onMouseDown={() => setLightboxImage(null)}
            >
              <div onMouseDown={(event) => event.stopPropagation()}>
                <img src={lightboxImage} alt="账单附件大图" />
                <Button
                  ref={lightboxCloseRef}
                  variant="secondary"
                  iconOnly
                  aria-label="关闭附件预览"
                  title="关闭"
                  onClick={() => setLightboxImage(null)}
                >
                  <X size={20} />
                </Button>
              </div>
            </div>
          ) : null}
        </section>
      </div>
    </>
  );

  return typeof document === 'undefined' ? drawer : createPortal(drawer, document.body);
}
