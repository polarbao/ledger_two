import { describe, it, expect, vi, beforeEach } from 'vitest';
import { request, ApiError } from './client';

describe('API Client Error Parsing', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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

    const data = await request<{ id: string; name: string }>('/api/test');
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
      await request('/api/test');
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
      await request('/api/test');
      expect.fail('Expected request to throw ApiError');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      const apiErr = err as ApiError;
      expect(apiErr.code).toBe('UNAUTHORIZED');
      expect(apiErr.status).toBe(401);
      expect(apiErr.details).toBeNull();
    }
  });
});
