import type {
  ApplyResponse,
  AuthInventoryResponse,
  CaddyEntriesResponse,
  CaddyDiffResponse,
  CaddyPreviewResponse,
  CaddyTemplatesResponse,
  CaddyValidateResult,
  ConfigResponse,
  ConfigTestResponse,
  EntriesResponse,
  PlanResponse,
	OwnershipAdoptionPreview,
	OperationRecord,
  PruneResponse,
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
  const body = await response.text();
  let data: unknown;
  if (body) {
    try {
      data = JSON.parse(body);
    } catch {
      if (!response.ok) {
        throw new ApiError(`Request failed (${response.status}${response.statusText ? ` ${response.statusText}` : ''})`, response.status);
      }
      throw new Error('Server returned an invalid JSON response');
    }
  }
  if (!response.ok) {
    const message = typeof (data as { error?: unknown } | undefined)?.error === 'string'
      ? (data as { error: string }).error
      : `Request failed (${response.status}${response.statusText ? ` ${response.statusText}` : ''})`;
    throw new ApiError(message, response.status);
  }
  return data as T;
}

export async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  return readJSON<T>(await fetch(path, signal ? { signal } : undefined));
}

export async function postJSON<T>(path: string, payload: unknown, signal?: AbortSignal): Promise<T> {
  return readJSON<T>(await fetch(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    },
    body: JSON.stringify(payload),
    ...(signal ? { signal } : {})
  }));
}

export async function putJSON<T>(path: string, payload: unknown, signal?: AbortSignal): Promise<T> {
  return readJSON<T>(await fetch(path, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    },
    body: JSON.stringify(payload),
    ...(signal ? { signal } : {})
  }));
}

export async function deleteJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  return readJSON<T>(await fetch(path, {
    method: 'DELETE',
    headers: {
      'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken || ''
    },
    ...(signal ? { signal } : {})
  }));
}

export const api = {
  config: (signal?: AbortSignal) => getJSON<ConfigResponse>('/api/config', signal),
  entries: (signal?: AbortSignal) => getJSON<EntriesResponse>('/api/entries', signal),
  logs: (since: number) => getJSON<{ lines: Array<{ index: number; level: string; message: string; time: string }>; cursor: number }>(`/api/logs?since=${since}`),
  planSync: (service: string, hostname = '', unsync = false) => {
    const query = new URLSearchParams();
    query.set('service', service);
    if (hostname) query.set('hostname', hostname);
    if (unsync) query.set('unsync', 'true');
    return getJSON<PlanResponse>(`/api/sync/plan?${query.toString()}`);
  },
  applySync: (payload: { dry_run: boolean; actions?: SyncAction[]; plan_id?: string; action_ids?: string[] }) =>
    postJSON<ApplyResponse>('/api/sync/apply', payload),
  saveConfig: (payload: unknown) => postJSON<ConfigResponse>('/api/config', payload),
  testConfig: (service: ServiceKey) => postJSON<ConfigTestResponse>('/api/config/test', { service }),

  // Caddy Editor
  caddyEntries: () => getJSON<CaddyEntriesResponse>('/api/caddy/entries'),
  caddyCreateEntry: (payload: { hostname: string; upstream: string; template: string; params?: Record<string, string>; commit_message?: string }) =>
    postJSON<{ status: string }>('/api/caddy/entries', payload),
  caddyUpdateEntry: (hostname: string, payload: { upstream: string; template: string; options?: Record<string, boolean>; params?: Record<string, string>; commit_message?: string }) =>
    putJSON<{ status: string }>(`/api/caddy/entries/${encodeURIComponent(hostname)}`, payload),
  caddyDeleteEntry: (hostname: string) =>
    deleteJSON<{ status: string }>(`/api/caddy/entries/${encodeURIComponent(hostname)}`),
  caddyDiff: () => getJSON<CaddyDiffResponse>('/api/caddy/diff'),
  caddyGitStatus: () => getJSON<{ remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string }>('/api/caddy/git/status'),
  caddyGitPull: () => postJSON<{ output: string; status: string }>('/api/caddy/git/pull', {}),
  caddyValidate: () => postJSON<CaddyValidateResult>('/api/caddy/validate', {}),
  caddyValidateDraft: (payload: { hostname: string; upstream: string; template: string; params?: Record<string, string> }) =>
    postJSON<CaddyValidateResult>('/api/caddy/validate-draft', payload),
  caddyTemplates: () => getJSON<CaddyTemplatesResponse>('/api/caddy/templates'),
  caddyPreview: (hostname: string, upstream: string, template: string, params?: Record<string, string>) => {
    const q = new URLSearchParams({ hostname, upstream, template });
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v) q.set(`param_${k}`, v);
      }
    }
    return getJSON<CaddyPreviewResponse>(`/api/caddy/preview?${q.toString()}`);
  },

  // DNS probe via Cloudflare's public resolver (1.1.1.1)
  dnsProbe: (hostname: string) => getJSON<{ resolved: boolean; cname?: string; addresses?: string[]; error?: string }>(`/api/dns-probe?hostname=${encodeURIComponent(hostname)}`),

  // Auth inventory
  authInventory: (signal?: AbortSignal) => getJSON<AuthInventoryResponse>('/api/auth/inventory', signal),

  // Auth fix
  authFixDoubleLogin: (hostname: string) =>
    postJSON<{ status: string }>('/api/auth/fix-double-login', { hostname }),

  // Diagnostics
  diagnosticsPrune: (payload: { dry_run: boolean; hostname?: string; hostnames?: string[] }) =>
    postJSON<PruneResponse>('/api/diagnostics/prune', payload),

  ownershipPreview: (provider: 'adguard' | 'cloudflare') =>
    getJSON<OwnershipAdoptionPreview>(`/api/ownership/adoption?provider=${provider}`),
  ownershipAdopt: (previewID: string, ids: string[]) =>
    postJSON<{ adopted: number }>('/api/ownership/adoption', { preview_id: previewID, ids }),
  operations: () => getJSON<OperationRecord[]>('/api/operations'),
};
