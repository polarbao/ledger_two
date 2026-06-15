import { useLedgerStore } from '../stores/ledger.store';

export class ApiError extends Error {
  code: string;
  status: number;
  details: unknown;

  constructor(code: string, message: string, status: number, details?: unknown) {
    super(message);
    this.code = code;
    this.status = status;
    this.details = details;
    this.name = 'ApiError';
  }
}

interface ApiResponse<T = unknown> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
    details?: unknown;
  };
}

export async function request<T>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const isFormData = options.body instanceof FormData;
  const headers: Record<string, string> = {};
  if (!isFormData) {
    headers['Content-Type'] = 'application/json';
  }
  if (options.headers) {
    Object.assign(headers, options.headers);
  }

  const { activeLedgerId } = useLedgerStore.getState();
  if (activeLedgerId) {
    headers['X-Ledger-Id'] = activeLedgerId;
  }

  const mergedOptions: RequestInit = {
    ...options,
    credentials: 'include',
    headers,
  };

  const response = await fetch(url, mergedOptions);

  let body: ApiResponse<T>;
  try {
    body = await response.json() as ApiResponse<T>;
  } catch {
    throw new ApiError('RESPONSE_PARSE_ERROR', '解析服务器响应失败', response.status);
  }

  if (!response.ok || !body.success) {
    const errCode = body?.error?.code || 'UNKNOWN_ERROR';
    const errMessage = body?.error?.message || '请求执行失败';
    const errDetails = body?.error?.details || null;

    if (response.status === 401 || errCode === 'UNAUTHORIZED' || errCode === 'SESSION_EXPIRED') {
      if (typeof window !== 'undefined' && window.location) {
        if (!window.location.pathname.endsWith('/login') && !window.location.pathname.endsWith('/init')) {
          window.location.href = '/login';
        }
      }
    }
    throw new ApiError(errCode, errMessage, response.status, errDetails);
  }

  return body.data;
}

export const api = {
  get: <T>(url: string, options?: RequestInit) =>
    request<T>(url, { ...options, method: 'GET' }),
  post: <T>(url: string, body?: unknown, options?: RequestInit) =>
    request<T>(url, {
      ...options,
      method: 'POST',
      body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined),
    }),
  put: <T>(url: string, body?: unknown, options?: RequestInit) =>
    request<T>(url, {
      ...options,
      method: 'PUT',
      body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined),
    }),
  patch: <T>(url: string, body?: unknown, options?: RequestInit) =>
    request<T>(url, {
      ...options,
      method: 'PATCH',
      body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined),
    }),
  delete: <T>(url: string, options?: RequestInit) =>
    request<T>(url, { ...options, method: 'DELETE' }),
};
