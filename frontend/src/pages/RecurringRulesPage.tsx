import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm, Controller, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Link } from 'react-router-dom';
import { 
  Clock, 
  ArrowLeft, 
  Plus, 
  Trash2, 
  CheckCircle2,
  Calendar, 
  DollarSign, 
  Tag, 
  AlertTriangle, 
  Ban,
  X
} from 'lucide-react';
import { transactionsApi } from '../api/transactions.api';
import type { CreateRecurringRulePayload } from '../types/transaction';
import { useAuthStore } from '../stores/auth.store';
import { dashboardApi } from '../api/dashboard.api';
import { queryKeys } from '../api/queryKeys';
import { centsToYuan, yuanToCents } from '../utils/money';
import { useLedgerStore } from '../stores/ledger.store';
import PageState from '../components/ui/PageState';
import EmptyState from '../components/ui/EmptyState';
import PermissionGate from '../components/ledger/PermissionGate';

// 验证 Schema
const ruleSchema = z.object({
  name: z.string().min(1, '请输入规则名称').max(50, '规则名称最大 50 字'),
  type: z.enum(['expense', 'income', 'shared_expense']),
  amount: z.string()
    .refine((val) => {
      if (!val) return true;
      const parsed = parseFloat(val);
      return !isNaN(parsed) && parsed >= 0;
    }, { message: '金额不能为负数' }),
  category_id: z.string().optional(),
  payer_user_id: z.string().optional(),
  split_method: z.enum(['equal', 'payer_only']).optional(),
  tag_names: z.string().optional(),
  note: z.string().max(200, '备注最多支持 200 字').optional(),
  frequency: z.enum(['weekly', 'monthly', 'yearly']),
  next_due_date: z.string().min(1, '请选择首次执行到期日期'),
});

type RuleFormValues = z.infer<typeof ruleSchema>;

export default function RecurringRulesPage() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  
  // 二次确认删除 Modal 状态
  const [deleteTargetId, setDeleteTargetId] = useState<string | null>(null);

  // 1. 获取周期规则列表
  const { data: rules, isLoading: isLoadingRules, error: loadRulesError, refetch: refetchRules } = useQuery({
    queryKey: queryKeys.recurringRules(activeLedgerId),
    queryFn: () => transactionsApi.listRecurringRules(),
  });

  const { data: reminders, isLoading: isLoadingReminders } = useQuery({
    queryKey: queryKeys.recurringReminders(activeLedgerId),
    queryFn: () => transactionsApi.listRecurringReminders(),
  });
  const pendingReminders = reminders?.filter((item) => item.status === 'pending') || [];

  // 2. 获取分类列表
  const { data: categories } = useQuery({
    queryKey: queryKeys.categories(activeLedgerId),
    queryFn: () => transactionsApi.getCategories(),
  });

  const catMap = categories?.reduce((acc, cat) => {
    acc[cat.id] = cat.name;
    return acc;
  }, {} as Record<string, string>) || {};

  // 3. 获取成员（由于是双人账本，可以直接用 dashboard 近期月份拿 user_stats）
  const currentMonth = new Date().toISOString().substring(0, 7);
  const { data: dashboardData } = useQuery({
    queryKey: queryKeys.dashboard.month(activeLedgerId, currentMonth),
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: !!currentUser,
  });

  const users = dashboardData?.user_stats || [];

  // 获取付款人 Display Name
  const getUserDisplayName = (userId: string) => {
    if (userId === currentUser?.id) return '我';
    const other = users.find((u) => u.user_id === userId);
    return other ? other.display_name : '对方';
  };

  const invalidateRecurringFlow = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.recurringRules(activeLedgerId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.recurringReminders(activeLedgerId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(activeLedgerId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(activeLedgerId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.settlements.root(activeLedgerId) });
    queryClient.invalidateQueries({ queryKey: queryKeys.reports.root(activeLedgerId) });
  };

  const getTodayString = () => {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  // 4. 表单绑定
  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<RuleFormValues>({
    resolver: zodResolver(ruleSchema),
    defaultValues: {
      name: '',
      type: 'expense',
      amount: '',
      category_id: '',
      payer_user_id: currentUser?.id || '',
      split_method: 'equal',
      tag_names: '',
      note: '',
      frequency: 'monthly',
      next_due_date: getTodayString(),
    },
  });

  const watchType = useWatch({ control, name: 'type' });

  // 5. 创建规则 Mutation
  const createMutation = useMutation({
    mutationFn: (payload: CreateRecurringRulePayload) => transactionsApi.createRecurringRule(payload),
    onSuccess: () => {
      setSuccessMsg('周期账单规则创建成功！');
      queryClient.invalidateQueries({ queryKey: queryKeys.recurringRules(activeLedgerId) });
      reset({
        name: '',
        type: 'expense',
        amount: '',
        category_id: '',
        payer_user_id: currentUser?.id || '',
        split_method: 'equal',
        tag_names: '',
        note: '',
        frequency: 'monthly',
        next_due_date: getTodayString(),
      });
      setTimeout(() => setSuccessMsg(null), 3000);
    },
    onError: (err: unknown) => {
      const error = err as Error;
      setErrorMsg(error.message || '创建周期规则失败，请检查填写内容');
      setTimeout(() => setErrorMsg(null), 4000);
    },
  });

  // 6. 删除规则 Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.deleteRecurringRule(id),
    onSuccess: () => {
      setSuccessMsg('周期规则已成功删除！');
      queryClient.invalidateQueries({ queryKey: queryKeys.recurringRules(activeLedgerId) });
      setDeleteTargetId(null);
      setTimeout(() => setSuccessMsg(null), 3000);
    },
    onError: (err: unknown) => {
      const error = err as Error;
      setErrorMsg(error.message || '删除周期规则失败');
      setDeleteTargetId(null);
      setTimeout(() => setErrorMsg(null), 4000);
    },
  });

  const confirmReminderMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.confirmReminder(id),
    onSuccess: () => {
      setSuccessMsg('周期账单已确认生成真实账单！');
      invalidateRecurringFlow();
      setTimeout(() => setSuccessMsg(null), 3000);
    },
    onError: (err: unknown) => {
      const error = err as Error;
      setErrorMsg(error.message || '确认周期账单失败');
      setTimeout(() => setErrorMsg(null), 4000);
    },
  });

  const skipReminderMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.skipReminder(id),
    onSuccess: () => {
      setSuccessMsg('已跳过本期周期账单提醒。');
      invalidateRecurringFlow();
      setTimeout(() => setSuccessMsg(null), 3000);
    },
    onError: (err: unknown) => {
      const error = err as Error;
      setErrorMsg(error.message || '跳过周期账单失败');
      setTimeout(() => setErrorMsg(null), 4000);
    },
  });

  const onSubmit = (values: RuleFormValues) => {
    const amountCents = values.amount ? yuanToCents(values.amount) : undefined;
    const tags = values.tag_names
      ? values.tag_names.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean)
      : [];

    const payload: CreateRecurringRulePayload = {
      name: values.name.trim(),
      type: values.type,
      amount_cents: amountCents,
      category_id: values.category_id || undefined,
      tag_names: tags,
      note: values.note || undefined,
      frequency: values.frequency,
      next_due_date: values.next_due_date,
      payer_user_id: values.type === 'shared_expense' ? values.payer_user_id : currentUser?.id,
      split_method: values.type === 'shared_expense' ? (values.split_method || 'equal') : undefined,
    };

    createMutation.mutate(payload);
  };

  const confirmDelete = () => {
    if (deleteTargetId) {
      deleteMutation.mutate(deleteTargetId);
    }
  };

  return (
    <div className="page-content animate-fade-in text-left">
      {/* 头部区 */}
      <div className="glass-card header-banner" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Clock className="banner-icon" style={{ color: 'var(--accent-purple)' }} />
          <div>
            <h2>周期账单规则</h2>
            <p>配置自动到期提醒规则，减轻每月固定消费的手动录入负担</p>
          </div>
        </div>
        <Link to="/settings" className="btn-secondary" style={{ display: 'flex', alignItems: 'center', gap: '6px', textDecoration: 'none', padding: '8px 16px', fontSize: '13px' }}>
          <ArrowLeft size={14} /> 返回设置
        </Link>
      </div>

      {/* 提示消息 */}
      {successMsg && (
        <div className="glass-card text-green animate-fade-in" style={{ padding: '12px 20px', margin: '0 0 16px 0', borderRadius: '12px', background: 'rgba(16, 185, 129, 0.06)', border: '1px solid rgba(16, 185, 129, 0.2)' }}>
          <span>{successMsg}</span>
        </div>
      )}
      {errorMsg && (
        <div className="error-banner animate-fade-in" style={{ margin: '0 0 16px 0', borderRadius: '12px' }}>
          <AlertTriangle size={18} style={{ marginRight: '8px', flexShrink: 0 }} />
          <span>{errorMsg}</span>
        </div>
      )}

      {(isLoadingReminders || pendingReminders.length > 0) && (
        <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '14px', marginBottom: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px', flexWrap: 'wrap', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
              <Calendar size={20} className="partner-highlight" />
              <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>待确认周期账单</h3>
            </div>
            <span className="dimmed-desc" style={{ fontSize: '12px' }}>
              确认后才会生成真实账单；跳过只影响本期提醒。
            </span>
          </div>

          {isLoadingReminders ? (
            <div className="skeleton-item" style={{ height: '80px', borderRadius: '12px' }} />
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {pendingReminders.map((reminder) => (
                <div
                  key={reminder.id}
                  className="recurring-reminder-card"
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                      <span style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
                        {reminder.rule_name}
                      </span>
                      <span style={{ fontSize: '11px', background: 'rgba(168,85,247,0.1)', color: 'var(--accent-purple)', padding: '1px 6px', borderRadius: '4px', border: '1px solid rgba(168,85,247,0.2)' }}>
                        到期 {reminder.due_date}
                      </span>
                      <span className={`type-badge ${reminder.type === 'shared_expense' ? 'badge-shared' : reminder.type === 'income' ? 'badge-income' : 'badge-expense'}`}>
                        {reminder.type === 'shared_expense' ? '共同支出' : reminder.type === 'income' ? '收入' : '支出'}
                      </span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap', fontSize: '12px', color: 'var(--text-secondary)' }}>
                      <span>金额: {reminder.amount_cents != null ? `¥${centsToYuan(reminder.amount_cents)}` : '未设定'}</span>
                      {reminder.category_id && <span>分类: {catMap[reminder.category_id] || reminder.category_name || '加载中'}</span>}
                      {reminder.payer_user_id && <span>付款人: {getUserDisplayName(reminder.payer_user_id)}</span>}
                      {reminder.type === 'shared_expense' && (
                        <span>分摊: {reminder.split_method === 'payer_only' ? '付款人全额' : '均等平分'}</span>
                      )}
                    </div>
                  </div>

                  <PermissionGate allow={['owner', 'editor']}>
                    <div className="recurring-reminder-actions">
                      <button
                        type="button"
                        className="btn-secondary mobile-full"
                        style={{ padding: '8px 12px', fontSize: '13px', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                        disabled={confirmReminderMutation.isPending || skipReminderMutation.isPending}
                        onClick={() => skipReminderMutation.mutate(reminder.id)}
                      >
                        <Ban size={14} />
                        <span>跳过本期</span>
                      </button>
                      <button
                        type="button"
                        className="btn-primary mobile-full"
                        style={{ padding: '8px 12px', fontSize: '13px', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                        disabled={confirmReminderMutation.isPending || skipReminderMutation.isPending}
                        onClick={() => confirmReminderMutation.mutate(reminder.id)}
                      >
                        <CheckCircle2 size={14} />
                        <span>确认入账</span>
                      </button>
                    </div>
                  </PermissionGate>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <PageState 
        isLoading={isLoadingRules}
        isError={!!loadRulesError}
        errorMsg="加载周期规则列表失败"
        onRetry={() => refetchRules()}
      >
        <div className="form-row-2">
          {/* 左栏：新建周期规则表单 */}
          <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
              <Plus size={20} className="partner-highlight" />
              <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>新建周期记账规则</h3>
            </div>

            <form onSubmit={handleSubmit(onSubmit)} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {/* 规则名称 */}
              <div className="form-group">
                <label className="form-label">规则名称</label>
                <input 
                  type="text" 
                  placeholder="例如: 房租、宽带费、视频会员" 
                  className={`form-input ${errors.name ? 'input-error' : ''}`}
                  {...register('name')} 
                />
                {errors.name && <span className="field-error">{errors.name.message}</span>}
              </div>

              {/* 账单类型 */}
              <div className="form-group">
                <label className="form-label">账单类型</label>
                <Controller
                  name="type"
                  control={control}
                  render={({ field }) => (
                    <div className="segmented-control">
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'expense' ? 'active' : ''}`}
                        onClick={() => field.onChange('expense')}
                      >
                        个人支出
                      </button>
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'income' ? 'active' : ''}`}
                        onClick={() => field.onChange('income')}
                      >
                        个人收入
                      </button>
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'shared_expense' ? 'active' : ''}`}
                        onClick={() => field.onChange('shared_expense')}
                      >
                        共同支出
                      </button>
                    </div>
                  )}
                />
              </div>

              {/* 金额 */}
              <div className="form-group">
                <label className="form-label">金额 (元，可选)</label>
                <div style={{ position: 'relative' }}>
                  <input
                    type="number"
                    step="0.01"
                    placeholder="不填则提醒时不预设金额"
                    className={`form-input ${errors.amount ? 'input-error' : ''}`}
                    {...register('amount')}
                  />
                </div>
                {errors.amount && <span className="field-error">{errors.amount.message}</span>}
              </div>

              {/* 所属分类 */}
              <div className="form-group">
                <label className="form-label">所属分类</label>
                <select className="form-select" {...register('category_id')}>
                  <option value="">-- 请选择分类 (选填) --</option>
                  {categories?.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* 周期频次 */}
              <div className="form-group">
                <label className="form-label">重复周期频次</label>
                <Controller
                  name="frequency"
                  control={control}
                  render={({ field }) => (
                    <div className="segmented-control">
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'weekly' ? 'active' : ''}`}
                        onClick={() => field.onChange('weekly')}
                      >
                        每周 (Weekly)
                      </button>
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'monthly' ? 'active' : ''}`}
                        onClick={() => field.onChange('monthly')}
                      >
                        每月 (Monthly)
                      </button>
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'yearly' ? 'active' : ''}`}
                        onClick={() => field.onChange('yearly')}
                      >
                        每年 (Yearly)
                      </button>
                    </div>
                  )}
                />
              </div>

              {/* 首次触发到期日期 */}
              <div className="form-group">
                <label className="form-label">首次执行提醒到期日期</label>
                <input 
                  type="date" 
                  className={`form-input ${errors.next_due_date ? 'input-error' : ''}`}
                  {...register('next_due_date')} 
                />
                {errors.next_due_date && <span className="field-error">{errors.next_due_date.message}</span>}
                <p className="dimmed-desc" style={{ fontSize: '11px', marginTop: '4px' }}>
                  从该日期开始，系统在您访问时自动生成 pending 到期提醒。
                </p>
              </div>

              {/* 共同支出特定字段 */}
              {watchType === 'shared_expense' && (
                <>
                  <div className="form-group">
                    <label className="form-label">付款人</label>
                    <select className="form-select" {...register('payer_user_id')}>
                      {users.map((u) => (
                        <option key={u.user_id} value={u.user_id}>
                          {u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div className="form-group">
                    <label className="form-label">分摊方式</label>
                    <Controller
                      name="split_method"
                      control={control}
                      render={({ field }) => (
                        <div className="segmented-control">
                          <button
                            type="button"
                            className={`segment-btn ${field.value === 'equal' ? 'active' : ''}`}
                            onClick={() => field.onChange('equal')}
                          >
                            均等平分
                          </button>
                          <button
                            type="button"
                            className={`segment-btn ${field.value === 'payer_only' ? 'active' : ''}`}
                            onClick={() => field.onChange('payer_only')}
                          >
                            付款人全额
                          </button>
                        </div>
                      )}
                    />
                  </div>
                </>
              )}

              {/* 标签 */}
              <div className="form-group">
                <label className="form-label">标签 (选填)</label>
                <input 
                  type="text" 
                  placeholder="多个标签以空格或逗号分隔" 
                  className="form-input"
                  {...register('tag_names')} 
                />
              </div>

              {/* 备注 */}
              <div className="form-group">
                <label className="form-label">备注 (选填)</label>
                <textarea 
                  placeholder="该周期下自动产生的账单默认备注..." 
                  className="form-input textarea" 
                  rows={2}
                  {...register('note')}
                />
              </div>

              <PermissionGate allow={['owner', 'editor']}>
                <button
                  type="submit"
                  className="btn-primary"
                  style={{ width: '100%', padding: '12px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', fontSize: '14px', fontWeight: 600, marginTop: '8px' }}
                  disabled={isSubmitting || createMutation.isPending}
                >
                  {isSubmitting || createMutation.isPending ? '创建中...' : '保存并启用该周期规则'}
                </button>
              </PermissionGate>
            </form>
          </div>

          {/* 右栏：已有周期规则列表 */}
          <div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', borderBottom: '1px solid rgba(255, 255, 255, 0.05)', paddingBottom: '12px' }}>
              <Clock size={20} className="partner-highlight" />
              <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 600 }}>当前已启用的周期规则列表</h3>
            </div>

            {(!rules || rules.length === 0) ? (
              <EmptyState 
                title="暂无周期记账规则"
                description="目前尚未创建任何周期记账提醒规则。您可以通过左侧表单，为房租、固定账单设置周期生成机制。"
              />
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', maxHeight: '720px', overflowY: 'auto', paddingRight: '4px' }}>
                {rules.map((rule) => {
                  let typeBadge = '';
                  let typeLabel = '';
                  switch (rule.type) {
                    case 'expense':
                      typeBadge = 'badge-expense';
                      typeLabel = '个人';
                      break;
                    case 'income':
                      typeBadge = 'badge-income';
                      typeLabel = '收入';
                      break;
                    case 'shared_expense':
                      typeBadge = 'badge-shared';
                      typeLabel = '共享';
                      break;
                  }

                  let freqLabel = '';
                  switch (rule.frequency) {
                    case 'weekly': freqLabel = '每周'; break;
                    case 'monthly': freqLabel = '每月'; break;
                    case 'yearly': freqLabel = '每年'; break;
                  }

                  return (
                    <div 
                      key={rule.id}
                      style={{ 
                        background: 'rgba(255,255,255,0.02)', 
                        border: '1px solid rgba(255,255,255,0.03)', 
                        borderRadius: '12px', 
                        padding: '16px', 
                        display: 'flex', 
                        flexDirection: 'column', 
                        gap: '12px' 
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '12px' }}>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                            <span style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-primary)' }}>
                              {rule.name}
                            </span>
                            <span className={`type-badge ${typeBadge}`}>
                              {typeLabel}
                            </span>
                            <span style={{ fontSize: '11px', background: 'rgba(168,85,247,0.1)', color: 'var(--accent-purple)', padding: '1px 6px', borderRadius: '4px', border: '1px solid rgba(168,85,247,0.2)' }}>
                              {freqLabel}
                            </span>
                          </div>
                          <span className="dimmed-desc" style={{ fontSize: '12px', marginTop: '2px' }}>
                            规则标题: {rule.title || '与规则名称相同'}
                          </span>
                        </div>

                        <PermissionGate allow={['owner', 'editor']}>
                          <button
                            onClick={() => setDeleteTargetId(rule.id)}
                            className="btn-close-drawer"
                            style={{ padding: '6px', color: 'var(--text-muted)' }}
                            title="删除规则"
                            onMouseEnter={(e) => {
                              e.currentTarget.style.color = '#ef4444';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.color = 'var(--text-muted)';
                            }}
                          >
                            <Trash2 size={16} />
                          </button>
                        </PermissionGate>
                      </div>

                      {/* 规则细节网格 */}
                      <div 
                        style={{ 
                          display: 'grid', 
                          gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))', 
                          gap: '8px 16px',
                          background: 'rgba(0,0,0,0.1)', 
                          padding: '10px 12px', 
                          borderRadius: '8px',
                          fontSize: '12px',
                          color: 'var(--text-secondary)'
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <DollarSign size={13} className="text-green" />
                          <span>金额: {rule.amount_cents != null ? `¥${centsToYuan(rule.amount_cents)}` : '未设定'}</span>
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <Calendar size={13} />
                          <span>下期到期: {rule.next_due_date}</span>
                        </div>
                        {rule.category_id && (
                          <div>分类: {catMap[rule.category_id] || '加载中'}</div>
                        )}
                        {rule.type === 'shared_expense' && (
                          <>
                            <div>付款人: {getUserDisplayName(rule.payer_user_id)}</div>
                            <div>分摊: {rule.split_method === 'payer_only' ? '付款人全额' : '均等平分'}</div>
                          </>
                        )}
                      </div>

                      {/* 标签与备注展示 */}
                      {(rule.tag_names?.length > 0 || rule.note) && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: '12px' }}>
                          {rule.tag_names?.length > 0 && (
                            <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', alignItems: 'center' }}>
                              <Tag size={12} style={{ color: 'var(--text-muted)' }} />
                              {rule.tag_names.map((t) => (
                                <span key={t} style={{ fontSize: '11px', color: 'var(--text-muted)', background: 'rgba(255,255,255,0.03)', padding: '1px 6px', borderRadius: '4px', border: '1px solid rgba(255,255,255,0.05)' }}>
                                  #{t}
                                </span>
                              ))}
                            </div>
                          )}
                          {rule.note && (
                            <div style={{ color: 'var(--text-muted)', fontStyle: 'italic', background: 'rgba(255,255,255,0.01)', padding: '6px 10px', borderRadius: '6px', borderLeft: '2px solid rgba(255,255,255,0.1)' }}>
                              注: {rule.note}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </PageState>

      {/* ==========================================
         二次确认删除 Modal (Danger 警示样式按钮)
         ========================================== */}
      {deleteTargetId && (
        <div className="drawer-overlay show" style={{ alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="confirm-modal-box animate-fade-in" style={{ maxWidth: '400px' }}>
            <div className="drawer-header" style={{ padding: '16px 20px' }}>
              <div className="header-title" style={{ color: '#ef4444' }}>
                <AlertTriangle className="title-icon" style={{ color: 'inherit' }} />
                <h3 style={{ fontSize: '16px' }}>删除周期记账规则？</h3>
              </div>
              <button className="btn-close-drawer" onClick={() => setDeleteTargetId(null)}>
                <X size={18} />
              </button>
            </div>

            <div className="modal-body-padding" style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <p className="modal-alert-text">
                确认删除这条周期规则吗？删除规则仅会停止未来的到期提醒，<strong className="text-expense">不会影响您以往由该规则生成并确认的历史交易账单记录</strong>。
              </p>

              <div className="drawer-footer" style={{ borderTop: 'none', paddingTop: 0, marginTop: '8px', display: 'flex', gap: '10px', justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                <button className="btn-secondary mobile-full" style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} onClick={() => setDeleteTargetId(null)}>
                  取消
                </button>
                <button className="btn-danger mobile-full" style={{ padding: '10px 20px', fontSize: '14px', borderRadius: '10px' }} onClick={confirmDelete}>
                  确认删除
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
