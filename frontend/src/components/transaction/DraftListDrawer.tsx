import { useState } from 'react';
import { AlertTriangle, CloudOff, Edit3, Trash2, Wifi } from 'lucide-react';
import { useDraftStore } from '../../stores/draft.store';
import { useUIStore } from '../../stores/ui.store';
import BottomSheet from '../ui/BottomSheet';
import Button from '../ui/Button';
import ConfirmDialog from '../ui/ConfirmDialog';
import StatePanel from '../ui/StatePanel';
import './DraftListDrawer.css';

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function DraftListDrawer({ open, onClose }: Props) {
  const { drafts, removeDraft, clearDrafts } = useDraftStore();
  const {
    setEditingDraftId,
    setAddDrawerOpen,
    setCopySourceTransaction,
    setEditSourceTransaction,
    isOffline,
  } = useUIStore();
  const [isClearConfirmOpen, setIsClearConfirmOpen] = useState(false);

  const handleEditDraft = (id: string) => {
    setCopySourceTransaction(null);
    setEditSourceTransaction(null);
    setEditingDraftId(id);
    setAddDrawerOpen(true);
    onClose();
  };

  const handleClose = () => {
    setIsClearConfirmOpen(false);
    onClose();
  };

  const handleClearAll = () => {
    clearDrafts();
    setIsClearConfirmOpen(false);
  };

  return (
    <>
      <BottomSheet
        open={open && !isClearConfirmOpen}
        title="离线草稿箱"
        description={drafts.length > 0 ? `共 ${drafts.length} 条本机草稿` : '草稿只保存在当前浏览器'}
        footer={drafts.length > 0 ? (
          <Button
            variant="ghost"
            className="draft-list__clear"
            startIcon={<Trash2 size={17} />}
            onClick={() => setIsClearConfirmOpen(true)}
          >
            全部清空
          </Button>
        ) : undefined}
        onClose={handleClose}
      >
        <div className="draft-list">
          {isOffline ? (
            <div className="draft-list__feedback draft-list__feedback--warning" role="status">
              <CloudOff size={18} aria-hidden="true" />
              <span>当前仍处于离线状态，可以继续编辑草稿，但暂时不能提交为正式账单。</span>
            </div>
          ) : drafts.length > 0 ? (
            <div className="draft-list__feedback draft-list__feedback--success" role="status">
              <Wifi size={18} aria-hidden="true" />
              <span>网络已恢复，可以打开草稿检查并提交。</span>
            </div>
          ) : null}

          {drafts.length === 0 ? (
            <StatePanel
              title="暂无离线草稿"
              description="离线记账后，尚未提交的内容会显示在这里。"
              icon={<CloudOff size={24} />}
            />
          ) : (
            <div className="draft-list__items" aria-label="离线草稿列表">
              {drafts.map((draft) => {
                const typeLabel = draft.formValues.type === 'expense'
                  ? '支出'
                  : draft.formValues.type === 'income'
                    ? '收入'
                    : '共同支出';
                const title = draft.formValues.title?.trim();

                return (
                  <article key={draft.id} className="draft-list__item">
                    <div className="draft-list__copy">
                      <strong className="draft-list__amount">
                        {typeLabel} · {draft.formValues.amount ? `¥${draft.formValues.amount}` : '未填金额'}
                      </strong>
                      <span className="draft-list__meta">
                        {draft.formValues.occurred_at}{title ? ` · ${title}` : ''}
                      </span>
                    </div>
                    <div className="draft-list__actions">
                      <Button
                        variant="secondary"
                        iconOnly
                        aria-label={`编辑${title || typeLabel}草稿`}
                        title="编辑并提交"
                        onClick={() => handleEditDraft(draft.id)}
                      >
                        <Edit3 size={17} />
                      </Button>
                      <Button
                        variant="ghost"
                        iconOnly
                        className="draft-list__delete"
                        aria-label={`删除${title || typeLabel}草稿`}
                        title="删除草稿"
                        onClick={() => removeDraft(draft.id)}
                      >
                        <Trash2 size={17} />
                      </Button>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </div>
      </BottomSheet>

      <ConfirmDialog
        open={open && isClearConfirmOpen}
        title="清空所有离线草稿？"
        description="草稿只保存在当前浏览器。清空后无法恢复，也不会删除已经提交的正式账单。"
        confirmLabel="清空全部草稿"
        tone="danger"
        icon={<AlertTriangle size={22} />}
        onConfirm={handleClearAll}
        onClose={() => setIsClearConfirmOpen(false)}
      />
    </>
  );
}
