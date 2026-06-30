import type {
  ApplyResponse,
  CaddyEntriesResponse,
  CaddyDiffResponse,
  CaddyPreviewResponse,
  CaddyTemplatesResponse,
  CaddyValidateResult,
  ConfigResponse,
  ConfigTestResponse,
  EntriesResponse,
  PlanResponse,
  ServiceKey,
  SyncAction
} from '../types';

export class ApiError extends Error {
  constructor(message: string, public readonly status: number) {
    super(message);
    this.name = 'ApiError';
  }
}

async function readJSON<T>(response: Response): Promise<T> {
  const data = await response.json();
  if (!response.ok) {
    const message = typeof data?.error === 'string' ? data.error : response.statusText;
    throw new ApiError(message, response.status);
  }
  return data as T;
}

export async function getJSON<T>(path: string): Promise<T> {
  return readJSON<T>(await fetch(path));
}

export async function postJSON<T>(path: string, payload: unknown): Promise<T> {
  return readJSON<T>(await fetch(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    },
    body: JSON.stringify(payload)
  }));
}

export async function putJSON<T>(path: string, payload: unknown): Promise<T> {
  return readJSON<T>(await fetch(path, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    },
    body: JSON.stringify(payload)
  }));
}

export async function deleteJSON<T>(path: string): Promise<T> {
  return readJSON<T>(await fetch(path, {
    method: 'DELETE',
    headers: {
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    }
  }));
}

export const api = {
  config: () => getJSON<ConfigResponse>('/api/config'),
  entries: () => getJSON<EntriesResponse>('/api/entries'),
  logs: (since: number) => getJSON<{ lines: Array<{ index: number; level: string; message: string; time: string }>; cursor: number }>(`/api/logs?since=${since}`),
  planSync: (service: string, hostname = '') => {
    const query = new URLSearchParams();
    query.set('service', service);
    if (hostname) query.set('hostname', hostname);
    return getJSON<PlanResponse>(`/api/sync/plan?${query.toString()}`);
  },
  applySync: (payload: { dry_run: boolean; actions?: SyncAction[]; plan_id?: string; action_ids?: string[] }) =>
    postJSON<ApplyResponse>('/api/sync/apply', payload),
  removeEntry: (hostname: string, service: 'all' | 'unbound' | 'adguard' = 'all') =>
    postJSON<{ removed: number; message: string }>('/api/sync/remove', { hostname, service }),
  saveConfig: (payload: unknown) => postJSON<ConfigResponse>('/api/config', payload),
  testConfig: (service: ServiceKey) => postJSON<ConfigTestResponse>('/api/config/test', { service }),

  // Caddy Editor
  caddyEntries: () => getJSON<CaddyEntriesResponse>('/api/caddy/entries'),
  caddyCreateEntry: (payload: { hostname: string; upstream: string; template: string; commit_message?: string }) =>
    postJSON<{ status: string }>('/api/caddy/entries', payload),
  caddyUpdateEntry: (hostname: string, payload: { upstream: string; template: string; options?: Record<string, boolean>; commit_message?: string }) =>
    putJSON<{ status: string }>(`/api/caddy/entries/${encodeURIComponent(hostname)}`, payload),
  caddyDeleteEntry: (hostname: string) =>
    deleteJSON<{ status: string }>(`/api/caddy/entries/${encodeURIComponent(hostname)}`),
  caddyDiff: () => getJSON<CaddyDiffResponse>('/api/caddy/diff'),
  caddyGitStatus: () => getJSON<{ remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string }>('/api/caddy/git/status'),
  caddyGitPull: () => postJSON<{ output: string; status: string }>('/api/caddy/git/pull', {}),
  caddyValidate: () => postJSON<CaddyValidateResult>('/api/caddy/validate', {}),
  caddyValidateDraft: (payload: { hostname: string; upstream: string; template: string }) =>
    postJSON<CaddyValidateResult>('/api/caddy/validate-draft', payload),
  caddyTemplates: () => getJSON<CaddyTemplatesResponse>('/api/caddy/templates'),
  caddyPreview: (hostname: string, upstream: string, template: string) => {
    const q = new URLSearchParams({ hostname, upstream, template });
    return getJSON<CaddyPreviewResponse>(`/api/caddy/preview?${q.toString()}`);
  },

  // Cloudflare
  cfSetRoute: (payload: { hostname: string; service: string; http_host_header?: string; no_tls_verify?: boolean }) =>
    postJSON<{ status: string }>('/api/cloudflare/set-route', payload),
  cfRemoveRoute: (hostname: string) =>
    postJSON<{ status: string }>('/api/cloudflare/remove-route', { hostname }),
  cfRepairDNS: () =>
    postJSON<{ fixed: string[]; failed: string[] }>('/api/cloudflare/repair-dns', {}),

  // DNS probe via Cloudflare's public resolver (1.1.1.1)
  dnsProbe: (hostname: string) => getJSON<{ resolved: boolean; cname?: string; addresses?: string[]; error?: string }>(`/api/dns-probe?hostname=${encodeURIComponent(hostname)}`),
};
