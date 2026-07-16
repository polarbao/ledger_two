import { describe, it, expect, vi, beforeEach } from 'vitest';
import { request, ApiError } from './client';
import { useLedgerStore } from '../stores/ledger.store';

describe('API Client Error Parsing', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
	useLedgerStore.getState().clearActiveLedger();
  });

  it('should parse success response correctly', async () => {
    const mockResponse = {
      success: true,
      data: { id: '123', name: 'Test' },
    };

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockResponse),
    }));

	const data = await request<{ id: string; name: string }>('/api/test', { ledgerScope: 'none' });
    expect(data).toEqual({ id: '123', name: 'Test' });
  });

  it('should parse validation error details correctly', async () => {
    const mockErrorResponse = {
      success: false,
      error: {
        code: 'VALIDATION_ERROR',
        message: '金额必须大于 0',
        details: { amount: '金额无效' },
      },
    };

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: () => Promise.resolve(mockErrorResponse),
    }));

    try {
		await request('/api/test', { ledgerScope: 'none' });
      expect.fail('Expected request to throw ApiError');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      const apiErr = err as ApiError;
      expect(apiErr.code).toBe('VALIDATION_ERROR');
      expect(apiErr.message).toBe('金额必须大于 0');
      expect(apiErr.status).toBe(400);
      expect(apiErr.details).toEqual({ amount: '金额无效' });
    }
  });

  it('should handle unauthorized error and bypass redirect if window is not mockable', async () => {
    const mockErrorResponse = {
      success: false,
      error: {
        code: 'UNAUTHORIZED',
        message: '请先登录系统',
        details: null,
      },
    };

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: () => Promise.resolve(mockErrorResponse),
    }));

    // 显式绕过 window 拦截或模拟
    vi.stubGlobal('window', {
      location: {
        pathname: '/login',
        href: '/login',
      },
    });

    try {
		await request('/api/test', { ledgerScope: 'none' });
      expect.fail('Expected request to throw ApiError');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      const apiErr = err as ApiError;
      expect(apiErr.code).toBe('UNAUTHORIZED');
      expect(apiErr.status).toBe(401);
      expect(apiErr.details).toBeNull();
    }
  });

	it('rejects ledger-scoped requests before fetch when no ledger is active', async () => {
		const fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);

		await expect(request('/api/transactions')).rejects.toMatchObject({
			code: 'LEDGER_REQUIRED',
			status: 400,
		});
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('sends the active ledger for ledger-scoped requests', async () => {
		useLedgerStore.getState().setActiveLedger('ledger-active', 'owner');
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ success: true, data: null }),
		}));

		await request('/api/transactions');
		expect(fetch).toHaveBeenCalledWith('/api/transactions', expect.objectContaining({
			headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-active' }),
		}));
	});

	it('never sends a ledger header for global requests', async () => {
		useLedgerStore.getState().setActiveLedger('ledger-active', 'owner');
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ success: true, data: null }),
		}));

		await request('/api/auth/me', { ledgerScope: 'none' });
		const options = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
		expect(options.headers).not.toHaveProperty('X-Ledger-Id');
	});

	it('uses the path ledger as the explicit header for ledger member routes', async () => {
		useLedgerStore.getState().setActiveLedger('ledger-active', 'owner');
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ success: true, data: null }),
		}));

		await request('/api/ledgers/ledger-path/members', { ledgerId: 'ledger-path' });
		expect(fetch).toHaveBeenCalledWith('/api/ledgers/ledger-path/members', expect.objectContaining({
			headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-path' }),
		}));
	});

	it('uses the temporary archived viewing ledger without changing the active preference', async () => {
		useLedgerStore.getState().setActiveLedger('ledger-active', 'owner');
		useLedgerStore.getState().enterArchivedLedgerView({
			id: 'ledger-archived',
			name: '历史账本',
			role: 'viewer',
			status: 'archived',
			version: 2,
			member_count: 2,
			archived_at: '2026-07-15T00:00:00Z',
			archived_by_user_id: 'user-owner',
			created_at: '2026-07-01T00:00:00Z',
			updated_at: '2026-07-15T00:00:00Z',
		});
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ success: true, data: null }),
		}));

		await request('/api/transactions');
		expect(fetch).toHaveBeenCalledWith('/api/transactions', expect.objectContaining({
			headers: expect.objectContaining({ 'X-Ledger-Id': 'ledger-archived' }),
		}));
		expect(useLedgerStore.getState().activeLedgerId).toBe('ledger-active');
		expect(useLedgerStore.getState().recentLedgerUsedAt).toHaveProperty('ledger-active');
		expect(useLedgerStore.getState().recentLedgerUsedAt).not.toHaveProperty('ledger-archived');
	});

	it('replaces caller-provided ledger headers with the resolved ledger scope', async () => {
		useLedgerStore.getState().setActiveLedger('ledger-active', 'owner');
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ success: true, data: null }),
		}));

		await request('/api/ledgers/ledger-path/members', {
			ledgerId: 'ledger-path',
			headers: { 'x-ledger-id': 'ledger-wrong' },
		});
		const options = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
		expect(options.headers).toEqual(expect.objectContaining({ 'X-Ledger-Id': 'ledger-path' }));
		expect(options.headers).not.toHaveProperty('x-ledger-id');
	});
});
