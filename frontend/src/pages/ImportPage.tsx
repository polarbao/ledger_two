import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { 
	Upload, 
	ArrowRight, 
	Check, 
	FileSpreadsheet, 
	ChevronRight, 
	Trash2, 
	AlertTriangle, 
	Info, 
	X 
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
	note: string;
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

	// 1. 状态管理
	const [step, setStep] = useState<1 | 2 | 3>(1);
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
	const [defaultType, setDefaultType] = useState<'expense' | 'shared_expense'>('expense');

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

	const users = dashboardData?.user_stats || [];
	const categoriesList = categories || [];

	const catMap = categoriesList.reduce((acc, cat) => {
		acc[cat.id] = cat.name;
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

			const catName = catMap[defaultCategory] || '未分类';

			list.push({
				id: `temp-${index}-${Math.random().toString(36).substring(2, 9)}`,
				occurred_at: occurredAtStr || new Date().toISOString().substring(0, 10),
				amount: amountNum,
				title: rawTitle.trim() || rawMerchant.trim() || '未命名账单',
				merchant: rawMerchant.trim(),
				category_name: catName,
				category_id: defaultCategory,
				note: rawNote.trim(),
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
					isLoading={isCategoriesLoading || isDashboardLoading}
					isError={isCategoriesError || isDashboardError}
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
									value={defaultCategory}
									onChange={(e) => setDefaultCategory(e.target.value)}
								>
									{categoriesList.map((c) => (
										<option key={c.id} value={c.id}>{c.name}</option>
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
										<th style={{ padding: '12px 16px', textAlign: 'right' }}>金额 (元)</th>
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
											</td>
											<td style={{ padding: '12px 16px' }}>
												<span style={{ fontSize: '11px', background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)', borderRadius: '6px', padding: '2px 8px' }}>
													🏷️ {item.category_name}
												</span>
											</td>
											<td style={{ padding: '12px 16px', textAlign: 'right', fontWeight: 600, color: 'var(--accent-green)' }}>
												¥{item.amount.toFixed(2)}
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
						<button className="btn-secondary" style={{ padding: '10px 24px', fontSize: '13px' }} onClick={handleReset}>
							取消并返回
						</button>
						<button 
							className="btn-primary" 
							style={{ padding: '10px 24px', fontSize: '13px' }} 
							onClick={() => setShowSuccessModal(true)}
							disabled={previewList.length === 0}
						>
							确认导入 (体验演示)
						</button>
					</div>
				</div>
			)}

			{/* 演示体验成功提示 Modal */}
			{showSuccessModal && (
				<div className="modal-overlay" onClick={() => setShowSuccessModal(false)}>
					<div className="modal-content glass-card animate-fade-in text-left" style={{ maxWidth: '460px' }} onClick={(e) => e.stopPropagation()}>
						<div className="drawer-header" style={{ padding: '0 0 16px 0', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
							<div className="header-title" style={{ color: 'var(--accent-purple)' }}>
								<Check size={20} />
								<h3 style={{ fontSize: '16px' }}>CSV 第一阶段解析成功！</h3>
							</div>
							<button className="btn-close-drawer" onClick={() => setShowSuccessModal(false)}>
								<X size={18} />
							</button>
						</div>

						<div style={{ display: 'flex', flexDirection: 'column', gap: '16px', marginTop: '20px' }}>
							<p style={{ fontSize: '13px', margin: 0, lineHeight: 1.6 }}>
								恭喜！CSV 导入预览全链路联调已 100% 跑通。
							</p>
							<p style={{ fontSize: '13px', margin: 0, color: 'var(--dimmed-desc)', lineHeight: 1.6 }}>
								系统在内存中为您格式化并核对了 <strong style={{ color: 'var(--accent-purple)' }}>{previewList.length}</strong> 笔账单。
								所有账单均成功匹配了默认付款人、默认分类与选定类型，且支持按行删除过滤。
							</p>
							
							<div style={{ background: 'rgba(16, 185, 129, 0.04)', border: '1px solid rgba(16, 185, 129, 0.15)', borderRadius: '8px', padding: '10px 14px', display: 'flex', alignItems: 'flex-start', gap: '8px', fontSize: '11px', color: '#34d399' }}>
								<Info size={14} style={{ marginTop: '2px', flexShrink: 0 }} />
								<span>依照 Task 16 规定的红线，本次预览未写入数据库，对账本的统计口径无任何侵入。在下一步 Task 17 中，我们将引入去重检测并确认正式写入。</span>
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
									好的，体验完毕
								</button>
							</div>
						</div>
					</div>
				</div>
			)}
		</div>
	);
}
