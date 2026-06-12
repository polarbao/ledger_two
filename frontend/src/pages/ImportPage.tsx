import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { ImportItemPayload } from '../types/transaction';
import { 
	Upload, 
	ArrowRight, 
	Check, 
	FileSpreadsheet, 
	ChevronRight, 
	Trash2, 
	AlertTriangle, 
	Info, 
	X,
	Target,
	Sliders
} from 'lucide-react';
import { transactionsApi } from '../api/transactions.api';
import { dashboardApi } from '../api/dashboard.api';
import { useAuthStore } from '../stores/auth.store';
import { ApiError } from '../api/client';
import PageState from '../components/ui/PageState';

interface PreviewTransaction {
	id: string; // 临时前端 UUID，方便删除
	occurred_at: string;
	amount: number; // 元
	title: string;
	merchant: string;
	category_name: string;
	category_id: string;
	account_name: string;
	account_id: string;
	tag_names: string[];
	note: string;
	rule_matched_keyword?: string;
}

interface MappingConfig {
	occurred_at: string;
	amount: string;
	title: string;
	merchant: string;
	note: string;
}

export default function ImportPage() {
	const currentUser = useAuthStore((state) => state.user);
	
	// 当前月份，用于拉取成员
	const currentMonth = new Date().toISOString().substring(0, 7);

	const queryClient = useQueryClient();

	// 1. 状态管理
	const [step, setStep] = useState<1 | 2 | 3>(1);
	const [submitting, setSubmitting] = useState(false);
	const [showConfirmModal, setShowConfirmModal] = useState(false);
	const [analyzeResult, setAnalyzeResult] = useState<{ total_count: number; import_count: number; skip_count: number } | null>(null);
	const [file, setFile] = useState<File | null>(null);
	const [headers, setHeaders] = useState<string[]>([]);
	const [rows, setRows] = useState<string[][]>([]);
	const [parsing, setParsing] = useState(false);
	const [errorMsg, setErrorMsg] = useState<string | null>(null);
	const [showSuccessModal, setShowSuccessModal] = useState(false);

	// 映射配置
	const [mapping, setMapping] = useState<MappingConfig>({
		occurred_at: '',
		amount: '',
		title: '',
		merchant: '',
		note: '',
	});

	// 默认兜底项
	const [defaultPayer, setDefaultPayer] = useState('');
	const [defaultCategory, setDefaultCategory] = useState('');
	const [defaultAccount, setDefaultAccount] = useState('');
	const [defaultType, setDefaultType] = useState<'expense' | 'shared_expense'>('expense');

	// 新建匹配规则表单状态
	const [newRuleKeyword, setNewRuleKeyword] = useState('');
	const [newRuleCategory, setNewRuleCategory] = useState('');
	const [newRuleAccount, setNewRuleAccount] = useState('');
	const [newRuleTags, setNewRuleTags] = useState('');
	const [ruleSubmitting, setRuleSubmitting] = useState(false);

	// 预览数据列表
	const [previewList, setPreviewList] = useState<PreviewTransaction[]>([]);

	// 2. 拉取数据 (React Query)
	const { data: categories, isLoading: isCategoriesLoading, isError: isCategoriesError } = useQuery({
		queryKey: ['categories'],
		queryFn: () => transactionsApi.getCategories(),
		enabled: step === 2,
	});

	const { data: dashboardData, isLoading: isDashboardLoading, isError: isDashboardError } = useQuery({
		queryKey: ['dashboard', currentMonth],
		queryFn: () => dashboardApi.getDashboard(currentMonth),
		enabled: step === 2 && !!currentUser,
	});

	const { data: importRules, isLoading: isRulesLoading, refetch: refetchImportRules } = useQuery({
		queryKey: ['importRules'],
		queryFn: () => transactionsApi.listImportRules(),
		enabled: step === 2,
	});

	const { data: accounts, isLoading: isAccountsLoading, isError: isAccountsError } = useQuery({
		queryKey: ['accounts'],
		queryFn: () => transactionsApi.listAccounts(),
		enabled: step === 2,
	});

	const users = dashboardData?.user_stats || [];
	const categoriesList = categories || [];
	const accountsList = accounts || [];

	const catMap = categoriesList.reduce((acc, cat) => {
		acc[cat.id] = cat.name;
		return acc;
	}, {} as Record<string, string>);

	const accountMap = accountsList.reduce((acc, act) => {
		acc[act.id] = act.name;
		return acc;
	}, {} as Record<string, string>);

	// 3. 文件上传与解析
	const handleFileUpload = async (selectedFile: File) => {
		setFile(selectedFile);
		setParsing(true);
		setErrorMsg(null);

		try {
			const res = await transactionsApi.parseCSV(selectedFile);
			setHeaders(res.headers);
			setRows(res.rows);
			
			// 自动智能匹配列名
			const newMapping = { occurred_at: '', amount: '', title: '', merchant: '', note: '' };
			res.headers.forEach((h) => {
				const headerStr = h.toLowerCase();
				if (headerStr.includes('时间') || headerStr.includes('日期') || headerStr.includes('date') || headerStr.includes('time')) {
					newMapping.occurred_at = h;
				} else if (headerStr.includes('金额') || headerStr.includes('amount') || headerStr.includes('元')) {
					newMapping.amount = h;
				} else if (headerStr.includes('商品') || headerStr.includes('标题') || headerStr.includes('title') || headerStr.includes('名称')) {
					newMapping.title = h;
				} else if (headerStr.includes('对方') || headerStr.includes('商户') || headerStr.includes('merchant') || headerStr.includes('收款')) {
					newMapping.merchant = h;
				} else if (headerStr.includes('备注') || headerStr.includes('说明') || headerStr.includes('note')) {
					newMapping.note = h;
				}
			});
			setMapping(newMapping);

			// 设置默认付款人
			if (currentUser) {
				setDefaultPayer(currentUser.id);
			} else if (users.length > 0) {
				setDefaultPayer(users[0].user_id);
			}

			// 设置默认分类
			if (categoriesList.length > 0) {
				setDefaultCategory(categoriesList[0].id);
			}

			setStep(2);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				setErrorMsg(err.message);
			} else {
				setErrorMsg('文件解析失败，请检查 CSV 格式');
			}
			setFile(null);
		} finally {
			setParsing(false);
		}
	};

	// 4. 重置状态回第一步
	const handleReset = () => {
		setStep(1);
		setFile(null);
		setHeaders([]);
		setRows([]);
		setPreviewList([]);
		setErrorMsg(null);
		setMapping({ occurred_at: '', amount: '', title: '', merchant: '', note: '' });
	};

	// 5. 生成预览数据 (在前端将二维 rows 转换为 PreviewTransaction 数组)
	const handleGeneratePreview = () => {
		if (!mapping.occurred_at || !mapping.amount) {
			setErrorMsg('必须配置“时间列”与“金额列”的映射');
			return;
		}

		const timeIdx = headers.indexOf(mapping.occurred_at);
		const amountIdx = headers.indexOf(mapping.amount);
		const titleIdx = mapping.title ? headers.indexOf(mapping.title) : -1;
		const merchantIdx = mapping.merchant ? headers.indexOf(mapping.merchant) : -1;
		const noteIdx = mapping.note ? headers.indexOf(mapping.note) : -1;

		if (timeIdx === -1 || amountIdx === -1) {
			setErrorMsg('映射列不存在，请核对配置');
			return;
		}

		const currentDefaultCategory = defaultCategory || (categoriesList[0]?.id || '');
		const currentDefaultAccount = defaultAccount || (accountsList[0]?.id || '');
		const rules = importRules || [];

		const list: PreviewTransaction[] = [];
		rows.forEach((row, index) => {
			// 读取时间
			const rawTime = row[timeIdx] || '';
			let occurredAtStr = rawTime.trim();
			// 时间简单修正适配：如果是 YYYYMMDD 转 YYYY-MM-DD
			if (occurredAtStr.length === 8 && /^\d+$/.test(occurredAtStr)) {
				occurredAtStr = `${occurredAtStr.substring(0, 4)}-${occurredAtStr.substring(4, 6)}-${occurredAtStr.substring(6, 8)}`;
			}

			// 读取金额
			const rawAmount = row[amountIdx] || '0';
			// 清理金额中的逗号和货币符号
			const cleanAmountStr = rawAmount.replace(/[¥$,，]/g, '').trim();
			let amountNum = parseFloat(cleanAmountStr);
			if (isNaN(amountNum)) {
				amountNum = 0;
			}
			// 账单导入时金额一般为正数，如带负号则自动取绝对值
			amountNum = Math.abs(amountNum);

			// 读取标题/商户/备注
			const rawTitle = titleIdx !== -1 ? row[titleIdx] : '';
			const rawMerchant = merchantIdx !== -1 ? row[merchantIdx] : '';
			const rawNote = noteIdx !== -1 ? row[noteIdx] : '';

			let catId = currentDefaultCategory;
			let actId = currentDefaultAccount;
			let tagNames: string[] = [];
			let matchedKeyword: string | undefined = undefined;

			const targetTitle = (rawTitle.trim() || rawMerchant.trim() || '').toLowerCase();
			const targetMerchant = rawMerchant.trim().toLowerCase();

			for (const rule of rules) {
				const ruleKw = rule.keyword.toLowerCase();
				if (targetTitle.includes(ruleKw) || targetMerchant.includes(ruleKw)) {
					if (rule.category_id) {
						catId = rule.category_id;
					}
					if (rule.account_id) {
						actId = rule.account_id;
					}
					if (rule.tag_names && rule.tag_names.length > 0) {
						tagNames = rule.tag_names;
					}
					matchedKeyword = rule.keyword;
					break;
				}
			}

			const catName = catMap[catId] || '未分类';
			const actName = accountMap[actId] || '未选账户';

			list.push({
				id: `temp-${index}-${Math.random().toString(36).substring(2, 9)}`,
				occurred_at: occurredAtStr || new Date().toISOString().substring(0, 10),
				amount: amountNum,
				title: rawTitle.trim() || rawMerchant.trim() || '未命名账单',
				merchant: rawMerchant.trim(),
				category_name: catName,
				category_id: catId,
				account_name: actName,
				account_id: actId,
				tag_names: tagNames,
				note: rawNote.trim(),
				rule_matched_keyword: matchedKeyword,
			});
		});

		setPreviewList(list);
		setErrorMsg(null);
		setStep(3);
	};

	// 6. 预览删除单行
	const handleRemovePreviewItem = (id: string) => {
		setPreviewList((prev) => prev.filter((item) => item.id !== id));
	};

	// 7. 去重分析与提交事务落库
	const getPayloadItems = (): ImportItemPayload[] => {
		return previewList.map((item) => ({
			occurred_at: item.occurred_at,
			amount_cents: Math.round(item.amount * 100),
			title: item.title,
			merchant: item.merchant,
			category_id: item.category_id,
			account_id: item.account_id,
			payer_user_id: defaultPayer,
			type: defaultType,
			tag_names: item.tag_names || [],
			note: item.note,
		}));
	};

	const handleAnalyzeImport = async () => {
		setErrorMsg(null);
		setParsing(true);
		try {
			const items = getPayloadItems();
			const res = await transactionsApi.analyzeImport({ items });
			setAnalyzeResult(res);
			setShowConfirmModal(true);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				setErrorMsg(err.message);
			} else {
				setErrorMsg('去重预分析失败，请稍后重试');
			}
		} finally {
			setParsing(false);
		}
	};

	const handleCommitImport = async () => {
		setErrorMsg(null);
		setSubmitting(true);
		try {
			const items = getPayloadItems();
			await transactionsApi.commitImport({
				filename: file?.name || 'statement.csv',
				items,
			});
			setShowConfirmModal(false);
			
			// 刷新 TanStack 缓存
			queryClient.invalidateQueries({ queryKey: ['transactions'] });
			queryClient.invalidateQueries({ queryKey: ['dashboard'] });
			
			// 显示成功弹窗
			setShowSuccessModal(true);
		} catch (err: unknown) {
			if (err instanceof ApiError) {
				setErrorMsg(err.message);
			} else {
				setErrorMsg('账单写入失败，事务已安全回滚');
			}
			setShowConfirmModal(false);
		} finally {
			setSubmitting(false);
		}
	};

	return (
		<div className="page-content animate-fade-in text-left">
			{/* 头部 Banner */}
			<div className="glass-card header-banner">
				<FileSpreadsheet className="banner-icon" />
				<div>
					<h2>CSV 账单导入中心</h2>
					<p>支持快速批量解析微信、支付宝等账单。当前处于第一阶段：基础上传、智能映射与临时数据预览工作区。</p>
				</div>
			</div>

			{/* 错误提示栏 */}
			{errorMsg && (
				<div className="error-banner animate-fade-in" style={{ margin: '0 0 16px 0', borderRadius: '12px' }}>
					<AlertTriangle size={18} style={{ marginRight: '8px', flexShrink: 0 }} />
					<span>{errorMsg}</span>
					<button className="btn-close-drawer" style={{ marginLeft: 'auto', background: 'none', border: 'none', color: 'inherit', cursor: 'pointer' }} onClick={() => setErrorMsg(null)}>
						<X size={16} />
					</button>
				</div>
			)}

			{/* 步骤条 (Step Wizard) */}
			<div className="glass-card" style={{ padding: '16px 24px', marginBottom: '20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', overflowX: 'auto', gap: '10px' }}>
				<div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: step >= 1 ? 'var(--accent-purple)' : 'var(--dimmed-desc)' }}>
					<div style={{ width: '24px', height: '24px', borderRadius: '50%', border: '1px solid currentColor', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 600 }}>
						{step > 1 ? <Check size={12} /> : '1'}
					</div>
					<span style={{ fontSize: '13px', fontWeight: 500 }}>上传账单 CSV</span>
				</div>
				<ChevronRight size={16} className="dimmed-desc" />
				<div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: step >= 2 ? 'var(--accent-purple)' : 'var(--dimmed-desc)' }}>
					<div style={{ width: '24px', height: '24px', borderRadius: '50%', border: '1px solid currentColor', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 600 }}>
						{step > 2 ? <Check size={12} /> : '2'}
					</div>
					<span style={{ fontSize: '13px', fontWeight: 500 }}>配置字段映射</span>
				</div>
				<ChevronRight size={16} className="dimmed-desc" />
				<div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: step >= 3 ? 'var(--accent-purple)' : 'var(--dimmed-desc)' }}>
					<div style={{ width: '24px', height: '24px', borderRadius: '50%', border: '1px solid currentColor', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 600 }}>
						3
					</div>
					<span style={{ fontSize: '13px', fontWeight: 500 }}>账单数据确认</span>
				</div>
			</div>

			{/* 步骤 1: 上传区 */}
			{step === 1 && (
				<div className="glass-card" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '60px 20px', minHeight: '300px', border: '2px dashed rgba(255, 255, 255, 0.08)', borderRadius: '16px' }}>
					{parsing ? (
						<div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}>
							<div className="shimmer-block" style={{ width: '48px', height: '48px', borderRadius: '50%' }} />
							<p style={{ margin: 0, fontSize: '14px' }}>正在为您分析 CSV 数据并自动识别文件编码...</p>
						</div>
					) : (
						<label style={{ cursor: 'pointer', display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%' }}>
							<div style={{ width: '64px', height: '64px', borderRadius: '50%', background: 'rgba(147, 51, 234, 0.08)', border: '1px solid rgba(147, 51, 234, 0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: '16px' }}>
								<Upload size={28} style={{ color: 'var(--accent-purple)' }} />
							</div>
							<h3 style={{ margin: '0 0 8px 0', fontSize: '16px' }}>点击或将账单 CSV 拖拽到这里</h3>
							<p className="dimmed-desc" style={{ fontSize: '12px', margin: '0 0 16px 0', maxWidth: '360px', textAlign: 'center' }}>
								支持 UTF-8 及 GBK 解码。自动跳过微信/支付宝账单顶部的描述说明信息，智能提取表头。
							</p>
							<input 
								type="file" 
								accept=".csv" 
								style={{ display: 'none' }} 
								onChange={(e) => {
									const files = e.target.files;
									if (files && files.length > 0) {
										handleFileUpload(files[0]);
									}
								}}
							/>
							<div className="btn-secondary" style={{ padding: '8px 24px', fontSize: '13px' }}>
								选择本地账单
							</div>
						</label>
					)}
				</div>
			)}

			{/* 步骤 2: 映射区 */}
			{step === 2 && (
				<PageState 
					isLoading={isCategoriesLoading || isDashboardLoading || isAccountsLoading}
					isError={isCategoriesError || isDashboardError || isAccountsError}
				>
					<div className="form-row-2">
						{/* 左栏：配置映射关系 */}
						<div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
							<div style={{ borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '10px' }}>
								<h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600 }}>① 字段列映射</h3>
								<p className="dimmed-desc" style={{ fontSize: '11px', margin: '4px 0 0 0' }}>请为系统属性分派 CSV 对应的列头，系统已根据智能识别自动推荐匹配。</p>
							</div>

							<div className="form-group">
								<label>发生时间列 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
								<select 
									className="filter-input"
									value={mapping.occurred_at}
									onChange={(e) => setMapping((prev) => ({ ...prev, occurred_at: e.target.value }))}
								>
									<option value="">-- 请选择 --</option>
									{headers.map((h) => <option key={h} value={h}>{h}</option>)}
								</select>
							</div>

							<div className="form-group">
								<label>账单金额列 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
								<select 
									className="filter-input"
									value={mapping.amount}
									onChange={(e) => setMapping((prev) => ({ ...prev, amount: e.target.value }))}
								>
									<option value="">-- 请选择 --</option>
									{headers.map((h) => <option key={h} value={h}>{h}</option>)}
								</select>
							</div>

							<div className="form-group">
								<label>账单标题列 (商品名称)</label>
								<select 
									className="filter-input"
									value={mapping.title}
									onChange={(e) => setMapping((prev) => ({ ...prev, title: e.target.value }))}
								>
									<option value="">-- 未选择 (默认空) --</option>
									{headers.map((h) => <option key={h} value={h}>{h}</option>)}
								</select>
							</div>

							<div className="form-group">
								<label>交易对方列 (商户名称)</label>
								<select 
									className="filter-input"
									value={mapping.merchant}
									onChange={(e) => setMapping((prev) => ({ ...prev, merchant: e.target.value }))}
								>
									<option value="">-- 未选择 (默认空) --</option>
									{headers.map((h) => <option key={h} value={h}>{h}</option>)}
								</select>
							</div>

							<div className="form-group">
								<label>备注说明列</label>
								<select 
									className="filter-input"
									value={mapping.note}
									onChange={(e) => setMapping((prev) => ({ ...prev, note: e.target.value }))}
								>
									<option value="">-- 未选择 (默认空) --</option>
									{headers.map((h) => <option key={h} value={h}>{h}</option>)}
								</select>
							</div>
						</div>

						{/* 右栏：默认兜底与全局配置 */}
						<div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
							<div style={{ borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '10px' }}>
								<h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600 }}>② 默认导入规则</h3>
								<p className="dimmed-desc" style={{ fontSize: '11px', margin: '4px 0 0 0' }}>外部账单缺少系统标识。在此处指定默认值用以补充空缺参数。</p>
							</div>

							<div className="form-group">
								<label>默认付款人 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
								<select 
									className="filter-input"
									value={defaultPayer}
									onChange={(e) => setDefaultPayer(e.target.value)}
								>
									{users.map((u) => (
										<option key={u.user_id} value={u.user_id}>
											{u.display_name} {u.user_id === currentUser?.id ? '(我)' : ''}
										</option>
									))}
								</select>
							</div>

							<div className="form-group">
								<label>默认导入分类 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
								<select 
									className="filter-input"
									value={defaultCategory || (categoriesList[0]?.id || '')}
									onChange={(e) => setDefaultCategory(e.target.value)}
								>
									{categoriesList.map((c) => (
										<option key={c.id} value={c.id}>{c.name}</option>
									))}
								</select>
							</div>

							<div className="form-group">
								<label>默认导入账户 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
								<select 
									className="filter-input"
									value={defaultAccount || (accountsList[0]?.id || '')}
									onChange={(e) => setDefaultAccount(e.target.value)}
								>
									{accountsList.map((a) => (
										<option key={a.id} value={a.id}>{a.name}</option>
									))}
								</select>
							</div>

							<div className="form-group">
								<label>默认账单类型 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
								<div style={{ display: 'flex', gap: '8px' }}>
									<button 
										className={`filter-input ${defaultType === 'expense' ? 'btn-primary' : ''}`}
										style={{ flex: 1, padding: '10px' }}
										onClick={() => setDefaultType('expense')}
									>
										个人支出
									</button>
									<button 
										className={`filter-input ${defaultType === 'shared_expense' ? 'btn-primary' : ''}`}
										style={{ flex: 1, padding: '10px' }}
										onClick={() => setDefaultType('shared_expense')}
									>
										共同支出
									</button>
								</div>
							</div>

							{/* CSV 片段展示 */}
							<div style={{ background: 'rgba(255,255,255,0.01)', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '12px', padding: '14px', marginTop: '10px' }}>
								<strong style={{ fontSize: '12px', display: 'block', marginBottom: '8px' }}>📁 CSV 数据片段核对 ({file?.name})：</strong>
								<div style={{ overflowX: 'auto', fontSize: '11px', color: 'var(--dimmed-desc)' }}>
									<table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
										<thead>
											<tr>
												{headers.slice(0, 3).map((h) => <th key={h} style={{ padding: '6px', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>{h}</th>)}
												{headers.length > 3 && <th style={{ padding: '6px', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>...</th>}
											</tr>
										</thead>
										<tbody>
											{rows.slice(0, 2).map((r, ri) => (
												<tr key={ri}>
													{r.slice(0, 3).map((c, ci) => <td key={ci} style={{ padding: '6px' }}>{c}</td>)}
													{r.length > 3 && <td style={{ padding: '6px' }}>...</td>}
												</tr>
											))}
										</tbody>
									</table>
								</div>
							</div>

							<div style={{ display: 'flex', gap: '10px', marginTop: 'auto', paddingTop: '16px' }}>
								<button className="btn-secondary" style={{ flex: 1, padding: '10px' }} onClick={handleReset}>
									重新上传
								</button>
								<button className="btn-primary" style={{ flex: 2, padding: '10px' }} onClick={handleGeneratePreview}>
									生成数据预览 <ArrowRight size={14} style={{ marginLeft: '4px' }} />
								</button>
							</div>
						</div>
					</div>

					{/* 导入分类规则管理器 */}
					<div className="glass-card" style={{ marginTop: '20px', padding: '24px' }}>
						<div style={{ display: 'flex', alignItems: 'center', gap: '8px', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '12px', marginBottom: '20px' }}>
							<Sliders size={18} style={{ color: 'var(--accent-purple)' }} />
							<h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600 }}>导入分类规则管理器</h3>
						</div>

						<div style={{ display: 'grid', gridTemplateColumns: '1.2fr 0.8fr', gap: '24px' }}>
							{/* 规则列表 */}
							<div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
								<span style={{ fontSize: '12px', fontWeight: 600, color: 'var(--dimmed-desc)' }}>📋 当前已配置规则：</span>
								
								{isRulesLoading ? (
									<div style={{ padding: '20px 0', fontSize: '12px', color: 'var(--dimmed-desc)' }}>加载规则列表中...</div>
								) : !importRules || importRules.length === 0 ? (
									<div style={{ padding: '20px 10px', fontSize: '12px', color: 'var(--dimmed-desc)', border: '1px dashed rgba(255,255,255,0.06)', borderRadius: '8px', textAlign: 'center' }}>
										暂无匹配规则，您可以在右侧配置并新增。
									</div>
								) : (
									<div style={{ maxHeight: '250px', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: '8px', paddingRight: '4px' }}>
										{importRules.map((rule) => (
											<div 
												key={rule.id} 
												style={{ 
													display: 'flex', 
													alignItems: 'center', 
													justifyContent: 'space-between', 
													background: 'rgba(255, 255, 255, 0.01)', 
													border: '1px solid rgba(255, 255, 255, 0.04)', 
													borderRadius: '10px', 
													padding: '10px 14px', 
													fontSize: '12px' 
												}}
											>
												<div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
													<div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
														<span style={{ fontWeight: 600, background: 'rgba(147, 51, 234, 0.1)', border: '1px solid rgba(147, 51, 234, 0.2)', padding: '2px 6px', borderRadius: '4px', color: '#c084fc' }}>
															🔍 '{rule.keyword}'
														</span>
														<span className="dimmed-desc">自动归类为：</span>
														<span style={{ fontWeight: 500 }}>{catMap[rule.category_id] || '未分类'}</span>
													</div>
													<div style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '11px', color: 'var(--dimmed-desc)' }}>
														<span>🏦 账户: {accountMap[rule.account_id] || '未关联账户'}</span>
														{rule.tag_names && rule.tag_names.length > 0 && (
															<span style={{ display: 'flex', gap: '4px' }}>
																🏷️ 标签:
																{rule.tag_names.map(tag => (
																	<span key={tag} style={{ background: 'rgba(255,255,255,0.05)', padding: '0px 4px', borderRadius: '3px' }}>{tag}</span>
																))}
															</span>
														)}
													</div>
												</div>
												<button 
													className="btn-close-drawer" 
													style={{ color: '#ef4444', padding: '6px', cursor: 'pointer' }}
													onClick={async () => {
														try {
															await transactionsApi.deleteImportRule(rule.id);
															refetchImportRules();
														} catch {
															setErrorMsg('删除匹配规则失败');
														}
													}}
												>
													<Trash2 size={14} />
												</button>
											</div>
										))}
									</div>
								)}
							</div>

							{/* 新增表单 */}
							<div 
								style={{ 
									background: 'rgba(255, 255, 255, 0.01)', 
									border: '1px solid rgba(255, 255, 255, 0.04)', 
									borderRadius: '12px', 
									padding: '16px', 
									display: 'flex', 
									flexDirection: 'column', 
									gap: '12px' 
								}}
							>
								<span style={{ fontSize: '12px', fontWeight: 600, color: 'var(--dimmed-desc)' }}>➕ 新增自动匹配规则：</span>
								
								<div className="form-group" style={{ margin: 0 }}>
									<label style={{ fontSize: '11px', marginBottom: '4px' }}>商户/商品关键词 <span style={{ color: 'var(--accent-purple)' }}>*</span></label>
									<input 
										type="text" 
										className="filter-input" 
										style={{ padding: '8px 10px', fontSize: '12px' }}
										placeholder="例如：星巴克、滴滴、美团"
										value={newRuleKeyword}
										onChange={(e) => setNewRuleKeyword(e.target.value)}
									/>
								</div>

								<div className="form-group" style={{ margin: 0 }}>
									<label style={{ fontSize: '11px', marginBottom: '4px' }}>自动设置分类</label>
									<select 
										className="filter-input" 
										style={{ padding: '8px 10px', fontSize: '12px' }}
										value={newRuleCategory || (categoriesList[0]?.id || '')}
										onChange={(e) => setNewRuleCategory(e.target.value)}
									>
										{categoriesList.map((c) => (
											<option key={c.id} value={c.id}>{c.name}</option>
										))}
									</select>
								</div>

								<div className="form-group" style={{ margin: 0 }}>
									<label style={{ fontSize: '11px', marginBottom: '4px' }}>自动匹配账户</label>
									<select 
										className="filter-input" 
										style={{ padding: '8px 10px', fontSize: '12px' }}
										value={newRuleAccount || (accountsList[0]?.id || '')}
										onChange={(e) => setNewRuleAccount(e.target.value)}
									>
										{accountsList.map((a) => (
											<option key={a.id} value={a.id}>{a.name}</option>
										))}
									</select>
								</div>

								<div className="form-group" style={{ margin: 0 }}>
									<label style={{ fontSize: '11px', marginBottom: '4px' }}>自动追加标签 (逗号分隔)</label>
									<input 
										type="text" 
										className="filter-input" 
										style={{ padding: '8px 10px', fontSize: '12px' }}
										placeholder="例如：咖啡, 餐饮 (可选)"
										value={newRuleTags}
										onChange={(e) => setNewRuleTags(e.target.value)}
									/>
								</div>

								<button 
									className="btn-primary" 
									style={{ padding: '8px', fontSize: '12px', marginTop: '4px', width: '100%' }}
									disabled={ruleSubmitting}
									onClick={async () => {
										if (!newRuleKeyword.trim()) {
											setErrorMsg('商户/商品关键词不能为空');
											return;
										}
										setRuleSubmitting(true);
										try {
											const catVal = newRuleCategory || categoriesList[0]?.id;
											const actVal = newRuleAccount || accountsList[0]?.id;
											const tagsVal = newRuleTags.split(/[,，]/).map(t => t.trim()).filter(Boolean);
											
											await transactionsApi.createImportRule({
												keyword: newRuleKeyword.trim(),
												category_id: catVal,
												account_id: actVal,
												tag_names: tagsVal,
											});

											setNewRuleKeyword('');
											setNewRuleTags('');
											refetchImportRules();
										} catch {
											setErrorMsg('创建匹配规则失败，请检查参数');
										} finally {
											setRuleSubmitting(false);
										}
									}}
								>
									{ruleSubmitting ? '正在添加...' : '添加规则'}
								</button>
							</div>
						</div>
					</div>
				</PageState>
			)}

			{/* 步骤 3: 预览列表 */}
			{step === 3 && (
				<div className="glass-card" style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
					<div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '12px' }}>
						<div>
							<h3 style={{ margin: 0, fontSize: '15px', fontWeight: 600 }}>③ 预览待导入账单</h3>
							<span className="dimmed-desc" style={{ fontSize: '11px' }}>
								已从文件 <strong style={{ color: 'var(--accent-purple)' }}>{file?.name}</strong> 中成功格式化了 <strong style={{ color: 'var(--accent-purple)' }}>{previewList.length}</strong> 笔交易。您可以点击删除按钮移除个别无需导入的行。
							</span>
						</div>
						<div style={{ fontSize: '12px', background: 'rgba(147, 51, 234, 0.05)', border: '1px solid rgba(147, 51, 234, 0.15)', borderRadius: '8px', padding: '6px 12px', color: '#c084fc', display: 'flex', alignItems: 'center', gap: '6px' }}>
							<Info size={14} />
							<span>纯内存预览 · 不入库</span>
						</div>
					</div>

					{/* 预览卡片/表格 */}
					{previewList.length === 0 ? (
						<div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '60px 0', color: 'var(--dimmed-desc)' }}>
							<Trash2 size={40} style={{ opacity: 0.3, marginBottom: '12px' }} />
							<p style={{ margin: 0, fontSize: '14px' }}>预览列表中已无待导入账单，您可以返回重新配置。</p>
						</div>
					) : (
						<div style={{ maxHeight: '420px', overflowY: 'auto', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '12px', background: 'rgba(10,12,16,0.2)' }}>
							<table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
								<thead>
									<tr style={{ background: 'rgba(255,255,255,0.02)', borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
										<th style={{ padding: '12px 16px', textAlign: 'left' }}>交易时间</th>
										<th style={{ padding: '12px 16px', textAlign: 'left' }}>账单商品/商户</th>
										<th style={{ padding: '12px 16px', textAlign: 'left' }}>默认分类</th>
										<th style={{ padding: '12px 16px', textAlign: 'left' }}>划账账户</th>
										<th style={{ padding: '12px 16px', textAlign: 'right' }}>金额 (元)</th>
										<th style={{ padding: '12px 16px', textAlign: 'left' }}>标签</th>
										<th style={{ padding: '12px 16px', textAlign: 'left' }}>备注</th>
										<th style={{ padding: '12px 16px', textAlign: 'center' }}>操作</th>
									</tr>
								</thead>
								<tbody>
									{previewList.map((item) => (
										<tr key={item.id} style={{ borderBottom: '1px solid rgba(255,255,255,0.03)' }}>
											<td style={{ padding: '12px 16px', whiteSpace: 'nowrap' }}>{item.occurred_at}</td>
											<td style={{ padding: '12px 16px' }}>
												<div style={{ fontWeight: 500 }}>{item.title}</div>
												{item.merchant && <div className="dimmed-desc" style={{ fontSize: '11px', marginTop: '2px' }}>商户: {item.merchant}</div>}
												{item.rule_matched_keyword && (
													<div style={{ display: 'inline-flex', alignItems: 'center', gap: '3px', fontSize: '10px', background: 'rgba(16, 185, 129, 0.08)', border: '1px solid rgba(16, 185, 129, 0.15)', borderRadius: '4px', padding: '1px 6px', marginTop: '4px', color: '#34d399' }}>
														<Target size={10} />
														<span>规则匹配: '{item.rule_matched_keyword}'</span>
													</div>
												)}
											</td>
											<td style={{ padding: '12px 16px' }}>
												<select
													className="filter-input"
													style={{ padding: '4px 8px', fontSize: '12px', width: 'auto', minWidth: '110px' }}
													value={item.category_id}
													onChange={(e) => {
														const newCatId = e.target.value;
														const newCatName = catMap[newCatId] || '未分类';
														setPreviewList((prev) =>
															prev.map((p) =>
																p.id === item.id
																	? { ...p, category_id: newCatId, category_name: newCatName }
																	: p
															)
														);
													}}
												>
													{categoriesList.map((c) => (
														<option key={c.id} value={c.id}>{c.name}</option>
													))}
												</select>
											</td>
											<td style={{ padding: '12px 16px' }}>
												<select
													className="filter-input"
													style={{ padding: '4px 8px', fontSize: '12px', width: 'auto', minWidth: '110px' }}
													value={item.account_id}
													onChange={(e) => {
														const newAccId = e.target.value;
														const newAccName = accountMap[newAccId] || '未选账户';
														setPreviewList((prev) =>
															prev.map((p) =>
																p.id === item.id
																	? { ...p, account_id: newAccId, account_name: newAccName }
																	: p
															)
														);
													}}
												>
													{accountsList.map((a) => (
														<option key={a.id} value={a.id}>{a.name}</option>
													))}
												</select>
											</td>
											<td style={{ padding: '12px 16px', textAlign: 'right', fontWeight: 600, color: 'var(--accent-green)' }}>
												¥{item.amount.toFixed(2)}
											</td>
											<td style={{ padding: '12px 16px' }}>
												{item.tag_names && item.tag_names.length > 0 ? (
													<div style={{ display: 'flex', flexWrap: 'wrap', gap: '3px' }}>
														{item.tag_names.map((tag) => (
															<span key={tag} style={{ fontSize: '10px', background: 'rgba(255,255,255,0.06)', borderRadius: '4px', padding: '1px 5px' }}>
																{tag}
															</span>
														))}
													</div>
												) : (
													<span className="dimmed-desc">-</span>
												)}
											</td>
											<td style={{ padding: '12px 16px', color: 'var(--dimmed-desc)' }}>{item.note || '-'}</td>
											<td style={{ padding: '12px 16px', textAlign: 'center' }}>
												<button 
													className="btn-close-drawer" 
													style={{ color: '#ef4444' }}
													onClick={() => handleRemovePreviewItem(item.id)}
												>
													<Trash2 size={15} />
												</button>
											</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>
					)}

					<div style={{ display: 'flex', gap: '12px', borderTop: '1px solid rgba(255,255,255,0.05)', paddingTop: '16px', justifyContent: 'flex-end' }}>
						<button className="btn-secondary" style={{ padding: '10px 24px', fontSize: '13px' }} onClick={handleReset} disabled={parsing || submitting}>
							取消并返回
						</button>
						<button 
							className="btn-primary" 
							style={{ padding: '10px 24px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '6px' }} 
							onClick={handleAnalyzeImport}
							disabled={previewList.length === 0 || parsing || submitting}
						>
							{parsing ? (
								<>
									<div className="shimmer-block" style={{ width: '12px', height: '12px', borderRadius: '50%' }} />
									<span>去重分析中...</span>
								</>
							) : (
								'确认导入'
							)}
						</button>
					</div>
				</div>
			)}

			{/* 二次高风险批量入账确认 Modal */}
			{showConfirmModal && analyzeResult && (
				<div className="modal-overlay" onClick={() => setShowConfirmModal(false)}>
					<div className="modal-content glass-card animate-fade-in text-left" style={{ maxWidth: '480px' }} onClick={(e) => e.stopPropagation()}>
						<div className="drawer-header" style={{ padding: '0 0 16px 0', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
							<div className="header-title" style={{ color: 'var(--accent-purple)' }}>
								<AlertTriangle size={20} style={{ color: '#fbbf24' }} />
								<h3 style={{ fontSize: '16px', fontWeight: 600 }}>确认导入这些账单？</h3>
							</div>
							<button className="btn-close-drawer" onClick={() => setShowConfirmModal(false)} disabled={submitting}>
								<X size={18} />
							</button>
						</div>

						<div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '20px' }}>
							<p style={{ fontSize: '13px', margin: 0, lineHeight: 1.6, color: 'var(--dimmed-desc)' }}>
								系统将分析并导入当前工作区中的交易，若有在系统中已存在相同指纹的交易，将会被自动跳过以保障账本的唯一性与统计口径稳定。
							</p>

							{/* 去重对比指标数据卡片 */}
							<div style={{ background: 'rgba(255,255,255,0.01)', border: '1px solid rgba(255,255,255,0.04)', borderRadius: '12px', padding: '16px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
								<strong style={{ fontSize: '12px', color: 'var(--dimmed-desc)' }}>📊 去重筛选结果统计：</strong>
								<div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '10px', textAlign: 'center' }}>
									<div style={{ background: 'rgba(255,255,255,0.02)', borderRadius: '8px', padding: '10px' }}>
										<div style={{ fontSize: '18px', fontWeight: 700, color: '#e2e8f0' }}>{analyzeResult.total_count}</div>
										<div style={{ fontSize: '10px', color: 'var(--dimmed-desc)', marginTop: '2px' }}>总笔数</div>
									</div>
									<div style={{ background: 'rgba(16, 185, 129, 0.05)', border: '1px solid rgba(16, 185, 129, 0.15)', borderRadius: '8px', padding: '10px' }}>
										<div style={{ fontSize: '18px', fontWeight: 700, color: '#34d399' }}>{analyzeResult.import_count}</div>
										<div style={{ fontSize: '10px', color: '#10b981', marginTop: '2px' }}>待导入 (新增)</div>
									</div>
									<div style={{ background: 'rgba(239, 68, 68, 0.05)', border: '1px solid rgba(239, 68, 68, 0.15)', borderRadius: '8px', padding: '10px' }}>
										<div style={{ fontSize: '18px', fontWeight: 700, color: '#f87171' }}>{analyzeResult.skip_count}</div>
										<div style={{ fontSize: '10px', color: '#ef4444', marginTop: '2px' }}>重复跳过</div>
									</div>
								</div>
							</div>

							{/* 高风险操作审计日志告警 */}
							<div style={{ background: 'rgba(245, 158, 11, 0.05)', border: '1px solid rgba(245, 158, 11, 0.15)', borderRadius: '12px', padding: '12px 14px', display: 'flex', alignItems: 'flex-start', gap: '10px', fontSize: '11px', color: '#fbbf24', lineHeight: 1.5 }}>
								<Info size={15} style={{ flexShrink: 0, marginTop: '2px' }} />
								<span>此操作为批量写入动作。一旦确认，待导入项将在完整的原子事务（sql.Tx）内落库并记录至全局操作审计日志中以做备查。</span>
							</div>

							<div className="drawer-footer" style={{ borderTop: 'none', padding: 0, marginTop: '12px', display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
								<button 
									className="btn-secondary" 
									style={{ padding: '8px 20px', fontSize: '13px', borderRadius: '10px' }} 
									onClick={() => setShowConfirmModal(false)}
									disabled={submitting}
								>
									取消
								</button>
								<button 
									className="btn-primary" 
									style={{ padding: '8px 24px', fontSize: '13px', borderRadius: '10px', display: 'flex', alignItems: 'center', gap: '6px' }} 
									onClick={handleCommitImport}
									disabled={submitting}
								>
									{submitting ? (
										<>
											<div className="shimmer-block" style={{ width: '12px', height: '12px', borderRadius: '50%' }} />
											<span>正在安全入账...</span>
										</>
									) : (
										'确认写入'
									)}
								</button>
							</div>
						</div>
					</div>
				</div>
			)}

			{/* 账单真正导入成功提示 Modal */}
			{showSuccessModal && (
				<div className="modal-overlay" onClick={() => { setShowSuccessModal(false); handleReset(); }}>
					<div className="modal-content glass-card animate-fade-in text-left" style={{ maxWidth: '460px' }} onClick={(e) => e.stopPropagation()}>
						<div className="drawer-header" style={{ padding: '0 0 16px 0', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
							<div className="header-title" style={{ color: 'var(--accent-green)' }}>
								<Check size={20} style={{ color: '#10b981' }} />
								<h3 style={{ fontSize: '16px', fontWeight: 600 }}>CSV 账单导入成功！</h3>
							</div>
							<button className="btn-close-drawer" onClick={() => { setShowSuccessModal(false); handleReset(); }}>
								<X size={18} />
							</button>
						</div>

						<div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '20px' }}>
							<p style={{ fontSize: '13px', margin: 0, lineHeight: 1.6 }}>
								恭喜！CSV 账单导入已安全完成。
							</p>
							{analyzeResult && (
								<p style={{ fontSize: '13px', margin: 0, color: 'var(--dimmed-desc)', lineHeight: 1.6 }}>
									系统已处理并记录 <strong style={{ color: 'var(--accent-purple)' }}>{analyzeResult.total_count}</strong> 笔账单：
									成功新增导入了 <strong style={{ color: 'var(--accent-green)' }}>{analyzeResult.import_count}</strong> 笔交易流向账表，
									并自动跳过了 <strong style={{ color: '#f87171' }}>{analyzeResult.skip_count}</strong> 笔重复存在的账单。
								</p>
							)}
							
							<div style={{ background: 'rgba(16, 185, 129, 0.04)', border: '1px solid rgba(16, 185, 129, 0.15)', borderRadius: '8px', padding: '10px 14px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#34d399' }}>
								<Info size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
								<span>依照设计，本次批量导入已经包裹在事务内安全落库，并在系统中生成了导入审计日志。流水表和主仪表盘已刷新缓存。</span>
							</div>

							<div className="drawer-footer" style={{ borderTop: 'none', padding: 0, marginTop: '8px', display: 'flex', justifyContent: 'flex-end' }}>
								<button 
									className="btn-primary" 
									style={{ padding: '10px 24px', fontSize: '13px', borderRadius: '10px' }} 
									onClick={() => {
										setShowSuccessModal(false);
										handleReset();
									}}
								>
									好的，导入完成
								</button>
							</div>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
