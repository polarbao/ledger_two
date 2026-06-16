import { X, Edit3, Trash2, CloudOff } from 'lucide-react';
import { useDraftStore } from '../../stores/draft.store';
import { useUIStore } from '../../stores/ui.store';

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function DraftListDrawer({ open, onClose }: Props) {
  const { drafts, removeDraft, clearDrafts } = useDraftStore();
  const { setEditingDraftId, setAddDrawerOpen, isOffline } = useUIStore();

  if (!open) return null;

  const handleEditDraft = (id: string) => {
    setEditingDraftId(id);
    setAddDrawerOpen(true);
    onClose();
  };

  const handleClearAll = () => {
    if (confirm('确定要清空所有离线草稿吗？')) {
      clearDrafts();
    }
  };

  return (
    <div className="drawer-overlay glass-blur show" onClick={onClose}>
      <div className="drawer-container glass-card" onClick={(e) => e.stopPropagation()}>
        <div className="drawer-header">
          <div className="header-title">
            <CloudOff className="title-icon text-glow" />
            <h3>离线草稿箱</h3>
          </div>
          <button className="btn-close-drawer" onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <div className="drawer-body p-4" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {isOffline && (
            <div className="error-banner" style={{ background: 'rgba(234, 179, 8, 0.1)', borderColor: 'rgba(234, 179, 8, 0.2)', color: '#ca8a04', margin: 0 }}>
              <p>当前仍处于离线状态，您可以继续编辑草稿，但无法提交为正式账单。</p>
            </div>
          )}

          {!isOffline && drafts.length > 0 && (
            <div className="success-banner" style={{ background: 'rgba(34, 197, 94, 0.1)', borderColor: 'rgba(34, 197, 94, 0.2)', color: '#22c55e', margin: 0, padding: '12px', borderRadius: '8px' }}>
              <p>网络已恢复，您可以点击下方草稿进行提交。</p>
            </div>
          )}

          {drafts.length === 0 ? (
            <div className="empty-state" style={{ textAlign: 'center', padding: '40px 0', color: 'var(--text-muted)' }}>
              暂无离线草稿
            </div>
          ) : (
            <>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>共 {drafts.length} 条草稿</span>
                <button 
                  className="btn-text" 
                  style={{ color: 'var(--accent-danger)', fontSize: '12px' }}
                  onClick={handleClearAll}
                >
                  全部清空
                </button>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {drafts.map((draft) => (
                  <div key={draft.id} className="transaction-card glass-card" style={{ padding: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <div style={{ fontWeight: '500', color: 'var(--text-primary)', marginBottom: '4px' }}>
                        {draft.formValues.type === 'expense' ? '支出' : draft.formValues.type === 'income' ? '收入' : '共同支出'} 
                        {' · '} 
                        {draft.formValues.amount ? `￥${draft.formValues.amount}` : '未填金额'}
                      </div>
                      <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
                        {draft.formValues.occurred_at} {draft.formValues.title ? ` · ${draft.formValues.title}` : ''}
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button 
                        className="btn-icon" 
                        onClick={() => handleEditDraft(draft.id)}
                        title="编辑/提交"
                      >
                        <Edit3 size={16} />
                      </button>
                      <button 
                        className="btn-icon danger" 
                        onClick={() => removeDraft(draft.id)}
                        title="删除草稿"
                      >
                        <Trash2 size={16} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
