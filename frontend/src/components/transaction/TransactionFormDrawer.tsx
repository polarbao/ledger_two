import { useEffect, useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Loader2, Sparkles, Check, Trash2 } from 'lucide-react';
import { useUIStore } from '../../stores/ui.store';
import { useAuthStore } from '../../stores/auth.store';
import { transactionsApi } from '../../api/transactions.api';
import { dashboardApi } from '../../api/dashboard.api';
import { yuanToCents } from '../../utils/money';
import type { TransactionTemplateResponse, CreateTemplatePayload } from '../../types/transaction';

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
  attachment_paths: z.array(z.string()).optional(),
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
  const { addDrawerOpen, setAddDrawerOpen, currentMonth, copySourceTransaction, setCopySourceTransaction } = useUIStore();

  const [showSuccessBanner, setShowSuccessBanner] = useState(false);
  const [submitAction, setSubmitAction] = useState<'close' | 'continue'>('close');
  const [isSaveTmplOpen, setIsSaveTmplOpen] = useState(false);
  const [tmplName, setTmplName] = useState('');
  const [isManageTmplOpen, setIsManageTmplOpen] = useState(false);


  const LAST_TYPE_KEY = 'ledger_two_last_type';
  const LAST_CATEGORY_KEY = 'ledger_two_last_category_id';
  const RECENT_CATEGORIES_KEY = 'ledger_two_recent_categories';
  const LAST_TAGS_KEY = 'ledger_two_last_tag_names';
  const RECENT_TAGS_KEY = 'ledger_two_recent_tags';
  const LAST_PAYER_KEY = 'ledger_two_last_payer_id';
  const LAST_VISIBILITY_KEY = 'ledger_two_last_visibility';

  // 读取本地缓存的分类和标签列表以供快捷气泡使用
  const recentCategories = JSON.parse(localStorage.getItem(RECENT_CATEGORIES_KEY) || '[]') as string[];
  const recentTags = JSON.parse(localStorage.getItem(RECENT_TAGS_KEY) || '[]') as string[];

  // 1. 获取全量分类列表
  const { data: categories, isLoading: isCategoriesLoading } = useQuery({
    queryKey: ['categories'],
    queryFn: () => transactionsApi.getCategories(),
    enabled: addDrawerOpen,
  });

  const catMap = categories?.reduce((acc, cat) => {
    acc[cat.id] = cat.name;
    return acc;
  }, {} as Record<string, string>) || {};

  // 2. 获取成员用户列表（复用 Dashboard 返回的 user_stats）
  const { data: dashboardData } = useQuery({
    queryKey: ['dashboard', currentMonth],
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: addDrawerOpen && !!currentUser,
  });

  const users = dashboardData?.user_stats || [];

  // 2.5 获取所有账单模板列表
  const { data: templates } = useQuery({
    queryKey: ['transaction-templates'],
    queryFn: () => transactionsApi.listTemplates(),
    enabled: addDrawerOpen,
  });

  // 创建模板 Mutation
  const createTemplateMutation = useMutation({
    mutationFn: (payload: CreateTemplatePayload) => transactionsApi.createTemplate(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transaction-templates'] });
      setIsSaveTmplOpen(false);
      setTmplName('');
    },
  });

  // 删除模板 Mutation
  const deleteTemplateMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.deleteTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['transaction-templates'] });
    },
  });

  const applyTemplate = (tmpl: TransactionTemplateResponse) => {
    const amountYuan = tmpl.amount_cents != null ? (tmpl.amount_cents / 100).toFixed(2) : '';
    const tagsStr = tmpl.tag_names ? tmpl.tag_names.join(', ') : '';
    setValue('type', tmpl.type);
    setValue('amount', amountYuan);
    setValue('title', tmpl.title || '');
    setValue('category_id', tmpl.category_id || '');
    setValue('tag_names', tagsStr);
    setValue('payer_user_id', tmpl.payer_user_id || currentUser?.id || '');
    setValue('split_method', tmpl.split_method === 'payer_only' ? 'payer_only' : 'equal');
    setValue('note', tmpl.note || '');
  };

  const handleSaveAsTemplate = () => {
    if (!tmplName.trim()) {
      return;
    }
    const formVals = watch();
    const cents = formVals.amount ? yuanToCents(formVals.amount) : undefined;
    const tags = formVals.tag_names
      ? formVals.tag_names.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean)
      : [];

    createTemplateMutation.mutate({
      name: tmplName.trim(),
      type: formVals.type,
      title: formVals.title || undefined,
      amount_cents: cents,
      category_id: formVals.category_id || undefined,
      payer_user_id: formVals.payer_user_id || undefined,
      split_method: formVals.split_method || undefined,
      tag_names: tags,
      note: formVals.note || undefined,
    });
  };

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
      attachment_paths: [],
    },
  });

  // 监听关键表单字段以实现动态联动
  const watchType = watch('type');
  const watchPayer = watch('payer_user_id');
  const watchSplitMethod = watch('split_method');
  const watchAttachmentPaths = watch('attachment_paths');

  const [uploadingCount, setUploadingCount] = useState(0);
  const [uploadError, setUploadError] = useState<string | null>(null);

  // 打开抽屉时注入最近记账默认值 (仅在非复制一笔时)
  useEffect(() => {
    if (addDrawerOpen && !copySourceTransaction) {
      const localType = (localStorage.getItem(LAST_TYPE_KEY) as FormValues['type']) || 'expense';
      const localCategory = localStorage.getItem(LAST_CATEGORY_KEY) || '';
      const localTags = localStorage.getItem(LAST_TAGS_KEY) || '';
      const localPayer = localStorage.getItem(LAST_PAYER_KEY) || currentUser?.id || '';
      const localVisibility = (localStorage.getItem(LAST_VISIBILITY_KEY) as FormValues['visibility']) || 'partner_readable';

      reset({
        type: localType,
        amount: '',
        title: '',
        category_id: localCategory,
        tag_names: localTags,
        payer_user_id: localPayer,
        split_method: 'equal',
        occurred_at: getTodayString(),
        note: '',
        visibility: localVisibility,
        attachment_paths: [],
      });
    }
  }, [addDrawerOpen, copySourceTransaction, currentUser, reset]);

  // 处理“复制一笔”回填逻辑
  useEffect(() => {
    if (addDrawerOpen && copySourceTransaction) {
      const amountYuan = (copySourceTransaction.amount_cents / 100).toFixed(2);
      const tagsStr = copySourceTransaction.tags ? copySourceTransaction.tags.join(', ') : '';
      
      reset({
        type: copySourceTransaction.type === 'settlement' ? 'expense' : copySourceTransaction.type,
        amount: amountYuan,
        title: copySourceTransaction.title || '',
        category_id: copySourceTransaction.category_id || '',
        tag_names: tagsStr,
        payer_user_id: copySourceTransaction.payer_user_id || currentUser?.id || '',
        split_method: copySourceTransaction.split_method || 'equal',
        occurred_at: getTodayString(),
        note: copySourceTransaction.note || '',
        visibility: copySourceTransaction.visibility === 'shared' ? 'partner_readable' : copySourceTransaction.visibility,
        attachment_paths: copySourceTransaction.attachment_paths || [],
      });
    }
  }, [addDrawerOpen, copySourceTransaction, reset, currentUser]);

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
          attachment_paths: values.attachment_paths || [],
        });
      }
    },
    onSuccess: (_, variables) => {
      // 自动失效相关缓存以触发现代大屏数据更新
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      
      // 更新 LocalStorage 快捷缓存默认值
      localStorage.setItem(LAST_TYPE_KEY, variables.type);
      localStorage.setItem(LAST_PAYER_KEY, variables.payer_user_id);
      localStorage.setItem(LAST_VISIBILITY_KEY, variables.visibility);

      if (variables.category_id) {
        localStorage.setItem(LAST_CATEGORY_KEY, variables.category_id);
        const recentCats = JSON.parse(localStorage.getItem(RECENT_CATEGORIES_KEY) || '[]') as string[];
        const updatedCats = [variables.category_id, ...recentCats.filter((id) => id !== variables.category_id)].slice(0, 3);
        localStorage.setItem(RECENT_CATEGORIES_KEY, JSON.stringify(updatedCats));
      } else {
        localStorage.removeItem(LAST_CATEGORY_KEY);
      }

      if (variables.tag_names) {
        localStorage.setItem(LAST_TAGS_KEY, variables.tag_names);
        const currentTags = variables.tag_names.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean);
        if (currentTags.length > 0) {
          const recentTags = JSON.parse(localStorage.getItem(RECENT_TAGS_KEY) || '[]') as string[];
          let updatedTags = [...recentTags];
          currentTags.forEach((tag) => {
            updatedTags = [tag, ...updatedTags.filter((t) => t !== tag)];
          });
          localStorage.setItem(RECENT_TAGS_KEY, JSON.stringify(updatedTags.slice(0, 3)));
        }
      } else {
        localStorage.removeItem(LAST_TAGS_KEY);
      }

      if (submitAction === 'continue') {
        setShowSuccessBanner(true);
        setTimeout(() => setShowSuccessBanner(false), 3000);

        reset({
          type: variables.type,
          amount: '',
          title: '',
          category_id: variables.category_id || '',
          tag_names: variables.tag_names || '',
          payer_user_id: variables.payer_user_id,
          split_method: variables.split_method || 'equal',
          occurred_at: variables.occurred_at ? variables.occurred_at.substring(0, 10) : getTodayString(),
          note: '',
          visibility: variables.visibility || 'partner_readable',
          attachment_paths: [],
        });
      } else {
        setAddDrawerOpen(false);
        setCopySourceTransaction(null);
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
          attachment_paths: [],
        });
      }
    },
  });

  const onSubmit = (values: FormValues) => {
    createTxMutation.mutate(values);
  };

  const handleClose = () => {
    setAddDrawerOpen(false);
    setCopySourceTransaction(null);
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
          {showSuccessBanner && (
            <div className="success-banner animate-fade-in" style={{
              background: 'rgba(53, 196, 137, 0.12)',
              border: '1px solid rgba(53, 196, 137, 0.25)',
              color: 'var(--accent-green)',
              borderRadius: '12px',
              padding: '12px 16px',
              marginBottom: '16px',
              fontSize: '14px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              backdropFilter: 'blur(8px)'
            }}>
              <Check size={16} />
              <span>账单已成功保存！您可以继续录入下一笔。</span>
            </div>
          )}
          {createTxMutation.isError && (
            <div className="error-banner">
              <p>
                {createTxMutation.error instanceof Error
                  ? createTxMutation.error.message
                  : '提交失败，请检查填写内容'}
              </p>
            </div>
          )}

          {/* 模板快速填充 */}
          <div className="form-group template-select-group">
            <label className="form-label" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span>使用模板快速填入</span>
              {templates && templates.length > 0 && (
                <button
                  type="button"
                  onClick={() => setIsManageTmplOpen(true)}
                  style={{
                    fontSize: '12px',
                    color: 'var(--accent-purple)',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    padding: 0
                  }}
                >
                  管理模板
                </button>
              )}
            </label>
            <div style={{ display: 'flex', gap: '8px', width: '100%', flexDirection: 'column' }}>
              <select
                className="form-select"
                onChange={(e) => {
                  const selectedId = e.target.value;
                  if (selectedId) {
                    const found = templates?.find((t) => t.id === selectedId);
                    if (found) {
                      applyTemplate(found);
                    }
                    e.target.value = ''; // 重置选择框
                  }
                }}
                defaultValue=""
              >
                <option value="">-- 选择模板一键回填表单 --</option>
                {templates?.map((tmpl) => (
                  <option key={tmpl.id} value={tmpl.id}>
                    {tmpl.name} ({tmpl.type === 'expense' ? '支出' : tmpl.type === 'income' ? '收入' : '共同支出'})
                  </option>
                ))}
              </select>

              {/* 快捷模板气泡滚动 */}
              {templates && templates.length > 0 && (
                <div className="template-badge-scroll">
                  {templates.slice(0, 5).map((tmpl) => (
                    <button
                      key={tmpl.id}
                      type="button"
                      className="template-badge-btn"
                      onClick={() => applyTemplate(tmpl)}
                    >
                      ⚡ {tmpl.name}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

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
            <div className="amount-input-wrapper" style={{ position: 'relative' }}>
              <span className="currency-symbol">¥</span>
              <input
                type="number"
                step="0.01"
                inputMode="decimal"
                pattern="[0-9]*\.?[0-9]*"
                placeholder="0.00"
                className={`amount-input ${errors.amount ? 'input-error' : ''}`}
                style={{ paddingRight: '40px' }}
                {...register('amount')}
              />
              {watch('amount') && (
                <button
                  type="button"
                  onClick={() => setValue('amount', '')}
                  style={{
                    position: 'absolute',
                    right: '12px',
                    top: '50%',
                    transform: 'translateY(-50%)',
                    background: 'rgba(255, 255, 255, 0.08)',
                    border: 'none',
                    borderRadius: '50%',
                    width: '22px',
                    height: '22px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer',
                    color: 'var(--text-muted)',
                    transition: 'background 0.2s',
                    padding: 0
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 255, 255, 0.15)'}
                  onMouseLeave={(e) => e.currentTarget.style.background = 'rgba(255, 255, 255, 0.08)'}
                >
                  <X size={12} />
                </button>
              )}
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
              <>
                <select className="form-select" {...register('category_id')}>
                  <option value="">-- 请选择分类 (选填) --</option>
                  {categories?.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </select>
                {/* 快捷分类气泡 */}
                {recentCategories.length > 0 && (
                  <div className="recent-helpers" style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', marginTop: '6px' }}>
                    <span className="dimmed-desc" style={{ fontSize: '12px', alignSelf: 'center', color: 'var(--text-muted)' }}>最近使用:</span>
                    {recentCategories.map((catId) => {
                      const catName = catMap[catId];
                      if (!catName) return null;
                      return (
                        <button
                          key={catId}
                          type="button"
                          className="badge-shared"
                          style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '11px', border: '1px solid rgba(255,255,255,0.08)', background: 'rgba(255,255,255,0.02)', cursor: 'pointer' }}
                          onClick={() => setValue('category_id', catId)}
                        >
                          {catName}
                        </button>
                      );
                    })}
                  </div>
                )}
              </>
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
            {/* 快捷标签气泡 */}
            {recentTags.length > 0 && (
              <div className="recent-helpers" style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', marginTop: '6px' }}>
                <span className="dimmed-desc" style={{ fontSize: '12px', alignSelf: 'center', color: 'var(--text-muted)' }}>最近标签:</span>
                {recentTags.map((tag) => (
                  <button
                    key={tag}
                    type="button"
                    className="badge-shared"
                    style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '11px', border: '1px solid rgba(255,255,255,0.08)', background: 'rgba(255,255,255,0.02)', cursor: 'pointer' }}
                    onClick={() => {
                      const currentVal = watch('tag_names') || '';
                      const trimmed = currentVal.trim();
                      if (!trimmed) {
                        setValue('tag_names', tag);
                      } else {
                        const tagsList = trimmed.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean);
                        if (!tagsList.includes(tag)) {
                          setValue('tag_names', `${trimmed}, ${tag}`);
                        }
                      }
                    }}
                  >
                    #{tag}
                  </button>
                ))}
              </div>
            )}
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

          {/* 图片附件 (仅对普通收支展示) */}
          {watchType !== 'shared_expense' && (
            <div className="form-group">
              <label className="form-label" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>图片附件与小票 ({watchAttachmentPaths?.length || 0}/5)</span>
                {uploadError && <span className="field-error" style={{ margin: 0 }}>{uploadError}</span>}
              </label>
              <Controller
                name="attachment_paths"
                control={control}
                render={({ field }) => {
                  const paths = field.value || [];
                  return (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                        {/* 已上传图片缩略图 */}
                        {paths.map((p, idx) => (
                          <div
                            key={p}
                            style={{
                              position: 'relative',
                              width: '72px',
                              height: '72px',
                              borderRadius: '8px',
                              overflow: 'hidden',
                              border: '1px solid rgba(255, 255, 255, 0.12)',
                              background: 'rgba(255, 255, 255, 0.05)',
                            }}
                          >
                            <img
                              src={p}
                              alt={`attachment-${idx}`}
                              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                            />
                            <button
                              type="button"
                              onClick={() => {
                                field.onChange(paths.filter((item) => item !== p));
                              }}
                              style={{
                                position: 'absolute',
                                top: '2px',
                                right: '2px',
                                background: 'rgba(0, 0, 0, 0.6)',
                                border: 'none',
                                borderRadius: '50%',
                                width: '18px',
                                height: '18px',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                color: '#fff',
                                cursor: 'pointer',
                                padding: 0,
                              }}
                            >
                              <X size={12} />
                            </button>
                          </div>
                        ))}

                        {/* 上传中的骨架屏 */}
                        {Array.from({ length: uploadingCount }).map((_, i) => (
                          <div
                            key={i}
                            style={{
                              width: '72px',
                              height: '72px',
                              borderRadius: '8px',
                              border: '1px dashed rgba(255, 255, 255, 0.2)',
                              background: 'rgba(255, 255, 255, 0.02)',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                            }}
                            className="animate-pulse"
                          >
                            <Loader2 size={16} className="spinner" style={{ color: 'var(--accent-purple)' }} />
                          </div>
                        ))}

                        {/* 上传按钮 */}
                        {paths.length + uploadingCount < 5 && (
                          <label
                            style={{
                              width: '72px',
                              height: '72px',
                              borderRadius: '8px',
                              border: '1px dashed rgba(255, 255, 255, 0.2)',
                              background: 'rgba(255, 255, 255, 0.05)',
                              display: 'flex',
                              flexDirection: 'column',
                              alignItems: 'center',
                              justifyContent: 'center',
                              cursor: 'pointer',
                              transition: 'all 0.2s',
                            }}
                            onMouseEnter={(e) => {
                              e.currentTarget.style.borderColor = 'var(--accent-purple)';
                              e.currentTarget.style.background = 'rgba(255, 255, 255, 0.08)';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.borderColor = 'rgba(255, 255, 255, 0.2)';
                              e.currentTarget.style.background = 'rgba(255, 255, 255, 0.05)';
                            }}
                          >
                            <svg
                              width="20"
                              height="20"
                              viewBox="0 0 24 24"
                              fill="none"
                              stroke="currentColor"
                              strokeWidth="2"
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              style={{ color: 'var(--text-secondary)' }}
                            >
                              <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
                              <circle cx="9" cy="9" r="2" />
                              <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21" />
                            </svg>
                            <span style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>添加图片</span>
                            <input
                              type="file"
                              accept="image/jpeg,image/jpg,image/png,image/webp"
                              style={{ display: 'none' }}
                              onChange={async (e) => {
                                const files = e.target.files;
                                if (!files || files.length === 0) return;
                                const file = files[0];

                                if (file.size > 10 * 1024 * 1024) {
                                  setUploadError('文件大小不能超过 10MB');
                                  return;
                                }

                                setUploadError(null);
                                setUploadingCount((prev) => prev + 1);
                                try {
                                  const res = await transactionsApi.uploadAttachment(file);
                                  if (res.path) {
                                    field.onChange([...paths, res.path]);
                                  }
                                } catch (err) {
                                  console.error(err);
                                  setUploadError(err instanceof Error ? err.message : '上传文件失败，请重试');
                                } finally {
                                  setUploadingCount((prev) => prev - 1);
                                }
                                e.target.value = '';
                              }}
                            />
                          </label>
                        )}
                      </div>
                    </div>
                  );
                }}
              />
            </div>
          )}

          {/* 底部操作区 */}
          <div className="drawer-footer" style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', flexWrap: 'wrap' }}>
            <button
              type="button"
              className="btn-secondary"
              style={{ marginRight: 'auto', borderColor: 'rgba(255, 255, 255, 0.12)' }}
              onClick={() => setIsSaveTmplOpen(true)}
            >
              存为模板
            </button>
            <button type="button" className="btn-secondary" onClick={handleClose}>
              取消
            </button>
            <button
              type="submit"
              className="btn-secondary"
              style={{ borderColor: 'var(--accent-primary)', color: 'var(--accent-primary)' }}
              disabled={isSubmitting || createTxMutation.isPending}
              onClick={() => setSubmitAction('continue')}
            >
              保存并继续
            </button>
            <button
              type="submit"
              className="btn-primary btn-submit"
              style={{ width: 'auto', padding: '10px 24px' }}
              disabled={isSubmitting || createTxMutation.isPending}
              onClick={() => setSubmitAction('close')}
            >
              {(isSubmitting || createTxMutation.isPending) && submitAction === 'close' ? (
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

      {/* 另存为模板对话框 */}
      {isSaveTmplOpen && (
        <div className="modal-overlay" onClick={() => setIsSaveTmplOpen(false)}>
          <div className="modal-content glass-card" style={{ maxWidth: '380px' }} onClick={(e) => e.stopPropagation()}>
            <h4 style={{ margin: '0 0 16px 0', fontSize: '18px', display: 'flex', gap: '8px', alignItems: 'center' }}>
              <Sparkles size={18} className="text-glow" style={{ color: 'var(--accent-purple)' }} />
              另存为账单模板
            </h4>
            <p style={{ fontSize: '13px', color: 'var(--text-secondary)', margin: '0 0 16px 0', lineHeight: 1.5 }}>
              将当前填写的金额、分类、标签等参数保存为模板，方便下次一键填入。
            </p>
            <div className="form-group" style={{ marginBottom: '20px' }}>
              <label className="form-label">模板名称</label>
              <input
                type="text"
                placeholder="例如: 每周吃黄焖鸡、日常午餐"
                className="form-input"
                value={tmplName}
                onChange={(e) => setTmplName(e.target.value)}
                style={{ width: '100%' }}
                autoFocus
              />
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setIsSaveTmplOpen(false)}
              >
                取消
              </button>
              <button
                type="button"
                className="btn-primary"
                disabled={createTemplateMutation.isPending || !tmplName.trim()}
                onClick={handleSaveAsTemplate}
              >
                {createTemplateMutation.isPending ? '保存中...' : '确认保存'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 模板管理器对话框 */}
      {isManageTmplOpen && (
        <div className="modal-overlay" onClick={() => setIsManageTmplOpen(false)}>
          <div className="modal-content glass-card" style={{ maxWidth: '420px', display: 'flex', flexDirection: 'column', maxHeight: '80vh' }} onClick={(e) => e.stopPropagation()}>
            <h4 style={{ margin: '0 0 16px 0', fontSize: '18px', display: 'flex', gap: '8px', alignItems: 'center' }}>
              <span>管理账单模板</span>
            </h4>
            
            <div className="template-manager-list">
              {templates && templates.length > 0 ? (
                templates.map((tmpl) => (
                  <div key={tmpl.id} className="template-item">
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', overflow: 'hidden' }}>
                      <span style={{ fontSize: '14px', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{tmpl.name}</span>
                      <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                        类型: {tmpl.type === 'expense' ? '支出' : tmpl.type === 'income' ? '收入' : '共同支出'}
                        {tmpl.amount_cents != null && ` · ¥${(tmpl.amount_cents / 100).toFixed(2)}`}
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        if (confirm(`确认删除模板 "${tmpl.name}" 吗？此操作不可撤销且不影响已有交易。`)) {
                          deleteTemplateMutation.mutate(tmpl.id);
                        }
                      }}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: '#ef4444',
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '4px',
                        padding: '4px 8px',
                        transition: 'opacity 0.2s'
                      }}
                      onMouseEnter={(e) => e.currentTarget.style.opacity = '0.8'}
                      onMouseLeave={(e) => e.currentTarget.style.opacity = '1'}
                    >
                      <Trash2 size={14} />
                      <span>删除</span>
                    </button>
                  </div>
                ))
              ) : (
                <div style={{ textAlign: 'center', padding: '30px 0', color: 'var(--text-muted)' }}>
                  暂无模板，可在记账后点击“存为模板”保存。
                </div>
              )}
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setIsManageTmplOpen(false)}
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

