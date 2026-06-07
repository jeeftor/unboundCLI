import type {
  ApplyResponse,
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

export const api = {
  config: () => getJSON<ConfigResponse>('/api/config'),
  entries: () => getJSON<EntriesResponse>('/api/entries'),
  planSync: (service: string, hostname = '') => {
    const query = new URLSearchParams();
    query.set('service', service);
    if (hostname) query.set('hostname', hostname);
    return getJSON<PlanResponse>(`/api/sync/plan?${query.toString()}`);
  },
  applySync: (payload: { dry_run: boolean; actions?: SyncAction[]; plan_id?: string; action_ids?: string[] }) =>
    postJSON<ApplyResponse>('/api/sync/apply', payload),
  saveConfig: (payload: unknown) => postJSON<ConfigResponse>('/api/config', payload),
  testConfig: (service: ServiceKey) => postJSON<ConfigTestResponse>('/api/config/test', { service })
};
