import { useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Archive, BookOpen, FolderOpen, Plus } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { ApiError } from '../../api/client';
import { ledgerApi, type LedgerWithRole } from '../../api/ledger.api';
import { queryKeys } from '../../api/queryKeys';
import type { LedgerContextNotice } from '../../stores/ledger.store';
import { useLedgerStore } from '../../stores/ledger.store';
import Button from '../ui/Button';
import StatePanel from '../ui/StatePanel';
import StatusChip from '../ui/StatusChip';

const ledgerFormSchema = z.object({
  name: z.string().trim().min(1, '请输入账本名称').max(60, '账本名称最多 60 个字符'),
});

type LedgerFormValues = z.infer<typeof ledgerFormSchema>;

interface NoActiveLedgerShellProps {
  notice: LedgerContextNotice | null;
}

export default function NoActiveLedgerShell({ notice }: NoActiveLedgerShellProps) {
  const queryClient = useQueryClient();
  const setActiveLedger = useLedgerStore((state) => state.setActiveLedger);
  const clearContextNotice = useLedgerStore((state) => state.clearContextNotice);
  const [showArchived, setShowArchived] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<LedgerFormValues>({
    resolver: zodResolver(ledgerFormSchema),
    defaultValues: { name: '' },
  });

  const archivedQuery = useQuery({
    queryKey: queryKeys.ledgers.list('archived'),
    queryFn: ({ signal }) => ledgerApi.listUserLedgers('archived', signal),
    enabled: showArchived,
  });

  const createMutation = useMutation({
    mutationFn: ({ name }: LedgerFormValues) => ledgerApi.createLedger({ name: name.trim() }),
    onSuccess: (ledger) => {
      queryClient.setQueryData<LedgerWithRole[]>(
        queryKeys.ledgers.list('active'),
        (current = []) => [ledger, ...current.filter((item) => item.id !== ledger.id)],
      );
      setActiveLedger(ledger.id, ledger.role);
      clearContextNotice();
      reset();
    },
  });

  const errorMessage = createMutation.error instanceof ApiError
    ? createMutation.error.message
    : createMutation.isError
      ? '创建账本失败，请稍后重试'
      : null;

  return (
    <section className="lt-no-active" aria-labelledby="no-active-ledger-title">
      <div className="lt-no-active__intro">
        <StatePanel
          tone={notice ? 'warning' : 'neutral'}
          icon={<BookOpen size={40} />}
          title="暂无活跃账本"
          description={notice
            ? '原账本已归档或当前账号已失去访问权限。创建新账本，或查看仍可访问的归档账本。'
            : '创建一个账本后即可开始记账；归档账本不会自动成为当前账本。'}
        />
      </div>

      <div className="lt-no-active__actions">
        <form className="lt-no-active__create" onSubmit={handleSubmit((values) => createMutation.mutate(values))}>
          <div>
            <span className="lt-no-active__eyebrow">创建账本</span>
            <h2 id="no-active-ledger-title">建立新的记账空间</h2>
            <p>新账本会成为当前账本，创建者为唯一 Owner。</p>
          </div>
          <label>
            <span>账本名称</span>
            <input
              {...register('name')}
              type="text"
              maxLength={60}
              placeholder="例如：共同生活"
              aria-invalid={Boolean(errors.name)}
            />
            {errors.name ? <small role="alert">{errors.name.message}</small> : null}
          </label>
          {errorMessage ? <p className="lt-no-active__error" role="alert">{errorMessage}</p> : null}
          <Button
            type="submit"
            variant="primary"
            startIcon={<Plus size={17} />}
            isLoading={createMutation.isPending}
          >
            创建并进入账本
          </Button>
        </form>

        <section className="lt-no-active__archived" aria-labelledby="archived-ledgers-title">
          <div>
            <span className="lt-no-active__eyebrow">历史账本</span>
            <h2 id="archived-ledgers-title">查看已归档账本</h2>
            <p>归档账本保持只读，不会写入最近使用的活跃账本偏好。</p>
          </div>
          <Button
            variant="secondary"
            startIcon={<Archive size={17} />}
            onClick={() => setShowArchived((current) => !current)}
          >
            {showArchived ? '收起归档账本' : '查看已归档账本'}
          </Button>

          {showArchived ? (
            <div className="lt-no-active__archived-list" aria-live="polite">
              {archivedQuery.isLoading ? <p>正在读取归档账本...</p> : null}
              {archivedQuery.isError ? (
                <div className="lt-no-active__archived-error">
                  <p>归档账本读取失败。</p>
                  <Button variant="ghost" onClick={() => void archivedQuery.refetch()}>重试</Button>
                </div>
              ) : null}
              {archivedQuery.data?.length === 0 ? <p>暂无已归档账本。</p> : null}
              {archivedQuery.data?.map((ledger) => (
                <div key={ledger.id} className="lt-no-active__archived-row">
                  <FolderOpen size={17} aria-hidden="true" />
                  <span>
                    <strong>{ledger.name}</strong>
                    <small>{ledger.archived_at ? new Date(ledger.archived_at).toLocaleDateString('zh-CN') : '已归档'}</small>
                  </span>
                  <StatusChip tone="warning">只读</StatusChip>
                </div>
              ))}
            </div>
          ) : null}
        </section>
      </div>
    </section>
  );
}
