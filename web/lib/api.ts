const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface RelayEndpoint {
  slug: string;
  url: string;
  created_at: string;
}

export interface RelayRequest {
  id: string;
  slug: string;
  method: string;
  path: string;
  headers: Record<string, string[]>;
  query_params: Record<string, string[]>;
  body: string; // base64 encoded
  body_size: number;
  content_type: string;
  received_at: string;
}

export interface DeliveryAttempt {
  id: string;
  request_id: string;
  slug: string;
  delivered: boolean;
  status_code: number;
  error: string;
  is_replay: boolean;
  attempted_at: string;
  response_body?: string;        // base64, only present on live WS events
  response_headers?: Record<string, string[]>;
}

export async function createEndpoint(): Promise<RelayEndpoint> {
  const res = await fetch(`${API}/endpoint`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error("Failed to create endpoint");
  return res.json();
}

export async function listRequests(slug: string): Promise<RelayRequest[]> {
  const res = await fetch(`${API}/request/${slug}`, { cache: "no-store" });
  if (!res.ok) throw new Error("Failed to fetch requests");
  return res.json();
}

export async function getRequest(
  slug: string,
  id: string
): Promise<RelayRequest> {
  const res = await fetch(`${API}/request/${slug}/${id}`, {
    cache: "no-store",
  });
  if (!res.ok) throw new Error("Failed to fetch request");
  return res.json();
}

export async function replayRequest(
  slug: string,
  id: string
): Promise<void> {
  const res = await fetch(`${API}/request/${slug}/${id}/replay`, {
    method: "POST",
  });
  if (!res.ok) throw new Error("Failed to replay request");
}

export async function listAttempts(slug: string): Promise<DeliveryAttempt[]> {
  const res = await fetch(`${API}/request/${slug}/attempts`, { cache: "no-store" });
  if (!res.ok) throw new Error("Failed to fetch attempts");
  return res.json();
}

export function getWebSocketUrl(slug: string): string {
  const wsBase = API.replace(/^http:/, "ws:").replace(/^https:/, "wss:");
  return `${wsBase}/ws/browser/${slug}`;
}
