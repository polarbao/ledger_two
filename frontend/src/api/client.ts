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

export type LedgerScope = 'required' | 'none';

export interface ApiRequestOptions extends RequestInit {
  ledgerScope?: LedgerScope;
  ledgerId?: string;
}

export async function request<T>(
  url: string,
  options: ApiRequestOptions = {}
): Promise<T> {
	const {
		ledgerScope = 'required',
		ledgerId,
		...fetchOptions
	} = options;
	const isFormData = fetchOptions.body instanceof FormData;
  const headers: Record<string, string> = {};
  if (!isFormData) {
    headers['Content-Type'] = 'application/json';
  }
	if (fetchOptions.headers) {
		new Headers(fetchOptions.headers).forEach((value, key) => {
			headers[key] = value;
		});
  }

	for (const key of Object.keys(headers)) {
		if (key.toLowerCase() === 'x-ledger-id') {
			delete headers[key];
		}
	}

	if (ledgerScope !== 'none') {
		const { activeLedgerId, archivedViewingLedger } = useLedgerStore.getState();
		const explicitLedgerId = ledgerId?.trim() || archivedViewingLedger?.id || activeLedgerId;
		if (!explicitLedgerId) {
			throw new ApiError('LEDGER_REQUIRED', '请先选择账本', 400);
		}
		headers['X-Ledger-Id'] = explicitLedgerId;
  }

  const mergedOptions: RequestInit = {
		...fetchOptions,
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
	get: <T>(url: string, options?: ApiRequestOptions) =>
    request<T>(url, { ...options, method: 'GET' }),
	post: <T>(url: string, body?: unknown, options?: ApiRequestOptions) =>
    request<T>(url, {
      ...options,
      method: 'POST',
      body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined),
    }),
	put: <T>(url: string, body?: unknown, options?: ApiRequestOptions) =>
    request<T>(url, {
      ...options,
      method: 'PUT',
      body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined),
    }),
	patch: <T>(url: string, body?: unknown, options?: ApiRequestOptions) =>
    request<T>(url, {
      ...options,
      method: 'PATCH',
      body: body instanceof FormData ? body : (body ? JSON.stringify(body) : undefined),
    }),
	delete: <T>(url: string, options?: ApiRequestOptions) =>
    request<T>(url, { ...options, method: 'DELETE' }),
};
