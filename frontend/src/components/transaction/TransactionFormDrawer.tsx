import { useCallback, useEffect, useRef, useState } from 'react';
import { useForm, Controller, type FieldErrors } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Check,
  ChevronDown,
  ImagePlus,
  Info,
  Loader2,
  Pencil,
  ReceiptText,
  Sparkles,
  Trash2,
  X,
} from 'lucide-react';
import { useUIStore } from '../../stores/ui.store';
import { useAuthStore } from '../../stores/auth.store';
import { transactionsApi } from '../../api/transactions.api';
import { dashboardApi } from '../../api/dashboard.api';
import { queryKeys } from '../../api/queryKeys';
import { yuanToCents } from '../../utils/money';
import type { TransactionTemplateResponse, CreateTemplatePayload } from '../../types/transaction';
import { useDraftStore } from '../../stores/draft.store';
import { useLedgerStore } from '../../stores/ledger.store';
import { useHasLedgerRole } from '../ledger/useLedgerPermission';
import Button from '../ui/Button';
import ConfirmDialog from '../ui/ConfirmDialog';
import SegmentedControl from '../ui/SegmentedControl';
import useModalSurface from '../ui/useModalSurface';
import SharedExpensePreview from './SharedExpensePreview';
import TransactionFormFooter from './TransactionFormFooter';
import type { TransactionFormMode } from './TransactionFormFooter';
import {
  buildContinueTransactionFormValues,
  buildSharedExpensePreview,
  shouldOpenAdvancedFields,
} from './transactionFormState';
import './TransactionFormDrawer.css';

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
  account_id: z.string().optional().nullable(),
  tag_names: z.string().optional(),
  payer_user_id: z.string().min(1, '请选择付款人'),
  split_method: z.enum(['equal', 'payer_only']),
  occurred_at: z.string().min(1, '请选择发生日期'),
  note: z.string().max(200, '备注最多支持 200 字').optional(),
  visibility: z.enum(['private', 'partner_readable']),
  attachment_paths: z.array(z.string()).optional(),
});

type FormValues = z.infer<typeof formSchema>;

type TemplateEditState = {
  id: string;
  name: string;
  type: 'expense' | 'income' | 'shared_expense';
  title: string;
  amount: string;
  category_id: string;
  account_id: string;
  payer_user_id: string;
  split_method: 'equal' | 'payer_only';
  tag_names: string;
  note: string;
  is_archived: boolean;
};

/**
 * @brief 记账滑出层组件 (TransactionFormDrawer)
 * @details 兼容普通支出/收入与共同账单创建，支持电脑右滑与手机底滑布局。
 * @return React.ReactElement 返回渲染的 React 节点
 */
export default function TransactionFormDrawer() {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);
  const activeLedgerId = useLedgerStore((state) => state.activeLedgerId);
  const canWriteLedger = useHasLedgerRole(['owner', 'editor']);
  const {
    addDrawerOpen,
    setAddDrawerOpen,
    currentMonth,
    copySourceTransaction,
    setCopySourceTransaction,
    openTemplateSaveOnDrawerOpen,
    setOpenTemplateSaveOnDrawerOpen,
    isOffline,
    editingDraftId,
    setEditingDraftId,
  } = useUIStore();
  const { addDraft, updateDraft, removeDraft, drafts } = useDraftStore();

  const [showSuccessBanner, setShowSuccessBanner] = useState(false);
  const [submitAction, setSubmitAction] = useState<'close' | 'continue'>('close');
  const [isSaveTmplOpen, setIsSaveTmplOpen] = useState(false);
  const [tmplName, setTmplName] = useState<string | null>(null);
  const [isManageTmplOpen, setIsManageTmplOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<TemplateEditState | null>(null);
  const [templateEditError, setTemplateEditError] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const amountInputRef = useRef<HTMLInputElement | null>(null);
  const drawerSurfaceRef = useRef<HTMLElement | null>(null);
  const drawerTriggerRef = useRef<HTMLElement | null>(null);
  const drawerWasOpenRef = useRef(false);

  const LAST_TYPE_KEY = 'ledger_two_last_type';
  const LAST_CATEGORY_KEY = 'ledger_two_last_category_id';
  const LAST_ACCOUNT_KEY = 'ledger_two_last_account_id';
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
    queryKey: queryKeys.categories(activeLedgerId),
    queryFn: () => transactionsApi.getCategories(),
    enabled: addDrawerOpen && canWriteLedger,
  });

  const catMap = categories?.reduce((acc, cat) => {
    acc[cat.id] = cat.name;
    return acc;
  }, {} as Record<string, string>) || {};

  const { data: accounts, isLoading: isAccountsLoading } = useQuery({
    queryKey: queryKeys.accounts(activeLedgerId),
    queryFn: () => transactionsApi.listAccounts(),
    enabled: addDrawerOpen && canWriteLedger,
  });

  const { data: transactionDefaults, isFetched: isDefaultsFetched } = useQuery({
    queryKey: queryKeys.transactionDefaults(activeLedgerId),
    queryFn: () => transactionsApi.getTransactionDefaults(),
    enabled: addDrawerOpen && canWriteLedger && !copySourceTransaction && !editingDraftId,
  });

  // 2. 获取成员用户列表（复用 Dashboard 返回的 user_stats）
  const { data: dashboardData } = useQuery({
    queryKey: queryKeys.dashboard.month(activeLedgerId, currentMonth),
    queryFn: () => dashboardApi.getDashboard(currentMonth),
    enabled: addDrawerOpen && canWriteLedger && !!currentUser,
  });

  const users = dashboardData?.user_stats || [];

  // 2.5 获取所有账单模板列表
  const { data: templates } = useQuery({
    queryKey: queryKeys.templates(activeLedgerId),
    queryFn: () => transactionsApi.listTemplates({ includeArchived: true }),
    enabled: addDrawerOpen && canWriteLedger,
  });
  const activeTemplates = templates?.filter((tmpl) => !tmpl.is_archived) || [];

  // 创建模板 Mutation
  const createTemplateMutation = useMutation({
    mutationFn: (payload: CreateTemplatePayload) => transactionsApi.createTemplate(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates(activeLedgerId) });
      setIsSaveTmplOpen(false);
      setOpenTemplateSaveOnDrawerOpen(false);
      setTmplName(null);
    },
  });

  // 归档模板 Mutation
  const archiveTemplateMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.archiveTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates(activeLedgerId) });
    },
  });

  // 恢复模板 Mutation
  const restoreTemplateMutation = useMutation({
    mutationFn: (id: string) => transactionsApi.restoreTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates(activeLedgerId) });
    },
  });

  const updateTemplateMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: CreateTemplatePayload }) =>
      transactionsApi.updateTemplate(id, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.templates(activeLedgerId) });
      setEditingTemplate(null);
      setTemplateEditError(null);
    },
    onError: (err) => {
      setTemplateEditError(err instanceof Error ? err.message : '更新模板失败，请稍后重试');
    },
  });

  const applyTemplate = (tmpl: TransactionTemplateResponse) => {
    const amountYuan = tmpl.amount_cents != null ? (tmpl.amount_cents / 100).toFixed(2) : '';
    const tagsStr = tmpl.tag_names ? tmpl.tag_names.join(', ') : '';
    const dirtyOptions = { shouldDirty: true } as const;
    setValue('type', tmpl.type, dirtyOptions);
    setValue('amount', amountYuan, dirtyOptions);
    setValue('title', tmpl.title || '', dirtyOptions);
    setValue('category_id', tmpl.category_id || '', dirtyOptions);
    setValue('account_id', tmpl.account_id || '', dirtyOptions);
    setValue('tag_names', tagsStr, dirtyOptions);
    setValue('payer_user_id', tmpl.payer_user_id || currentUser?.id || '', dirtyOptions);
    setValue('split_method', tmpl.split_method === 'payer_only' ? 'payer_only' : 'equal', dirtyOptions);
    setValue('note', tmpl.note || '', dirtyOptions);
    setAdvancedOpen(shouldOpenAdvancedFields({
      type: tmpl.type,
      amount: amountYuan,
      title: tmpl.title || '',
      category_id: tmpl.category_id || '',
      account_id: tmpl.account_id || '',
      tag_names: tagsStr,
      payer_user_id: tmpl.payer_user_id || currentUser?.id || '',
      split_method: tmpl.split_method === 'payer_only' ? 'payer_only' : 'equal',
      occurred_at: getTodayString(),
      note: tmpl.note || '',
      visibility: watch('visibility'),
      attachment_paths: watch('attachment_paths'),
    }));
  };

  const handleSaveAsTemplate = () => {
    const resolvedName = tmplName ?? (
      openTemplateSaveOnDrawerOpen
        ? copySourceTransaction?.title
          ? `${copySourceTransaction.title}模板`
          : '账单模板'
        : ''
    );
    if (!resolvedName.trim()) {
      return;
    }
    const formVals = watch();
    const cents = formVals.amount ? yuanToCents(formVals.amount) : undefined;
    const tags = formVals.tag_names
      ? formVals.tag_names.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean)
      : [];

    createTemplateMutation.mutate({
      name: resolvedName.trim(),
      type: formVals.type,
      title: formVals.title || undefined,
      amount_cents: cents,
      category_id: formVals.category_id || undefined,
      account_id: formVals.account_id || undefined,
      payer_user_id: formVals.payer_user_id || undefined,
      split_method: formVals.split_method || undefined,
      tag_names: tags,
      note: formVals.note || undefined,
    });
  };

  const openEditTemplate = (tmpl: TransactionTemplateResponse) => {
    setTemplateEditError(null);
    setEditingTemplate({
      id: tmpl.id,
      name: tmpl.name || '',
      type: tmpl.type,
      title: tmpl.title || '',
      amount: tmpl.amount_cents != null ? (tmpl.amount_cents / 100).toFixed(2) : '',
      category_id: tmpl.category_id || '',
      account_id: tmpl.account_id || '',
      payer_user_id: tmpl.payer_user_id || currentUser?.id || '',
      split_method: tmpl.split_method === 'payer_only' ? 'payer_only' : 'equal',
      tag_names: tmpl.tag_names ? tmpl.tag_names.join(', ') : '',
      note: tmpl.note || '',
      is_archived: tmpl.is_archived,
    });
  };

  const patchEditingTemplate = (patch: Partial<TemplateEditState>) => {
    setTemplateEditError(null);
    setEditingTemplate((prev) => {
      if (!prev) return prev;
      const next = { ...prev, ...patch };
      if (patch.type === 'shared_expense') {
        next.account_id = '';
      }
      return next;
    });
  };

  const buildTemplatePayload = (formVals: TemplateEditState): CreateTemplatePayload | null => {
    const name = formVals.name.trim();
    if (!name) {
      setTemplateEditError('模板名称不能为空');
      return null;
    }

    let cents: number | undefined;
    const amount = formVals.amount.trim();
    if (amount) {
      try {
        cents = yuanToCents(amount);
      } catch {
        setTemplateEditError('模板金额格式错误，最多支持两位小数');
        return null;
      }
    }

    const tags = formVals.tag_names
      ? formVals.tag_names.split(/[，, ；;]/).map((t) => t.trim()).filter(Boolean)
      : [];

    return {
      name,
      type: formVals.type,
      title: formVals.title.trim() || undefined,
      amount_cents: cents,
      category_id: formVals.category_id || undefined,
      account_id: formVals.type !== 'shared_expense' ? formVals.account_id || undefined : undefined,
      payer_user_id: formVals.payer_user_id || undefined,
      split_method: formVals.split_method || undefined,
      tag_names: tags,
      note: formVals.note.trim() || undefined,
    };
  };

  const handleUpdateTemplate = () => {
    if (!editingTemplate) return;
    const payload = buildTemplatePayload(editingTemplate);
    if (!payload) return;
    updateTemplateMutation.mutate({ id: editingTemplate.id, payload });
  };

  // 3. 表单初始化与 Zod Resolver 挂载

  function getTodayString() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  const {
    register,
    handleSubmit,
    control,
    watch,
    setValue,
    reset,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      type: 'expense',
      amount: '',
      title: '',
      category_id: '',
      account_id: '',
      tag_names: '',
      payer_user_id: currentUser?.id || '',
      split_method: 'equal',
      occurred_at: getTodayString(),
      note: '',
      visibility: 'partner_readable',
      attachment_paths: [],
    },
  });
  const amountField = register('amount');

  // 监听关键表单字段以实现动态联动
  const watchType = watch('type');
  const watchAmount = watch('amount');
  const watchPayer = watch('payer_user_id');
  const watchSplitMethod = watch('split_method');
  const watchAttachmentPaths = watch('attachment_paths');

  const [uploadingCount, setUploadingCount] = useState(0);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const closeGuardRef = useRef({ isDirty: false, uploadingCount: 0 });

  useEffect(() => {
    closeGuardRef.current = { isDirty, uploadingCount };
  }, [isDirty, uploadingCount]);

  const closeDrawer = useCallback(() => {
    setShowDiscardConfirm(false);
    setIsSaveTmplOpen(false);
    setIsManageTmplOpen(false);
    setEditingTemplate(null);
    setAddDrawerOpen(false);
    setCopySourceTransaction(null);
    setOpenTemplateSaveOnDrawerOpen(false);
    setEditingDraftId(null);
  }, [
    setAddDrawerOpen,
    setCopySourceTransaction,
    setEditingDraftId,
    setOpenTemplateSaveOnDrawerOpen,
  ]);

  const requestClose = useCallback(() => {
    if (closeGuardRef.current.isDirty || closeGuardRef.current.uploadingCount > 0) {
      setShowDiscardConfirm(true);
      return;
    }
    closeDrawer();
  }, [closeDrawer]);

  const nestedDialogOpen = showDiscardConfirm
    || isSaveTmplOpen
    || openTemplateSaveOnDrawerOpen
    || isManageTmplOpen
    || editingTemplate !== null;

  useEffect(() => {
    const wasOpen = drawerWasOpenRef.current;
    if (addDrawerOpen && !wasOpen && typeof document !== 'undefined') {
      drawerTriggerRef.current = document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    }
    if (!addDrawerOpen && wasOpen) {
      const frame = window.requestAnimationFrame(() => drawerTriggerRef.current?.focus());
      drawerWasOpenRef.current = addDrawerOpen;
      return () => window.cancelAnimationFrame(frame);
    }
    drawerWasOpenRef.current = addDrawerOpen;
    return undefined;
  }, [addDrawerOpen]);

  useModalSurface({
    open: addDrawerOpen && !nestedDialogOpen,
    onClose: requestClose,
    surfaceRef: drawerSurfaceRef,
    initialFocusRef: amountInputRef,
  });

  // 打开抽屉时注入最近记账默认值 (仅在非复制且非草稿编辑时)
  useEffect(() => {
    let advancedFrame: number | undefined;
    if (addDrawerOpen && !copySourceTransaction && !editingDraftId) {
      const localType = (localStorage.getItem(LAST_TYPE_KEY) as FormValues['type']) || 'expense';
      const localCategory = localStorage.getItem(LAST_CATEGORY_KEY) || '';
      const localAccount = localStorage.getItem(LAST_ACCOUNT_KEY) || '';
      const localTags = localStorage.getItem(LAST_TAGS_KEY) || '';
      const localPayer = localStorage.getItem(LAST_PAYER_KEY) || currentUser?.id || '';
      const localVisibility = (localStorage.getItem(LAST_VISIBILITY_KEY) as FormValues['visibility']) || 'partner_readable';
      const defaults = transactionDefaults;

      const nextValues: FormValues = {
        type: defaults?.type || localType,
        amount: '',
        title: '',
        category_id: defaults?.category_id || localCategory,
        account_id: defaults?.account_id || localAccount,
        tag_names: defaults?.tag_names?.length ? defaults.tag_names.join(', ') : localTags,
        payer_user_id: defaults?.payer_user_id || localPayer,
        split_method: defaults?.split_method || 'equal',
        occurred_at: getTodayString(),
        note: '',
        visibility: defaults?.visibility || localVisibility,
        attachment_paths: [],
      };
      reset(nextValues);
      advancedFrame = window.requestAnimationFrame(() => {
        setAdvancedOpen(shouldOpenAdvancedFields(nextValues));
      });
    }
    return () => {
      if (advancedFrame !== undefined) window.cancelAnimationFrame(advancedFrame);
    };
  }, [addDrawerOpen, copySourceTransaction, editingDraftId, currentUser, reset, transactionDefaults, isDefaultsFetched]);

  // 处理“草稿编辑”回填逻辑
  useEffect(() => {
    let advancedFrame: number | undefined;
    if (addDrawerOpen && editingDraftId) {
      const draft = drafts.find(d => d.id === editingDraftId);
      if (draft) {
        reset(draft.formValues);
        advancedFrame = window.requestAnimationFrame(() => {
          setAdvancedOpen(shouldOpenAdvancedFields(draft.formValues));
        });
      }
    }
    return () => {
      if (advancedFrame !== undefined) window.cancelAnimationFrame(advancedFrame);
    };
  }, [addDrawerOpen, editingDraftId, drafts, reset]);

  // 处理“复制一笔”回填逻辑
  useEffect(() => {
    let advancedFrame: number | undefined;
    if (addDrawerOpen && copySourceTransaction) {
      const amountYuan = (copySourceTransaction.amount_cents / 100).toFixed(2);
      const tagsStr = copySourceTransaction.tags ? copySourceTransaction.tags.join(', ') : '';

      const nextValues: FormValues = {
        type: copySourceTransaction.type === 'settlement' ? 'expense' : copySourceTransaction.type,
        amount: amountYuan,
        title: copySourceTransaction.title || '',
        category_id: copySourceTransaction.category_id || '',
        account_id: copySourceTransaction.account_id || '',
        tag_names: tagsStr,
        payer_user_id: copySourceTransaction.payer_user_id || currentUser?.id || '',
        split_method: copySourceTransaction.split_method || 'equal',
        occurred_at: getTodayString(),
        note: copySourceTransaction.note || '',
        visibility: copySourceTransaction.visibility === 'shared' ? 'partner_readable' : copySourceTransaction.visibility,
        attachment_paths: copySourceTransaction.attachment_paths || [],
      };
      reset(nextValues);
      advancedFrame = window.requestAnimationFrame(() => {
        setAdvancedOpen(shouldOpenAdvancedFields(nextValues));
      });
    }
    return () => {
      if (advancedFrame !== undefined) window.cancelAnimationFrame(advancedFrame);
    };
  }, [addDrawerOpen, copySourceTransaction, reset, currentUser]);

  // 4. 定义创建账单的 Mutation
  const createTxMutation = useMutation({
    mutationFn: async ({ values }: { values: FormValues; action: 'close' | 'continue' }) => {
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
          account_id: values.account_id || undefined,
          visibility: values.visibility,
          tag_names: tags,
          note: values.note || '',
          attachment_paths: values.attachment_paths || [],
        });
      }
    },
    onSuccess: (_, { values: variables, action }) => {
      if (editingDraftId) {
        removeDraft(editingDraftId);
        setEditingDraftId(null);
      }

      // 自动失效相关缓存以触发现代大屏数据更新
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.root(activeLedgerId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.root(activeLedgerId) });
      
      // 更新 LocalStorage 快捷缓存默认值
      localStorage.setItem(LAST_TYPE_KEY, variables.type);
      localStorage.setItem(LAST_PAYER_KEY, variables.payer_user_id);
      localStorage.setItem(LAST_VISIBILITY_KEY, variables.visibility);
      if (variables.type !== 'shared_expense' && variables.account_id) {
        localStorage.setItem(LAST_ACCOUNT_KEY, variables.account_id);
      } else {
        localStorage.removeItem(LAST_ACCOUNT_KEY);
      }

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

      if (action === 'continue') {
        setShowSuccessBanner(true);
        setTimeout(() => setShowSuccessBanner(false), 3000);

        const nextValues = buildContinueTransactionFormValues(variables, getTodayString());
        reset(nextValues);
        setAdvancedOpen(shouldOpenAdvancedFields(nextValues));
        window.requestAnimationFrame(() => amountInputRef.current?.focus({ preventScroll: true }));
      } else {
        closeDrawer();
        reset({
          type: 'expense',
          amount: '',
          title: '',
          category_id: '',
          account_id: '',
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

  const onSubmit = (values: FormValues, action: 'close' | 'continue') => {
    if (isOffline) {
      if (editingDraftId) {
        updateDraft(editingDraftId, {
          id: editingDraftId,
          formValues: values,
          createdAt: new Date().toISOString(),
        });
      } else {
        addDraft({
          id: crypto.randomUUID(),
          formValues: values,
          createdAt: new Date().toISOString(),
        });
      }
      closeDrawer();
      return;
    }
    createTxMutation.mutate({ values, action });
  };

  const handleInvalidSubmit = (invalidFields: FieldErrors<FormValues>) => {
    if (invalidFields.title || invalidFields.note) setAdvancedOpen(true);
  };

  const setCurrentSubmitAction = (action: 'close' | 'continue') => {
    setSubmitAction(action);
  };

  const copyTemplateDefaultName = copySourceTransaction?.title
    ? `${copySourceTransaction.title}模板`
    : '账单模板';
  const resolvedTemplateName = tmplName ?? (openTemplateSaveOnDrawerOpen ? copyTemplateDefaultName : '');
  const templateSaveOpen = isSaveTmplOpen || openTemplateSaveOnDrawerOpen;
  const closeTemplateSave = () => {
    setIsSaveTmplOpen(false);
    setOpenTemplateSaveOnDrawerOpen(false);
    setTmplName(null);
  };

  const sharedPreview = buildSharedExpensePreview(
    watchAmount,
    users,
    watchPayer,
    watchSplitMethod,
  );
  const formMode: TransactionFormMode = isOffline
    ? 'offline'
    : editingDraftId
      ? 'draft'
      : copySourceTransaction
        ? 'copy'
        : 'default';
  const isFormPending = isSubmitting || createTxMutation.isPending || uploadingCount > 0;

  if (!addDrawerOpen) return null;

  if (!canWriteLedger) {
    return (
      <div
        className="lt-entry-overlay"
        onMouseDown={(event) => event.target === event.currentTarget && requestClose()}
      >
        <section
          ref={drawerSurfaceRef}
          className="lt-entry-drawer lt-entry-drawer--permission"
          role="dialog"
          aria-modal="true"
          aria-labelledby="lt-entry-permission-title"
          tabIndex={-1}
        >
          <header className="lt-entry-header">
            <div className="lt-entry-header__title">
              <ReceiptText size={20} aria-hidden="true" />
              <h2 id="lt-entry-permission-title">无记账权限</h2>
            </div>
            <Button iconOnly variant="ghost" aria-label="关闭记账抽屉" onClick={requestClose}>
              <X size={20} />
            </Button>
          </header>
          <div className="lt-entry-body">
            <div className="lt-entry-banner lt-entry-banner--danger">
              当前是观察者权限，无法在此账本新增、复制或提交账单。
            </div>
          </div>
        </section>
      </div>
    );
  }

  return (
    <>
      <div
        className="lt-entry-overlay"
        onMouseDown={(event) => event.target === event.currentTarget && requestClose()}
      >
        <section
          ref={drawerSurfaceRef}
          className="lt-entry-drawer"
          role="dialog"
          aria-modal="true"
          aria-labelledby="lt-entry-title"
          tabIndex={-1}
        >
          <header className="lt-entry-header">
            <div className="lt-entry-header__title">
              <ReceiptText size={20} aria-hidden="true" />
              <div>
                <span className="lt-entry-section__eyebrow">
                  {isOffline ? '离线草稿' : copySourceTransaction ? '复制账单' : '新增账单'}
                </span>
                <h2 id="lt-entry-title">
                  {copySourceTransaction ? '保存一笔新账单' : '记一笔账单'}
                </h2>
              </div>
            </div>
            <Button iconOnly variant="ghost" aria-label="关闭记账抽屉" onClick={requestClose}>
              <X size={20} />
            </Button>
          </header>

        <form
          onSubmit={handleSubmit(
            (values) => {
              setCurrentSubmitAction('close');
              onSubmit(values, 'close');
            },
            handleInvalidSubmit,
          )}
          className="lt-entry-form"
        >
          <div className="lt-entry-body">
            {isOffline && (
              <div className="lt-entry-banner lt-entry-banner--warning">
                当前处于离线状态，本次内容会保存为离线草稿。
              </div>
            )}
            {showSuccessBanner && !isOffline && (
              <div className="lt-entry-banner lt-entry-banner--success" role="status">
                <Check size={16} aria-hidden="true" />
                账单已保存，可以继续录入下一笔。
              </div>
            )}
            {copySourceTransaction && (
              <div className="lt-entry-banner lt-entry-banner--info">
                <strong>
                  复制来源：{copySourceTransaction.title || '未命名账单'} · ¥{(copySourceTransaction.amount_cents / 100).toFixed(2)}
                </strong>
                <span>保存后生成新账单，原账单保持不变，日期已重置为今天。</span>
              </div>
            )}
            {createTxMutation.isError && (
              <div className="lt-entry-banner lt-entry-banner--danger" role="alert">
                {createTxMutation.error instanceof Error
                  ? createTxMutation.error.message
                  : '提交失败，请检查填写内容'}
              </div>
            )}

            <section className="lt-entry-amount">
              <label htmlFor="lt-entry-amount-input">金额</label>
              <div className="lt-entry-amount__control">
                <span aria-hidden="true">¥</span>
                <input
                  id="lt-entry-amount-input"
                  type="number"
                  step="0.01"
                  min="0"
                  inputMode="decimal"
                  enterKeyHint="done"
                  autoComplete="off"
                  placeholder="0.00"
                  aria-invalid={Boolean(errors.amount)}
                  {...amountField}
                  ref={(element) => {
                    amountField.ref(element);
                    amountInputRef.current = element;
                  }}
                />
                {watchAmount ? (
                  <Button
                    iconOnly
                    variant="ghost"
                    aria-label="清空金额"
                    onClick={() => setValue('amount', '', { shouldDirty: true })}
                  >
                    <X size={16} />
                  </Button>
                ) : null}
              </div>
              {errors.amount ? <span className="lt-entry-field__error">{errors.amount.message}</span> : null}
            </section>

            <div className="lt-entry-type">
              <span className="lt-entry-field__label">账单类型</span>
              <Controller
                name="type"
                control={control}
                render={({ field }) => (
                  <SegmentedControl
                    ariaLabel="账单类型"
                    value={field.value}
                    options={[
                      { value: 'expense', label: '支出' },
                      { value: 'income', label: '收入' },
                      { value: 'shared_expense', label: '共同' },
                    ]}
                    onChange={field.onChange}
                    fullWidth
                  />
                )}
              />
            </div>

            <div className="lt-entry-core-grid">
              <div className="lt-entry-field">
                <label htmlFor="lt-entry-category">分类</label>
                {isCategoriesLoading ? (
                  <div className="lt-entry-field__loading">
                    <Loader2 size={16} className="spinner" /> 加载中
                  </div>
                ) : (
                  <select id="lt-entry-category" {...register('category_id')}>
                    <option value="">未分类</option>
                    {categories?.map((category) => (
                      <option key={category.id} value={category.id}>{category.name}</option>
                    ))}
                  </select>
                )}
                {recentCategories.length > 0 ? (
                  <div className="lt-entry-chips" aria-label="最近使用的分类">
                    {recentCategories.map((categoryId) => {
                      const categoryName = catMap[categoryId];
                      return categoryName ? (
                        <button
                          key={categoryId}
                          type="button"
                          onClick={() => setValue('category_id', categoryId, { shouldDirty: true })}
                        >
                          {categoryName}
                        </button>
                      ) : null;
                    })}
                  </div>
                ) : null}
              </div>

              {watchType !== 'shared_expense' ? (
                <div className="lt-entry-field">
                  <label htmlFor="lt-entry-account">账户</label>
                  {isAccountsLoading ? (
                    <div className="lt-entry-field__loading">
                      <Loader2 size={16} className="spinner" /> 加载中
                    </div>
                  ) : (
                    <select id="lt-entry-account" {...register('account_id')}>
                      <option value="">未指定</option>
                      {accounts?.map((account) => (
                        <option key={account.id} value={account.id}>{account.name}</option>
                      ))}
                    </select>
                  )}
                </div>
              ) : null}

              <div className="lt-entry-field">
                <label htmlFor="lt-entry-date">日期</label>
                <input id="lt-entry-date" type="date" {...register('occurred_at')} />
                {errors.occurred_at ? (
                  <span className="lt-entry-field__error">{errors.occurred_at.message}</span>
                ) : null}
              </div>
            </div>

            {watchType === 'shared_expense' ? (
              <section className="lt-entry-shared">
                <header className="lt-entry-section__header">
                  <div>
                    <span className="lt-entry-section__eyebrow">共同账单</span>
                    <h3>付款与分摊</h3>
                  </div>
                </header>
                <div className="lt-entry-shared__controls">
                  <div className="lt-entry-field">
                    <label htmlFor="lt-entry-payer">付款人</label>
                    <select id="lt-entry-payer" {...register('payer_user_id')}>
                      <option value="">请选择</option>
                      {users.map((user) => (
                        <option key={user.user_id} value={user.user_id}>
                          {user.display_name}{user.user_id === currentUser?.id ? '（我）' : ''}
                        </option>
                      ))}
                    </select>
                    {errors.payer_user_id ? (
                      <span className="lt-entry-field__error">{errors.payer_user_id.message}</span>
                    ) : null}
                  </div>
                  <div className="lt-entry-field">
                    <span className="lt-entry-field__label">分摊方式</span>
                    <Controller
                      name="split_method"
                      control={control}
                      render={({ field }) => (
                        <SegmentedControl
                          ariaLabel="分摊方式"
                          value={field.value}
                          options={[
                            { value: 'equal', label: '均等平分' },
                            { value: 'payer_only', label: '付款人承担' },
                          ]}
                          onChange={field.onChange}
                          fullWidth
                        />
                      )}
                    />
                  </div>
                </div>
                <SharedExpensePreview items={sharedPreview} currentUserId={currentUser?.id} />
              </section>
            ) : null}

            <details
              className="lt-entry-advanced"
              open={advancedOpen}
              onToggle={(event) => setAdvancedOpen(event.currentTarget.open)}
            >
              <summary>
                <span>
                  <span className="lt-entry-section__eyebrow">低频字段</span>
                  <strong>更多选项</strong>
                </span>
                <ChevronDown size={18} aria-hidden="true" />
              </summary>
              <div className="lt-entry-advanced__body">
                <div className="lt-entry-template">
                  <div className="lt-entry-field">
                    <div className="lt-entry-field__heading">
                      <label htmlFor="lt-entry-template">账单模板</label>
                      {templates?.length ? (
                        <button type="button" onClick={() => setIsManageTmplOpen(true)}>管理模板</button>
                      ) : null}
                    </div>
                    <select
                      id="lt-entry-template"
                      defaultValue=""
                      onChange={(event) => {
                        const template = templates?.find((item) => item.id === event.target.value);
                        if (template) applyTemplate(template);
                        event.target.value = '';
                      }}
                    >
                      <option value="">选择模板</option>
                      {activeTemplates.map((template) => (
                        <option key={template.id} value={template.id}>{template.name}</option>
                      ))}
                    </select>
                  </div>
                  {activeTemplates.length > 0 ? (
                    <div className="lt-entry-chips" aria-label="快捷模板">
                      {activeTemplates.slice(0, 5).map((template) => (
                        <button key={template.id} type="button" onClick={() => applyTemplate(template)}>
                          <Sparkles size={13} aria-hidden="true" /> {template.name}
                        </button>
                      ))}
                    </div>
                  ) : null}
                  <Button
                    className="lt-entry-template__save"
                    variant="ghost"
                    startIcon={<Sparkles size={16} />}
                    onClick={() => setIsSaveTmplOpen(true)}
                    disabled={isOffline}
                  >
                    存为模板
                  </Button>
                </div>

                <div className="lt-entry-field">
                  <label htmlFor="lt-entry-title-input">标题</label>
                  <input
                    id="lt-entry-title-input"
                    type="text"
                    placeholder={watchType === 'shared_expense' ? '例如：晚餐平摊' : '例如：周末采购'}
                    {...register('title')}
                  />
                  {errors.title ? <span className="lt-entry-field__error">{errors.title.message}</span> : null}
                </div>

                <div className="lt-entry-field">
                  <label htmlFor="lt-entry-tags">标签</label>
                  <input id="lt-entry-tags" type="text" placeholder="通勤，周末" {...register('tag_names')} />
                  {recentTags.length > 0 ? (
                    <div className="lt-entry-chips" aria-label="最近使用的标签">
                      {recentTags.map((tag) => (
                        <button
                          key={tag}
                          type="button"
                          onClick={() => {
                            const currentValue = watch('tag_names')?.trim() || '';
                            const tags = currentValue.split(/[，, ；;]/).map((item) => item.trim()).filter(Boolean);
                            if (!tags.includes(tag)) {
                              setValue('tag_names', currentValue ? `${currentValue}, ${tag}` : tag, { shouldDirty: true });
                            }
                          }}
                        >
                          #{tag}
                        </button>
                      ))}
                    </div>
                  ) : null}
                </div>

                {watchType !== 'shared_expense' ? (
                  <div className="lt-entry-field">
                    <span className="lt-entry-field__label">可见性</span>
                    <Controller
                      name="visibility"
                      control={control}
                      render={({ field }) => (
                        <SegmentedControl
                          ariaLabel="账单可见性"
                          value={field.value}
                          options={[
                            { value: 'partner_readable', label: '对方可见' },
                            { value: 'private', label: '仅自己' },
                          ]}
                          onChange={field.onChange}
                          fullWidth
                        />
                      )}
                    />
                  </div>
                ) : null}

                <div className="lt-entry-field">
                  <label htmlFor="lt-entry-note">备注</label>
                  <textarea id="lt-entry-note" rows={3} placeholder="补充账单信息" {...register('note')} />
                  {errors.note ? <span className="lt-entry-field__error">{errors.note.message}</span> : null}
                </div>

                {watchType !== 'shared_expense' ? (
                  <div className="lt-entry-field">
                    <div className="lt-entry-field__heading">
                      <span className="lt-entry-field__label">图片与小票</span>
                      <span>{watchAttachmentPaths?.length || 0}/5</span>
                    </div>
                    {uploadError ? <span className="lt-entry-field__error">{uploadError}</span> : null}
                    <Controller
                      name="attachment_paths"
                      control={control}
                      render={({ field }) => {
                        const paths = field.value || [];
                        return (
                          <div className="lt-entry-attachments">
                            {paths.map((path, index) => (
                              <div className="lt-entry-attachment" key={path}>
                                <img src={path} alt={`账单附件 ${index + 1}`} />
                                <button
                                  type="button"
                                  aria-label={`移除附件 ${index + 1}`}
                                  onClick={() => field.onChange(paths.filter((item) => item !== path))}
                                >
                                  <X size={14} />
                                </button>
                              </div>
                            ))}
                            {Array.from({ length: uploadingCount }).map((_, index) => (
                              <div className="lt-entry-attachment lt-entry-attachment--loading" key={`upload-${index}`}>
                                <Loader2 size={18} className="spinner" />
                              </div>
                            ))}
                            {paths.length + uploadingCount < 5 ? (
                              <label className="lt-entry-attachment lt-entry-attachment--add">
                                <ImagePlus size={20} aria-hidden="true" />
                                <span>添加图片</span>
                                <input
                                  type="file"
                                  accept="image/jpeg,image/jpg,image/png,image/webp"
                                  onChange={async (event) => {
                                    const file = event.target.files?.[0];
                                    if (!file) return;
                                    if (file.size > 10 * 1024 * 1024) {
                                      setUploadError('文件大小不能超过 10MB');
                                      event.target.value = '';
                                      return;
                                    }
                                    setUploadError(null);
                                    setUploadingCount((count) => count + 1);
                                    try {
                                      const response = await transactionsApi.uploadAttachment(file);
                                      if (response.path) field.onChange([...paths, response.path]);
                                    } catch (error) {
                                      setUploadError(error instanceof Error ? error.message : '上传文件失败，请重试');
                                    } finally {
                                      setUploadingCount((count) => count - 1);
                                      event.target.value = '';
                                    }
                                  }}
                                />
                              </label>
                            ) : null}
                          </div>
                        );
                      }}
                    />
                  </div>
                ) : null}
              </div>
            </details>
          </div>

          <TransactionFormFooter
            mode={formMode}
            isPending={isFormPending}
            activeAction={submitAction}
            onCancel={requestClose}
            onContinue={() => {
              setCurrentSubmitAction('continue');
              handleSubmit(
                (values) => onSubmit(values, 'continue'),
                handleInvalidSubmit,
              )();
            }}
            onPrimary={() => setCurrentSubmitAction('close')}
          />
        </form>
        </section>
      </div>

      <ConfirmDialog
        open={showDiscardConfirm}
        title="放弃本次修改？"
        description={uploadingCount > 0
          ? '仍有图片正在上传，离开后本次填写内容和上传进度都会丢失。'
          : '离开后，本次尚未保存的填写内容会丢失。'}
        confirmLabel="放弃修改"
        cancelLabel="继续编辑"
        tone="danger"
        icon={<Info size={20} />}
        onConfirm={closeDrawer}
        onClose={() => setShowDiscardConfirm(false)}
      />

      {/* 另存为模板对话框 */}
      {templateSaveOpen && (
        <div className="modal-overlay" onClick={closeTemplateSave}>
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
                value={resolvedTemplateName}
                onChange={(e) => setTmplName(e.target.value)}
                style={{ width: '100%' }}
                autoFocus
              />
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', flexWrap: 'wrap' }}>
              <button
                type="button"
                className="btn-secondary mobile-full"
                onClick={closeTemplateSave}
              >
                取消
              </button>
              <button
                type="button"
                className="btn-primary mobile-full"
                disabled={createTemplateMutation.isPending || !resolvedTemplateName.trim()}
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
          <div className="modal-content glass-card" style={{ maxWidth: '520px', display: 'flex', flexDirection: 'column', maxHeight: '80vh' }} onClick={(e) => e.stopPropagation()}>
            <h4 style={{ margin: '0 0 16px 0', fontSize: '18px', display: 'flex', gap: '8px', alignItems: 'center' }}>
              <span>管理账单模板</span>
            </h4>
            
            <div className="template-manager-list">
              {templates && templates.length > 0 ? (
                templates.map((tmpl) => (
                  <div key={tmpl.id} className="template-item">
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '4px', overflow: 'hidden' }}>
                      <span style={{ fontSize: '14px', fontWeight: 500, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', opacity: tmpl.is_archived ? 0.58 : 1 }}>{tmpl.name}</span>
                      <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                        类型: {tmpl.type === 'expense' ? '支出' : tmpl.type === 'income' ? '收入' : '共同支出'}
                        {tmpl.amount_cents != null && ` · ¥${(tmpl.amount_cents / 100).toFixed(2)}`}
                        {tmpl.is_archived && ' · 已归档'}
                      </span>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
                      <button
                        type="button"
                        onClick={() => openEditTemplate(tmpl)}
                        style={{
                          background: 'none',
                          border: 'none',
                          color: 'var(--accent-purple)',
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
                        <Pencil size={14} />
                        <span>编辑</span>
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          if (tmpl.is_archived) {
                            restoreTemplateMutation.mutate(tmpl.id);
                          } else if (confirm(`确认归档模板 "${tmpl.name}" 吗？归档后不会出现在快捷填入中，可在此处恢复。`)) {
                            archiveTemplateMutation.mutate(tmpl.id);
                          }
                        }}
                        style={{
                          background: 'none',
                          border: 'none',
                          color: tmpl.is_archived ? 'var(--accent-green)' : '#ef4444',
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
                        {tmpl.is_archived ? <Check size={14} /> : <Trash2 size={14} />}
                        <span>{tmpl.is_archived ? '恢复' : '归档'}</span>
                      </button>
                    </div>
                  </div>
                ))
              ) : (
                <div style={{ textAlign: 'center', padding: '30px 0', color: 'var(--text-muted)' }}>
                  暂无模板，可在记账后点击“存为模板”保存。
                </div>
              )}
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', flexWrap: 'wrap' }}>
              <button
                type="button"
                className="btn-secondary mobile-full"
                onClick={() => setIsManageTmplOpen(false)}
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 模板编辑对话框 */}
      {editingTemplate && (
        <div className="modal-overlay" onClick={() => setEditingTemplate(null)}>
          <div className="modal-content glass-card" style={{ maxWidth: '520px', display: 'flex', flexDirection: 'column', maxHeight: '86vh' }} onClick={(e) => e.stopPropagation()}>
            <h4 style={{ margin: '0 0 16px 0', fontSize: '18px', display: 'flex', gap: '8px', alignItems: 'center' }}>
              <Pencil size={18} style={{ color: 'var(--accent-purple)' }} />
              <span>编辑账单模板</span>
            </h4>
            {editingTemplate.is_archived && (
              <div className="error-banner" style={{ marginBottom: '14px' }}>
                <p>该模板已归档，修改后仍保持归档状态；需要重新出现在快捷填入中请点击恢复。</p>
              </div>
            )}
            {templateEditError && (
              <div className="error-banner" style={{ marginBottom: '14px' }}>
                <p>{templateEditError}</p>
              </div>
            )}
            <div style={{ overflowY: 'auto', paddingRight: '2px' }}>
              <div className="form-group">
                <label className="form-label">模板名称</label>
                <input
                  type="text"
                  className="form-input"
                  value={editingTemplate.name}
                  onChange={(e) => patchEditingTemplate({ name: e.target.value })}
                  autoFocus
                />
              </div>

              <div className="form-group">
                <label className="form-label">账单类型</label>
                <select
                  className="form-select"
                  value={editingTemplate.type}
                  onChange={(e) => patchEditingTemplate({ type: e.target.value as TemplateEditState['type'] })}
                >
                  <option value="expense">个人支出</option>
                  <option value="income">个人收入</option>
                  <option value="shared_expense">共同支出</option>
                </select>
              </div>

              <div className="form-group">
                <label className="form-label">标题</label>
                <input
                  type="text"
                  className="form-input"
                  value={editingTemplate.title}
                  onChange={(e) => patchEditingTemplate({ title: e.target.value })}
                  placeholder="可留空，生成账单时由分类兜底"
                />
              </div>

              <div className="form-group">
                <label className="form-label">金额</label>
                <input
                  type="text"
                  inputMode="decimal"
                  className="form-input"
                  value={editingTemplate.amount}
                  onChange={(e) => patchEditingTemplate({ amount: e.target.value })}
                  placeholder="可留空，生成时再填写"
                />
              </div>

              <div className="form-group">
                <label className="form-label">分类</label>
                <select
                  className="form-select"
                  value={editingTemplate.category_id}
                  onChange={(e) => patchEditingTemplate({ category_id: e.target.value })}
                >
                  <option value="">-- 不指定分类 --</option>
                  {categories?.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </select>
              </div>

              {editingTemplate.type !== 'shared_expense' && (
                <div className="form-group">
                  <label className="form-label">账户</label>
                  <select
                    className="form-select"
                    value={editingTemplate.account_id}
                    onChange={(e) => patchEditingTemplate({ account_id: e.target.value })}
                  >
                    <option value="">-- 不指定账户 --</option>
                    {accounts?.map((account) => (
                      <option key={account.id} value={account.id}>
                        {account.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <div className="form-group">
                <label className="form-label">付款人</label>
                <select
                  className="form-select"
                  value={editingTemplate.payer_user_id}
                  onChange={(e) => patchEditingTemplate({ payer_user_id: e.target.value })}
                >
                  <option value="">-- 生成时再选择 --</option>
                  {users.map((u) => (
                    <option key={u.user_id} value={u.user_id}>
                      {u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
                    </option>
                  ))}
                </select>
              </div>

              {editingTemplate.type === 'shared_expense' && (
                <div className="form-group">
                  <label className="form-label">分摊方式</label>
                  <select
                    className="form-select"
                    value={editingTemplate.split_method}
                    onChange={(e) => patchEditingTemplate({ split_method: e.target.value as TemplateEditState['split_method'] })}
                  >
                    <option value="equal">均等平分</option>
                    <option value="payer_only">付款人承担</option>
                  </select>
                </div>
              )}

              <div className="form-group">
                <label className="form-label">标签</label>
                <input
                  type="text"
                  className="form-input"
                  value={editingTemplate.tag_names}
                  onChange={(e) => patchEditingTemplate({ tag_names: e.target.value })}
                  placeholder="多个标签用逗号分隔"
                />
              </div>

              <div className="form-group">
                <label className="form-label">备注</label>
                <textarea
                  className="form-input textarea"
                  rows={3}
                  value={editingTemplate.note}
                  onChange={(e) => patchEditingTemplate({ note: e.target.value })}
                />
              </div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '16px', flexWrap: 'wrap' }}>
              <button
                type="button"
                className="btn-secondary mobile-full"
                onClick={() => setEditingTemplate(null)}
              >
                取消
              </button>
              <button
                type="button"
                className="btn-primary mobile-full"
                disabled={updateTemplateMutation.isPending || !editingTemplate.name.trim()}
                onClick={handleUpdateTemplate}
              >
                {updateTemplateMutation.isPending ? '保存中...' : '保存修改'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

