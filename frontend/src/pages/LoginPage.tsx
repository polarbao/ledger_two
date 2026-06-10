import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { authApi } from '../api/auth.api';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../stores/auth.store';
import { KeyRound, User as UserIcon, LogIn, Eye, EyeOff } from 'lucide-react';

const loginSchema = z.object({
  username: z.string().min(3, '用户名至少 3 位'),
  password: z.string().min(6, '密码至少 6 位'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const navigate = useNavigate();
  const setUser = useAuthStore((state) => state.setUser);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: '',
      password: '',
    },
  });

  const onSubmit = async (data: LoginFormData) => {
    setErrorMsg(null);
    setLoading(true);
    try {
      const user = await authApi.login(data.username, data.password);
      setUser(user);
      navigate('/');
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : '登录失败，请检查用户名或密码');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page-container">
      <div className="glass-card login-card animate-fade-in">
        <div className="login-header">
          <div className="logo-glow">
            <KeyRound className="icon-glow" />
          </div>
          <h1>LedgerTwo 共享记账</h1>
          <p className="subtitle">欢迎回来，请登录您的账本账户</p>
        </div>

        {errorMsg && <div className="error-banner">{errorMsg}</div>}

        <form onSubmit={handleSubmit(onSubmit)} className="login-form">
          <div className="form-group">
            <label>用户名</label>
            <div className="input-wrapper">
              <UserIcon className="input-icon" />
              <input
                type="text"
                placeholder="请输入用户名"
                {...register('username')}
                className={errors.username ? 'input-error' : ''}
              />
            </div>
            {errors.username && <span className="field-error">{errors.username.message}</span>}
          </div>

          <div className="form-group">
            <label>密码</label>
            <div className="input-wrapper">
              <KeyRound className="input-icon" />
              <input
                type={showPassword ? 'text' : 'password'}
                placeholder="请输入密码"
                {...register('password')}
                className={errors.password ? 'input-error' : ''}
              />
              <button
                type="button"
                className="btn-toggle-password"
                onClick={() => setShowPassword(!showPassword)}
                tabIndex={-1}
              >
                {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
            {errors.password && <span className="field-error">{errors.password.message}</span>}
          </div>

          <button type="submit" disabled={loading} className="btn-primary login-btn">
            {loading ? (
              <span className="spinner-container">
                <span className="btn-spinner"></span> 登录中...
              </span>
            ) : (
              <>
                <LogIn className="btn-icon" /> 立即登录
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
}
