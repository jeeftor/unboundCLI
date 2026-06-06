export type ServiceKey = 'caddy' | 'unbound' | 'adguard' | 'dhcp' | 'cloudflare';

export type ConfigSource = {
  kind: string;
  label: string;
  path?: string;
};

export type ConfigServiceSummary = {
  label: string;
  enabled: boolean;
  client_ready: boolean;
  source: ConfigSource;
  endpoint?: string;
  insecure?: boolean;
  fields?: Record<string, boolean>;
  details?: Record<string, string>;
  missing?: string[];
};

export type ConfigResponse = {
  caddy: {
    server_ip: string;
    server_port: number;
  };
  enabled: Record<ServiceKey, boolean>;
  mutation_enabled: boolean;
  save_target: string;
  summary: Record<ServiceKey, ConfigServiceSummary>;
};

export type ServiceStatus = {
  configured: boolean;
  ip: string;
  in_sync: boolean;
};

export type DHCPStatus = {
  configured: boolean;
  type: string;
  ip: string;
  mac: string;
  hostname: string;
  in_sync: boolean;
};

export type CloudflareStatus = {
  configured: boolean;
  tunnel_name: string;
  tunnel_id: string;
  service: string;
  path: string;
  is_default_tunnel: boolean;
  http_host_header: string;
  no_tls_verify: boolean;
  http2_origin: boolean;
  has_access_policy: boolean;
};

export type Entry = {
  hostname: string;
  caddy_upstream: string;
  caddy_ip: string;
  caddy_port: string;
  unbound_status: ServiceStatus;
  adguard_status: ServiceStatus;
  dhcp_status: DHCPStatus;
  dns_resolved: string;
  cloudflare_status: CloudflareStatus;
  overall_status: number;
  status_label: string;
  data_source: string;
};

export type EntriesResponse = {
  entries: Entry[];
  report: {
    services?: Record<string, { count?: number; status?: string; error?: string }>;
  };
};

export type SyncAction = {
  type: string;
  service: string;
  hostname: string;
  new_ip?: string;
  enabled?: boolean;
};

export type PlanResponse = {
  plan_id: string;
  action_ids: string[];
  actions: SyncAction[];
};

export type ApplyResponse = {
  result: {
    success: boolean;
    message: string;
    items_added: number;
    items_updated: number;
    items_deleted: number;
  };
};

export type ConfigTestResponse = {
  service: string;
  success: boolean;
  message: string;
  details?: Record<string, string>;
};

export type WebConfig = {
  applyToken?: string;
  mutationEnabled?: boolean;
};

declare global {
  interface Window {
    UNBOUNDCLI_WEB_CONFIG?: WebConfig;
    UNBOUNDCLI_TEST_HOOKS?: boolean;
  }
}
