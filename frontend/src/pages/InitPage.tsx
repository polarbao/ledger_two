import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { initApi } from '../api/init.api';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/auth.store';
import { Shield, Sparkles, BookOpen, Users } from 'lucide-react';
import DeploymentBadge from '../components/layout/DeploymentBadge';
import ThemeToggle from '../components/theme/ThemeToggle';

const setupSchema = z.object({
  ledger_name: z.string().min(1, '账本名称不能为空'),
  default_currency: z.string().min(1, '默认币种不能为空'),
  user_a_display_name: z.string().min(1, '成员 A 显示名不能为空'),
  user_a_username: z.string().min(3, '成员 A 用户名至少 3 位'),
  user_a_password: z.string().min(6, '成员 A 密码至少 6 位'),
  user_b_display_name: z.string().min(1, '成员 B 显示名不能为空'),
  user_b_username: z.string().min(3, '成员 B 用户名至少 3 位'),
  user_b_password: z.string().min(6, '成员 B 密码至少 6 位'),
}).refine((data) => data.user_a_username !== data.user_b_username, {
  message: '两位成员的用户名不能相同',
  path: ['user_b_username'],
});

type SetupFormData = z.infer<typeof setupSchema>;

export default function InitPage() {
  const navigate = useNavigate();
  const setIsInitialized = useAuthStore((state) => state.setIsInitialized);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SetupFormData>({
    resolver: zodResolver(setupSchema),
    defaultValues: {
      ledger_name: '',
      default_currency: 'CNY',
      user_a_display_name: '',
      user_a_username: '',
      user_a_password: '',
      user_b_display_name: '',
      user_b_username: '',
      user_b_password: '',
    },
  });

  const onSubmit = async (data: SetupFormData) => {
    setErrorMsg(null);
    setLoading(true);
    try {
      await initApi.setup(data);
      setIsInitialized(true);
      navigate('/login');
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : '初始化失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="init-page-container">
      <ThemeToggle className="auth-theme-toggle" />
      <div className="glass-card init-card">
        <div className="init-header">
          <div className="logo-glow">
            <Sparkles className="icon-glow" />
          </div>
          <h1>初始化 LedgerTwo 双人共享账本</h1>
          <p className="subtitle">只需几步配置，即可开启专属你们的透明账本空间。</p>
          <DeploymentBadge />
        </div>

        {errorMsg && <div className="error-banner" role="alert">{errorMsg}</div>}

        <form onSubmit={handleSubmit(onSubmit)} className="init-form">
          <div className="form-section">
            <div className="section-title">
              <BookOpen className="sec-icon" />
              <h3>账本基础信息</h3>
            </div>
            
            <div className="form-row-2">
              <div className="form-group">
                <label htmlFor="setup-ledger-name">账本名称</label>
                <input
                  id="setup-ledger-name"
                  type="text"
                  placeholder="例如：小家温馨账本、情侣日常开销"
                  {...register('ledger_name')}
                  className={errors.ledger_name ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.ledger_name)}
                  aria-describedby={errors.ledger_name ? 'setup-ledger-name-error' : undefined}
                />
                {errors.ledger_name && <span id="setup-ledger-name-error" className="field-error">{errors.ledger_name.message}</span>}
              </div>

              <div className="form-group">
                <label htmlFor="setup-currency">默认币种</label>
                <select
                  id="setup-currency"
                  {...register('default_currency')}
                  className={`form-select ${errors.default_currency ? 'input-error' : ''}`}
                  aria-invalid={Boolean(errors.default_currency)}
                  aria-describedby={errors.default_currency ? 'setup-currency-error' : undefined}
                >
                  <option value="CNY">CNY - 人民币 (¥)</option>
                  <option value="USD">USD - 美元 ($)</option>
                  <option value="EUR">EUR - 欧元 (€)</option>
                  <option value="HKD">HKD - 港币 (HK$)</option>
                </select>
                {errors.default_currency && <span id="setup-currency-error" className="field-error">{errors.default_currency.message}</span>}
              </div>
            </div>
          </div>

          <div className="members-grid">
            {/* 成员 A */}
            <div className="form-section">
              <div className="section-title">
                <Users className="sec-icon text-a" />
                <h3>创建成员 A (你)</h3>
              </div>
              <div className="form-group">
                <label htmlFor="setup-user-a-display-name">显示昵称</label>
                <input
                  id="setup-user-a-display-name"
                  type="text"
                  placeholder="例如：Lynn、Polar"
                  {...register('user_a_display_name')}
                  className={errors.user_a_display_name ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.user_a_display_name)}
                  aria-describedby={errors.user_a_display_name ? 'setup-user-a-display-name-error' : undefined}
                />
                {errors.user_a_display_name && (
                  <span id="setup-user-a-display-name-error" className="field-error">{errors.user_a_display_name.message}</span>
                )}
              </div>
              <div className="form-group">
                <label htmlFor="setup-user-a-username">登录用户名</label>
                <input
                  id="setup-user-a-username"
                  type="text"
                  placeholder="登录账号，至少3个字符"
                  {...register('user_a_username')}
                  className={errors.user_a_username ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.user_a_username)}
                  aria-describedby={errors.user_a_username ? 'setup-user-a-username-error' : undefined}
                  autoComplete="username"
                />
                {errors.user_a_username && (
                  <span id="setup-user-a-username-error" className="field-error">{errors.user_a_username.message}</span>
                )}
              </div>
              <div className="form-group">
                <label htmlFor="setup-user-a-password">密码</label>
                <input
                  id="setup-user-a-password"
                  type="password"
                  placeholder="至少6位密码"
                  {...register('user_a_password')}
                  className={errors.user_a_password ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.user_a_password)}
                  aria-describedby={errors.user_a_password ? 'setup-user-a-password-error' : undefined}
                  autoComplete="new-password"
                />
                {errors.user_a_password && (
                  <span id="setup-user-a-password-error" className="field-error">{errors.user_a_password.message}</span>
                )}
              </div>
            </div>

            {/* 成员 B */}
            <div className="form-section">
              <div className="section-title">
                <Users className="sec-icon text-b" />
                <h3>创建成员 B (伙伴)</h3>
              </div>
              <div className="form-group">
                <label htmlFor="setup-user-b-display-name">显示昵称</label>
                <input
                  id="setup-user-b-display-name"
                  type="text"
                  placeholder="例如：Bob、Alice"
                  {...register('user_b_display_name')}
                  className={errors.user_b_display_name ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.user_b_display_name)}
                  aria-describedby={errors.user_b_display_name ? 'setup-user-b-display-name-error' : undefined}
                />
                {errors.user_b_display_name && (
                  <span id="setup-user-b-display-name-error" className="field-error">{errors.user_b_display_name.message}</span>
                )}
              </div>
              <div className="form-group">
                <label htmlFor="setup-user-b-username">登录用户名</label>
                <input
                  id="setup-user-b-username"
                  type="text"
                  placeholder="伙伴的登录账号，至少3个字符"
                  {...register('user_b_username')}
                  className={errors.user_b_username ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.user_b_username)}
                  aria-describedby={errors.user_b_username ? 'setup-user-b-username-error' : undefined}
                  autoComplete="username"
                />
                {errors.user_b_username && (
                  <span id="setup-user-b-username-error" className="field-error">{errors.user_b_username.message}</span>
                )}
              </div>
              <div className="form-group">
                <label htmlFor="setup-user-b-password">密码</label>
                <input
                  id="setup-user-b-password"
                  type="password"
                  placeholder="伙伴的登录密码"
                  {...register('user_b_password')}
                  className={errors.user_b_password ? 'input-error' : ''}
                  aria-invalid={Boolean(errors.user_b_password)}
                  aria-describedby={errors.user_b_password ? 'setup-user-b-password-error' : undefined}
                  autoComplete="new-password"
                />
                {errors.user_b_password && (
                  <span id="setup-user-b-password-error" className="field-error">{errors.user_b_password.message}</span>
                )}
              </div>
            </div>
          </div>

          <button type="submit" disabled={loading} className="btn-primary init-btn">
            {loading ? (
              <span className="spinner-container">
                <span className="btn-spinner"></span> 初始化配置中...
              </span>
            ) : (
              <>
                <Shield className="btn-icon" /> 开启账本之旅
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
}
