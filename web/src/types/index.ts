export type Channel = "sms" | "email";
export type Direction = "outbound" | "inbound";
export type MessageStatus = "queued" | "sent" | "delivered" | "undelivered" | "failed" | "received";

export type VerificationStatus = "pending" | "approved" | "canceled" | "expired" | "max_attempts";

export interface Message {
  id: string;
  project_id: string;
  batch_id?: string;
  channel: Channel;
  direction: Direction;
  provider: string;
  provider_ref: string;
  from: string;
  to: string;
  cc?: string[];
  bcc?: string[];
  subject?: string;
  body_text?: string;
  body_html?: string;
  raw_blob_id?: string;
  encoding?: string;
  segments: number;
  status: MessageStatus;
  error_code?: string;
  error_message?: string;
  callback_url?: string;
  extracted_codes?: string[];
  extracted_links?: string[];
  primary_link?: string;
  created_at: string;
  updated_at: string;
}

export interface StatusEvent {
  id: string;
  message_id: string;
  status: MessageStatus;
  error_code?: string;
  at: string;
}

export interface Verification {
  id: string;
  project_id: string;
  provider: string;
  provider_ref: string;
  service_ref?: string;
  to: string;
  channel: Channel;
  code: string;
  status: VerificationStatus;
  attempts: number;
  max_attempts: number;
  expires_at: string;
  message_id?: string;
  created_at: string;
}

export interface BatchRejectedRecipient {
  to: string;
  code: string;
  message: string;
}

export interface Batch {
  id: string;
  project_id: string;
  provider: string;
  channel: Channel;
  total: number;
  counts: Record<string, number>;
  rejected?: BatchRejectedRecipient[];
  created_at: string;
}

export interface RequestLog {
  id: string;
  project_id: string;
  adapter: string;
  method: string;
  path: string;
  request_headers: Record<string, string>;
  request_body?: string;
  response_status: number;
  response_body?: string;
  duration_ms: number;
  created_at: string;
}

export interface WebhookDelivery {
  id: string;
  project_id: string;
  message_id?: string;
  verification_id?: string;
  kind: string;
  url: string;
  payload: Record<string, unknown>;
  headers: Record<string, string>;
  provider?: string;
  attempt: number;
  status: "pending" | "succeeded" | "failed";
  response_status?: number;
  response_body?: string;
  next_retry_at?: string;
  note?: string;
  created_at: string;
}

export interface SimConfig {
  latency_ms?: number;
  failure_rate?: number;
}

export interface CountryEstimate {
  country: string;
  messages: number;
  unit_price: number;
  cost: number;
}

export interface MissingPrice {
  country: string;
  messages: number;
}

export interface ProviderEstimate {
  provider: string;
  currency: string;
  as_of: string;
  estimate_only: boolean;
  monthly_messages: number;
  total_cost: number;
  by_country: CountryEstimate[];
  monthly_verifications: number;
  verify_cost: number;
  no_price_data: MissingPrice[];
}

export interface EstimateResponse {
  volume: number;
  observed_messages: number;
  observed_verifications: number;
  providers: ProviderEstimate[];
}

export interface Project {
  id: string;
  name: string;
  settings?: Record<string, unknown>;
  created_at: string;
}
