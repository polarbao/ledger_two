import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { AlertTriangle, Loader2 } from 'lucide-react';
import { safetyApi, BackupInfo } from '../../api/safety.api';

interface Props {
  backup: BackupInfo;
  onClose: () => void;
  onSuccess: (instructions: string) => void;
}

export default function RestoreBackupModal({ backup, onClose, onSuccess }: Props) {
  const [confirmText, setConfirmText] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const restoreMutation = useMutation({
    mutationFn: () => safetyApi.restoreBackup(backup.filename),
    onSuccess: (res) => {
      onSuccess(res.instructions);
    },
    onError: (err: any) => {
      setErrorMsg(err.message || '恢复前置流程失败，请重试');
    },
  });

  const handleConfirm = () => {
    if (confirmText !== '确认恢复') return;
    setErrorMsg(null);
    restoreMutation.mutate();
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
  };

  const formatTime = (ts: string) => {
    return new Date(ts).toLocaleString();
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content glass-card" onClick={(e) => e.stopPropagation()} style={{ maxWidth: '420px' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--accent-danger)' }}>
            <AlertTriangle size={24} />
            <h3 style={{ margin: 0 }}>安全恢复备份</h3>
          </div>

          <div style={{ background: 'var(--surface-color)', padding: '12px', borderRadius: '8px', fontSize: '13px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '80px 1fr', gap: '8px' }}>
              <span style={{ color: 'var(--text-muted)' }}>备份文件：</span>
              <span style={{ wordBreak: 'break-all' }}>{backup.filename}</span>
              <span style={{ color: 'var(--text-muted)' }}>文件大小：</span>
              <span>{formatSize(backup.size_bytes)}</span>
              <span style={{ color: 'var(--text-muted)' }}>备份时间：</span>
              <span>{formatTime(backup.created_at)}</span>
            </div>
          </div>

          <div style={{ background: 'rgba(239, 68, 68, 0.1)', padding: '12px', borderRadius: '8px', border: '1px solid rgba(239, 68, 68, 0.2)' }}>
            <p style={{ margin: 0, color: '#ef4444', fontSize: '14px', fontWeight: 500, marginBottom: '4px' }}>
              恢复备份会覆盖当前数据
            </p>
            <p style={{ margin: 0, color: '#ef4444', fontSize: '13px', opacity: 0.9 }}>
              恢复后，当前数据库中的账单、结算、分类、标签和设置将被备份内容替换。此操作不可直接撤销。
              为安全起见，系统将在恢复前自动创建一个当前数据的备份。
            </p>
          </div>

          {errorMsg && (
            <div style={{ color: 'var(--accent-danger)', fontSize: '13px', textAlign: 'center' }}>
              {errorMsg}
            </div>
          )}

          <div style={{ marginTop: '8px' }}>
            <label style={{ display: 'block', fontSize: '13px', marginBottom: '8px', color: 'var(--text-primary)' }}>
              请输入 <strong style={{ color: 'var(--accent-danger)' }}>确认恢复</strong> 以继续：
            </label>
            <input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder="确认恢复"
              className="form-input"
              style={{ width: '100%' }}
            />
          </div>

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', marginTop: '16px' }}>
            <button className="btn-secondary" onClick={onClose} disabled={restoreMutation.isPending}>
              取消
            </button>
            <button
              className="btn-primary danger"
              disabled={confirmText !== '确认恢复' || restoreMutation.isPending}
              onClick={handleConfirm}
            >
              {restoreMutation.isPending ? (
                <>
                  <Loader2 size={16} className="spinner" /> 恢复中...
                </>
              ) : (
                '执行恢复'
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
