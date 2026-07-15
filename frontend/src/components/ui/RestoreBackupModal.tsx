import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { AlertTriangle } from 'lucide-react';
import { safetyApi, type BackupInfo } from '../../api/safety.api';
import ConfirmDialog from './ConfirmDialog';
import './RestoreBackupModal.css';

interface Props {
  backup: BackupInfo;
  onClose: () => void;
  onSuccess: (instructions: string) => void;
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export default function RestoreBackupModal({ backup, onClose, onSuccess }: Props) {
  const [confirmText, setConfirmText] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const restoreMutation = useMutation({
    mutationFn: () => safetyApi.restoreBackup(backup.filename),
    onSuccess: (response) => onSuccess(response.instructions),
    onError: (error: unknown) => {
      setErrorMsg(error instanceof Error ? error.message : '恢复准备流程失败，请重试');
    },
  });

  const closeSafely = () => {
    if (!restoreMutation.isPending) onClose();
  };

  return (
    <ConfirmDialog
      open
      title="准备从备份恢复数据库？"
      description="系统不会在线覆盖当前数据库。本步骤会先创建当前数据的安全备份，再返回停机后人工替换数据库的操作指引。"
      confirmLabel="创建前置备份并生成指引"
      tone="danger"
      icon={<AlertTriangle />}
      isConfirming={restoreMutation.isPending}
      confirmDisabled={confirmText !== '确认恢复'}
      onClose={closeSafely}
      onConfirm={() => {
        setErrorMsg(null);
        restoreMutation.mutate();
      }}
    >
      <div className="restore-backup">
        <dl className="restore-backup__file">
          <div><dt>备份文件</dt><dd>{backup.filename}</dd></div>
          <div><dt>文件大小</dt><dd>{formatSize(backup.size_bytes)}</dd></div>
          <div><dt>备份时间</dt><dd>{new Date(backup.created_at).toLocaleString('zh-CN', { hour12: false })}</dd></div>
        </dl>

        <div className="restore-backup__warning">
          <strong>最终人工替换会覆盖运行数据</strong>
          <p>停机替换后，账单、结算、分类、标签和设置将回到该备份的时间点。系统创建的前置备份可用于回退。</p>
        </div>

        {errorMsg ? <div className="restore-backup__error" role="alert">{errorMsg}</div> : null}

        <label className="restore-backup__confirm-field">
          <span>输入 <strong>确认恢复</strong> 以继续</span>
          <input
            type="text"
            value={confirmText}
            onChange={(event) => setConfirmText(event.target.value)}
            placeholder="确认恢复"
            autoComplete="off"
          />
        </label>
      </div>
    </ConfirmDialog>
  );
}
