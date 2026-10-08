import type {
  Message,
  Project,
  RequestLog,
  StatusEvent,
  Verification,
  WebhookDelivery,
} from "../types";

const API_BASE = "/api/v1";

function buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
  const url = new URL(`${API_BASE}${path}`, window.location.origin);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null && value !== "") {
        url.searchParams.append(key, String(value));
      }
    }
  }
  return url.toString();
}

async function request<T>(
  path: string,
  options: RequestInit & { params?: Record<string, string | number | undefined> } = {}
): Promise<T> {
  const { params, headers, ...init } = options;
  const url = buildUrl(path, params);

  const response = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      "X-Mocksms": "true",
      ...headers,
    },
    credentials: "include",
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({
      error: { message: response.statusText, code: "request_failed" },
    }));
    const message = errorData.error?.message || errorData.message || response.statusText;
    throw new Error(message);
  }

  if (response.status === 204) {
    return undefined as unknown as T;
  }

  return response.json();
}

export interface ListMessagesParams {
  channel?: string;
  to?: string;
  from?: string;
  status?: string;
  batch_id?: string;
  direction?: string;
  since?: string;
  limit?: number;
  cursor?: string;
  project?: string;
}

export interface SendSMSInput {
  to: string;
  from?: string;
  body: string;
  callback_url?: string;
  project?: string;
}

export interface SendEmailInput {
  to: string;
  from: string;
  subject: string;
  text?: string;
  html?: string;
  headers?: Record<string, string>;
  project?: string;
}

export interface SimulateInboundInput {
  to: string;
  from: string;
  body: string;
}

// Message API
export const messagesApi = {
  list: (params?: ListMessagesParams) =>
    request<{ messages: Message[]; next_cursor: string | null }>("/messages", {
      method: "GET",
      params: params as Record<string, string | number | undefined>,
    }),

  get: (id: string) =>
    request<{ message: Message; status_events: StatusEvent[] }>(`/messages/${id}`),

  getRaw: (id: string) =>
    fetch(`${API_BASE}/messages/${id}/raw`, {
      credentials: "include",
      headers: { "X-Mocksms": "true" },
    }).then((r) => r.blob()),

  sendSMS: (body: SendSMSInput) =>
    request<Message>("/sms", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  sendEmail: (body: SendEmailInput) =>
    request<Message>("/email", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  simulateInbound: (body: SimulateInboundInput) =>
    request<Message>("/inbound", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  delete: (projectId?: string) =>
    request<Record<string, unknown>>(`/messages${projectId ? `?project=${projectId}` : ""}`, {
      method: "DELETE",
    }),

  wait: (params: { to?: string; channel?: string; since?: string; timeout?: number }) =>
    request<{ message: Message }>("/messages/wait", {
      method: "GET",
      params,
    }),
};

// Verification API
export const verificationsApi = {
  list: (params?: { limit?: number; cursor?: string; project?: string }) =>
    request<{ verifications: Verification[]; next_cursor: string | null }>("/verifications", {
      method: "GET",
      params,
    }),

  create: (body: {
    to: string;
    channel: "sms" | "email";
    code_length?: number;
    ttl_seconds?: number;
    max_attempts?: number;
  }) =>
    request<{ verification: Verification; message: Message }>("/verifications", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  get: (id: string, params?: { project?: string }) =>
    request<Verification>(`/verifications/${id}`, {
      method: "GET",
      params,
    }),

  check: (id: string, code: string) =>
    request<{ valid: boolean; status: string }>(`/verifications/${id}/check`, {
      method: "POST",
      body: JSON.stringify({ code }),
    }),

  expire: (id: string) =>
    request<Record<string, unknown>>(`/verifications/${id}/expire`, {
      method: "POST",
      body: JSON.stringify({}),
    }),
};

// Projects API
export const projectsApi = {
  list: (params?: { limit?: number; cursor?: string }) =>
    request<{ projects: Project[]; next_cursor: string | null }>("/projects", {
      method: "GET",
      params,
    }),

  get: (id: string) => request<Project>(`/projects/${id}`),

  update: (id: string, body: { name?: string; settings?: Record<string, unknown> }) =>
    request<Project>(`/projects/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),

  linkCredential: (id: string, body: { provider: string; key: string }) =>
    request<{ message: string }>(`/projects/${id}/credentials`, {
      method: "POST",
      body: JSON.stringify(body),
    }),
};

// Request logs & Webhook inspector API
export const inspectorApi = {
  listRequests: (params?: { limit?: number; cursor?: string }) =>
    request<{ logs: RequestLog[]; next_cursor: string | null }>("/requests", {
      method: "GET",
      params,
    }),

  getRequest: (id: string) => request<RequestLog>(`/requests/${id}`),

  listWebhooks: (params?: { limit?: number; cursor?: string }) =>
    request<{ deliveries: WebhookDelivery[]; next_cursor: string | null }>("/webhooks", {
      method: "GET",
      params,
    }),
};

// Health & System
export const healthApi = {
  check: () => request<{ status: string; version: string }>("/healthz"),
};
