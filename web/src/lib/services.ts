import { Activity, Cloud, Database, HardDrive, Network, ShieldCheck } from 'lucide-react';
import type { ComponentType } from 'react';
import type { ConfigSource, Entry, ServiceKey, SyncAction } from '../types';

export const serviceOrder: ServiceKey[] = ['caddy', 'unbound', 'adguard', 'dhcp', 'cloudflare'];

export const serviceMeta: Record<ServiceKey, { label: string; shortLabel: string; icon: ComponentType<{ size?: number }>; tone: string }> = {
  caddy: { label: 'Caddy', shortLabel: 'Caddy', icon: Network, tone: 'green' },
  unbound: { label: 'OPNSense / Unbound', shortLabel: 'Unbound', icon: Database, tone: 'blue' },
  adguard: { label: 'AdGuard', shortLabel: 'AdGuard', icon: ShieldCheck, tone: 'teal' },
  dhcp: { label: 'DHCP / DNSMasq', shortLabel: 'DHCP', icon: HardDrive, tone: 'yellow' },
  cloudflare: { label: 'Cloudflare', shortLabel: 'Cloudflare', icon: Cloud, tone: 'violet' }
};

export const statusFilterOptions: [string, string][] = [
  ['all', 'All status'],
  ['out_of_sync', 'Out of sync'],
  ['caddy_only', 'Caddy only'],
  ['stale', 'Stale DNS'],
  ['cloudflare', 'Cloudflare routed']
];

export const serviceFilterOptions: [string, string][] = [
  ['all', 'All services'],
  ['unbound', 'Unbound'],
  ['adguard', 'AdGuard'],
  ['dhcp', 'DHCP'],
  ['cloudflare', 'Cloudflare']
];

export function serviceEnabled(enabled: Partial<Record<ServiceKey, boolean>>, service: string) {
  if (service === 'all') return enabled.unbound !== false || enabled.adguard !== false;
  if (service === 'dhcp') return true;
  if (service === 'unbound' || service === 'adguard' || service === 'cloudflare') return enabled[service] !== false;
  return true;
}

export function syncOptions(enabled: Partial<Record<ServiceKey, boolean>>): [string, string][] {
  const allLabel = enabled.unbound !== false && enabled.adguard !== false
    ? 'All DNS targets'
    : 'Available DNS targets';
  return [
    ['all', allLabel],
    ['unbound', 'Unbound'],
    ['adguard', 'AdGuard'],
    ['dhcp', 'DHCP preview']
  ];
}

export function disabledSyncValues(enabled: Partial<Record<ServiceKey, boolean>>) {
  const disabled = new Set<string>();
  if (enabled.unbound === false && enabled.adguard === false) disabled.add('all');
  if (enabled.unbound === false) disabled.add('unbound');
  if (enabled.adguard === false) disabled.add('adguard');
  return disabled;
}

export function statusClassByCode(status: number) {
  if (status === 0 || status === 1) return 'ok';
  if (status === 2 || status === 4 || status === 5) return 'bad';
  return 'warn';
}

export function dnsResultClass(value: string) {
  const normalized = String(value || '').toLowerCase();
  return normalized && normalized !== 'fail' ? 'ok' : 'bad';
}

export function serviceStateText(status: { configured: boolean; in_sync: boolean; ip: string }) {
  if (!status?.configured) return 'Missing';
  if (status.in_sync) return status.ip ? `In sync (${status.ip})` : 'In sync';
  return status.ip ? `Mismatch (${status.ip})` : 'Mismatch';
}

export function cloudflareStateText(status: Entry['cloudflare_status']) {
  if (!status?.configured) return 'Not routed';
  if (status.http_host_header) return status.service || 'Routed';
  return 'Missing host header';
}

export function formatConfigKey(key: string) {
  return key.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase())
    .replace(/\bApi\b/g, 'API')
    .replace(/\bUrl\b/g, 'URL')
    .replace(/\bTls\b/g, 'TLS')
    .replace(/\bDhcp\b/g, 'DHCP')
    .replace(/\bId\b/g, 'ID');
}

export function formatTestDetails(details: Record<string, string>) {
  return Object.entries(details)
    .filter(([, value]) => value !== '')
    .map(([key, value]) => `${formatConfigKey(key)}: ${value}`)
    .join(', ');
}

export function sourceLabel(source?: ConfigSource) {
  const label = source?.label || source?.kind || 'Unknown';
  return source?.path ? `${label}: ${source.path}` : label;
}

export function compactSourceKind(source?: ConfigSource) {
  if (!source?.kind) return 'unknown';
  if (source.kind === 'config-file') return 'config file';
  return source.kind.replace(/-/g, ' ');
}

export function renderActionLine(action: SyncAction) {
  const target = action.new_ip ? ` -> ${action.new_ip}` : '';
  return `${action.type.toUpperCase()} ${action.service} ${action.hostname}${target}`;
}

export function serviceIcon(service: ServiceKey) {
  return serviceMeta[service]?.icon || Activity;
}
