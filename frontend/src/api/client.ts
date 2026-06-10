export class ApiError extends Error {
  code: string;
  status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
    this.name = 'ApiError';
  }
}

interface ApiResponse<T = unknown> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
  };
}

export async function request<T>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  const mergedOptions: RequestInit = {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
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

    if (response.status === 401 || errCode === 'UNAUTHORIZED') {
      if (!window.location.pathname.endsWith('/login') && !window.location.pathname.endsWith('/init')) {
        window.location.href = '/login';
      }
    }
    throw new ApiError(errCode, errMessage, response.status);
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
      body: body ? JSON.stringify(body) : undefined,
    }),
  put: <T>(url: string, body?: unknown, options?: RequestInit) =>
    request<T>(url, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    }),
  patch: <T>(url: string, body?: unknown, options?: RequestInit) =>
    request<T>(url, {
      ...options,
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(url: string, options?: RequestInit) =>
    request<T>(url, { ...options, method: 'DELETE' }),
};
