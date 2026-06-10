import { Outlet, useNavigate, useLocation, Link } from 'react-router-dom';
import { useAuthStore } from '../../stores/auth.store';
import { useUIStore } from '../../stores/ui.store';
import { authApi } from '../../api/auth.api';
import {
  LayoutDashboard,
  ReceiptText,
  DollarSign,
  BarChart3,
  Settings,
  LogOut,
  Sparkles,
  Calendar,
} from 'lucide-react';
import TransactionFormDrawer from '../transaction/TransactionFormDrawer';

export default function AppShell() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clear);
  const { currentMonth, setCurrentMonth } = useUIStore();

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch {
      // Ignore network errors
    } finally {
      clearAuth();
      navigate('/login');
    }
  };

  const navItems = [
    { label: '仪表盘', path: '/', icon: LayoutDashboard },
    { label: '账单明细', path: '/transactions', icon: ReceiptText },
    { label: '结算中心', path: '/settlement', icon: DollarSign },
    { label: '分析统计', path: '/analytics', icon: BarChart3 },
    { label: '系统设置', path: '/settings', icon: Settings },
  ];

  return (
    <div className="app-shell">
      {/* 桌面端 Sidebar 侧边栏 */}
      <aside className="sidebar glass-card">
        <div className="sidebar-brand">
          <Sparkles className="brand-logo" />
          <span>LedgerTwo</span>
        </div>

        <nav className="sidebar-nav">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`nav-item ${isActive ? 'active' : ''}`}
              >
                <Icon size={20} />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>

        <div className="sidebar-footer">
          <div className="user-profile">
            <div className="avatar">{user?.displayName?.charAt(0) || 'U'}</div>
            <div className="info">
              <span className="name">{user?.displayName}</span>
              <span className="role">@{user?.username}</span>
            </div>
          </div>
          <button className="btn-logout" onClick={handleLogout}>
            <LogOut size={18} />
            <span>退出登录</span>
          </button>
        </div>
      </aside>

      {/* 主界面区域 */}
      <main className="main-content">
        {/* 顶部 TopBar */}
        <header className="topbar glass-card">
          <div className="mobile-brand">
            <Sparkles size={20} className="brand-logo" />
            <h2>LedgerTwo</h2>
          </div>

          <div className="month-picker-wrapper">
            <Calendar size={18} className="picker-icon" />
            <input
              type="month"
              value={currentMonth}
              onChange={(e) => setCurrentMonth(e.target.value)}
              className="month-picker"
            />
          </div>

          <div className="desktop-user-info">
            <span className="welcome-text">
              你好, <strong className="text-glow">{user?.displayName}</strong>
            </span>
          </div>
        </header>

        {/* 页面内容注入点 */}
        <div className="page-outlet">
          <Outlet />
        </div>
      </main>

      {/* 移动端 BottomTabBar 底部栏 */}
      <nav className="mobile-tabbar glass-card">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          return (
            <Link
              key={item.path}
              to={item.path}
              className={`tab-item ${isActive ? 'active' : ''}`}
            >
              <Icon size={22} />
              <span className="tab-label">{item.label}</span>
            </Link>
          );
        })}
      </nav>

      {/* 记账表单滑出层 */}
      <TransactionFormDrawer />
    </div>
  );
}
