import { useEffect } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Loader2, Sparkles, Check } from 'lucide-react';
import { useUIStore } from '../../stores/ui.store';
import { useAuthStore } from '../../stores/auth.store';
import { transactionsApi } from '../../api/transactions.api';
import { dashboardApi } from '../../api/dashboard.api';
import { yuanToCents } from '../../utils/money';

/**
 * @brief 表单校验 Schema 结构定义
 */
const formSchema = z.object({
  type: z.enum(['expense', 'income', 'shared_expense']),
  amount: z.string()
    .min(1, '请输入金额')
    .refine((val) => {
      const parsed = parseFloat(val);
      return !isNaN(parsed) && parsed > 0;
    }, { message: '金额必须大于 0' }),
  title: z.string().max(100, '标题最大支持 100 字').optional(),
  category_id: z.string().optional().nullable(),
  tag_names: z.string().optional(),
  payer_user_id: z.string().min(1, '请选择付款人'),
  split_method: z.enum(['equal', 'payer_only']),
  occurred_at: z.string().min(1, '请选择发生日期'),
  note: z.string().max(200, '备注最多支持 200 字').optional(),
  visibility: z.enum(['private', 'partner_readable']),
});

type FormValues = z.infer<typeof formSchema>;

/**
 * @brief 记账滑出层组件 (TransactionFormDrawer)
 * @details 兼容普通支出/收入与共同账单创建，支持电脑右滑与手机底滑布局。
 * @return React.ReactElement 返回渲染的 React 节点
 */
export default function TransactionFormDrawer() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const { addDrawerOpen, setAddDrawerOpen, currentMonth } = useUIStore();

  // 1. 获取全量分类列表
  const { data: categories, isLoading: isCategoriesLoading } = useQuery({
    queryKey: ['categories'],
    queryFn: () => transactionsApi.getCategories(),
    enabled: addDrawerOpen,
  });

  // 2. 获取成员用户列表（复用 Dashboard 返回的 user_stats）
  const { data: dashboardData } = useQuery({
    queryKey: ['dashboard', currentMonth],
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: addDrawerOpen && !!currentUser,
  });

  const users = dashboardData?.user_stats || [];

  // 3. 表单初始化与 Zod Resolver 挂载
  const getTodayString = () => {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  const {
    register,
    handleSubmit,
    control,
    watch,
    setValue,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      type: 'expense',
      amount: '',
      title: '',
      category_id: '',
      tag_names: '',
      payer_user_id: currentUser?.id || '',
      split_method: 'equal',
      occurred_at: getTodayString(),
      note: '',
      visibility: 'partner_readable',
    },
  });

  // 监听关键表单字段以实现动态联动
  const watchType = watch('type');
  const watchPayer = watch('payer_user_id');
  const watchSplitMethod = watch('split_method');

  // 当登录用户发生变化或抽屉打开时，更新默认付款人
  useEffect(() => {
    if (currentUser?.id) {
      setValue('payer_user_id', currentUser.id);
    }
  }, [currentUser, setValue, addDrawerOpen]);

  // 4. 定义创建账单的 Mutation
  const createTxMutation = useMutation({
    mutationFn: async (values: FormValues) => {
      const cents = yuanToCents(values.amount);
      const tags = values.tag_names
        ? values.tag_names.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean)
        : [];

      // 转换为 UTC 的 ISO8601 标准字符串，以防后端解析失败
      const rfc3339Date = new Date(values.occurred_at).toISOString();

      if (values.type === 'shared_expense') {
        return transactionsApi.createSharedExpense({
          title: values.title || '',
          amount_cents: cents,
          currency: 'CNY',
          occurred_at: rfc3339Date,
          payer_user_id: values.payer_user_id,
          category_id: values.category_id || undefined,
          split_method: values.split_method,
          tag_names: tags,
          note: values.note || '',
        });
      } else {
        return transactionsApi.createTransaction({
          type: values.type,
          title: values.title || '',
          amount_cents: cents,
          currency: 'CNY',
          occurred_at: rfc3339Date,
          payer_user_id: values.payer_user_id,
          category_id: values.category_id || undefined,
          visibility: values.visibility,
          tag_names: tags,
          note: values.note || '',
        });
      }
    },
    onSuccess: () => {
      // 自动失效相关缓存以触发现代大屏数据更新
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      setAddDrawerOpen(false);
      reset({
        type: 'expense',
        amount: '',
        title: '',
        category_id: '',
        tag_names: '',
        payer_user_id: currentUser?.id || '',
        split_method: 'equal',
        occurred_at: getTodayString(),
        note: '',
        visibility: 'partner_readable',
      });
    },
  });

  const onSubmit = (values: FormValues) => {
    createTxMutation.mutate(values);
  };

  const handleClose = () => {
    setAddDrawerOpen(false);
  };

  if (!addDrawerOpen) return null;

  return (
    <div className="drawer-overlay glass-blur show" onClick={handleClose}>
      <div className="drawer-container glass-card" onClick={(e) => e.stopPropagation()}>
        {/* 头部区 */}
        <div className="drawer-header">
          <div className="header-title">
            <Sparkles className="title-icon text-glow" />
            <h3>记一笔账单</h3>
          </div>
          <button className="btn-close-drawer" onClick={handleClose}>
            <X size={20} />
          </button>
        </div>

        {/* 表单体 */}
        <form onSubmit={handleSubmit(onSubmit)} className="drawer-body">
          {createTxMutation.isError && (
            <div className="error-banner">
              <p>
                {createTxMutation.error instanceof Error
                  ? createTxMutation.error.message
                  : '提交失败，请检查填写内容'}
              </p>
            </div>
          )}

          {/* 类型分段选择器 */}
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

          {/* 金额大输入框 */}
          <div className="form-group amount-group">
            <label className="form-label">交易金额 (元)</label>
            <div className="amount-input-wrapper">
              <span className="currency-symbol">¥</span>
              <input
                type="number"
                step="0.01"
                placeholder="0.00"
                className={`amount-input ${errors.amount ? 'input-error' : ''}`}
                {...register('amount')}
              />
            </div>
            {errors.amount && <span className="field-error">{errors.amount.message}</span>}
          </div>

          {/* 标题 */}
          <div className="form-group">
            <label className="form-label">账单标题</label>
            <input
              type="text"
              placeholder={
                watchType === 'shared_expense' ? '例如: 晚餐平摊、超市采购' : '例如: 购买水果、发工资'
              }
              className="form-input"
              {...register('title')}
            />
            {errors.title && <span className="field-error">{errors.title.message}</span>}
          </div>

          {/* 交易分类 */}
          <div className="form-group">
            <label className="form-label">所属分类</label>
            {isCategoriesLoading ? (
              <div className="select-loading">
                <Loader2 size={16} className="spinner" />
                <span>加载分类中...</span>
              </div>
            ) : (
              <select className="form-select" {...register('category_id')}>
                <option value="">-- 请选择分类 (选填) --</option>
                {categories?.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* 日期选择 */}
          <div className="form-group">
            <label className="form-label">发生日期</label>
            <input type="date" className="form-input" {...register('occurred_at')} />
            {errors.occurred_at && (
              <span className="field-error">{errors.occurred_at.message}</span>
            )}
          </div>

          {/* 标签列表 */}
          <div className="form-group">
            <label className="form-label">账单标签</label>
            <input
              type="text"
              placeholder="多个标签请用逗号或空格分隔"
              className="form-input"
              {...register('tag_names')}
            />
          </div>

          {/* 共同支出特定字段 */}
          {watchType === 'shared_expense' && (
            <>
              {/* 付款人 */}
              <div className="form-group">
                <label className="form-label">付款人</label>
                <select className="form-select" {...register('payer_user_id')}>
                  <option value="">-- 请选择付款人 --</option>
                  {users.map((u) => (
                    <option key={u.user_id} value={u.user_id}>
                      {u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
                    </option>
                  ))}
                </select>
                {errors.payer_user_id && (
                  <span className="field-error">{errors.payer_user_id.message}</span>
                )}
              </div>

              {/* 分摊方式 */}
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
                        均等平分 (Equal)
                      </button>
                      <button
                        type="button"
                        className={`segment-btn ${field.value === 'payer_only' ? 'active' : ''}`}
                        onClick={() => field.onChange('payer_only')}
                      >
                        付款人全额承担
                      </button>
                    </div>
                  )}
                />
              </div>

              {/* 参与人展示（不可变选项以提供防错保障） */}
              <div className="form-group">
                <label className="form-label">账单参与人</label>
                <div className="participants-box">
                  {users.map((u) => {
                    // 默认当 equal 时两人都是参与人；当 payer_only 时只有付款人是参与人
                    const isParticipating =
                      watchSplitMethod === 'equal' || u.user_id === watchPayer;
                    return (
                      <div
                        key={u.user_id}
                        className={`participant-item ${isParticipating ? 'checked' : 'disabled'}`}
                      >
                        <div className="checkbox-icon">
                          {isParticipating && <Check size={14} />}
                        </div>
                        <span className="participant-name">
                          {u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
                        </span>
                      </div>
                    );
                  })}
                </div>
                <p className="dimmed-desc">
                  {watchSplitMethod === 'equal'
                    ? '均等平分模式下，全员均自动勾选参与。'
                    : '单方承担模式下，仅付款人被视为唯一消费人。'}
                </p>
              </div>
            </>
          )}

          {/* 个人账单特定字段 - 可见性 */}
          {watchType !== 'shared_expense' && (
            <div className="form-group">
              <label className="form-label">账单可见性</label>
              <Controller
                name="visibility"
                control={control}
                render={({ field }) => (
                  <div className="segmented-control">
                    <button
                      type="button"
                      className={`segment-btn ${field.value === 'partner_readable' ? 'active' : ''}`}
                      onClick={() => field.onChange('partner_readable')}
                    >
                      对方可见 (只读)
                    </button>
                    <button
                      type="button"
                      className={`segment-btn ${field.value === 'private' ? 'active' : ''}`}
                      onClick={() => field.onChange('private')}
                    >
                      仅自己可见 (Private)
                    </button>
                  </div>
                )}
              />
            </div>
          )}

          {/* 备注 */}
          <div className="form-group">
            <label className="form-label">交易备注</label>
            <textarea
              placeholder="记录账目的详细备注信息..."
              className="form-input textarea"
              rows={3}
              {...register('note')}
            />
            {errors.note && <span className="field-error">{errors.note.message}</span>}
          </div>

          {/* 底部操作区 */}
          <div className="drawer-footer">
            <button type="button" className="btn-secondary" onClick={handleClose}>
              取消
            </button>
            <button
              type="submit"
              className="btn-primary btn-submit"
              disabled={isSubmitting || createTxMutation.isPending}
            >
              {(isSubmitting || createTxMutation.isPending) ? (
                <>
                  <Loader2 size={16} className="spinner" />
                  <span>保存中...</span>
                </>
              ) : (
                <span>确认记账</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
