const API_BASE = '/api/v1'

function buildUrl(path: string, params?: Record<string, string>) {
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.append(key, value)
    })
  }
  return url.toString()
}

async function request<T>(path: string, options: RequestInit & { params?: Record<string, string> } = {}): Promise<T> {
  const { params, headers, ...init } = options
  const url = buildUrl(path, params)

  const response = await fetch(url, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-Mocksms': 'true',
      ...headers,
    },
    credentials: 'include',
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: { message: response.statusText } }))
    throw new Error(error.error?.message || response.statusText)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json()
}

async function requestWithParams<T>(path: string, options: RequestInit & { params?: Record<string, string | number> } = {}): Promise<T> {
  const { params, headers, ...init } = options
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.append(key, String(value))
    })
  }

  const response = await fetch(url.toString(), {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-Mocksms': 'true',
      ...headers,
    },
    credentials: 'include',
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: { message: response.statusText } }))
    throw new Error(error.error?.message || response.statusText)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json()
}

// Message API
export const messagesApi = {
  list: (params?: { channel?: string; to?: string; from?: string; status?: string; batch_id?: string; direction?: string; since?: string; limit?: number; cursor?: string }) =>
    requestWithParams<{ messages: any[]; next_cursor: string | null }>('/messages', { method: 'GET', params }),

  get: (id: string) =>
    request<{ message: any; status_events: any[] }>(`/messages/${id}`),

  getRaw: (id: string) =>
    fetch(`${API_BASE}/messages/${id}/raw`, { credentials: 'include' }).then(r => r.blob()),

  delete: (projectId?: string) =>
    request<{}>(`/messages${projectId ? `?project=${projectId}` : ''}`, { method: 'DELETE' }),

  wait: (params: { to?: string; channel?: string; since?: string; timeout?: number }) =>
    requestWithParams<{ message: any }>('/messages/wait', { method: 'GET', params }),
}

// Verification API
export const verificationsApi = {
  list: (params?: { limit?: number; cursor?: string; project?: string }) =>
    requestWithParams<{ verifications: any[]; next_cursor: string | null }>('/verifications', { method: 'GET', params }),

  create: (body: { to: string; channel: 'sms' | 'email'; code_length?: number; ttl_seconds?: number; max_attempts?: number }) =>
    request<{ verification: any; message: any }>('/verifications', { method: 'POST', body: JSON.stringify(body) }),

  get: (id: string, params?: { project?: string }) =>
    requestWithParams<any>(`/verifications/${id}`, { method: 'GET', params }),

  check: (id: string, code: string) =>
    request<{ valid: boolean; status: string }>(`/verifications/${id}/check`, { method: 'POST', body: JSON.stringify({ code }) }),

  expire: (id: string) =>
    request<{ }>(`/verifications/${id}/expire`, { method: 'POST', body: JSON.stringify({}) }),
}

// Projects API
export const projectsApi = {
  list: (params?: { limit?: number; cursor?: string }) =>
    requestWithParams<{ projects: any[]; next_cursor: string | null }>('/projects', { method: 'GET', params }),

  get: (id: string) =>
    request<any>(`/projects/${id}`),

  update: (id: string, body: { name?: string; settings?: any }) =>
    requestWithParams<any>(`/projects/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),

  linkCredential: (id: string, body: { provider: string; key: string }) =>
    request<{ message: string }>(`/projects/${id}/credentials`, { method: 'POST', body: JSON.stringify(body) }),
}

// Health
export const healthApi = {
  check: () => request<{ status: string; version: string }>('/healthz'),
}