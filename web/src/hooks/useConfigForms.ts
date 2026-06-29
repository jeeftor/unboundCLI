import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import { formatTestDetails } from '../lib/services';
import type { ConfigResponse, ServiceKey } from '../types';

export type CaddyEditorForm = {
  enabled: boolean;
  repo_path: string;
  caddyfile: string;
  deploy_command: string;
  validate_command: string;
  git_auto_commit: boolean;
  git_auto_push: boolean;
  git_remote: string;
  git_branch: string;
  entry_template: string;
};

export type ConfigForms = {
  unbound: { base_url: string; api_key: string; api_secret: string; insecure: boolean };
  adguard: { enabled: boolean; base_url: string; username: string; password: string; insecure: boolean };
  cloudflare: {
    enabled: boolean;
    api_token: string;
    account_id: string;
    zone_id: string;
    tunnel_id: string;
    caddy_service_url: string;
    insecure: boolean;
  };
  caddyEditor: CaddyEditorForm;
};

export type TestResults = Partial<Record<ServiceKey, { text: string; kind: 'info' | 'ok' | 'error' }>>;

export const emptyForms: ConfigForms = {
  unbound: { base_url: '', api_key: '', api_secret: '', insecure: false },
  adguard: { enabled: false, base_url: '', username: '', password: '', insecure: false },
  cloudflare: {
    enabled: false,
    api_token: '',
    account_id: '',
    zone_id: '',
    tunnel_id: '',
    caddy_service_url: '',
    insecure: false
  },
  caddyEditor: {
    enabled: false,
    repo_path: '',
    caddyfile: 'caddy/Caddyfile',
    deploy_command: '',
    validate_command: '',
    git_auto_commit: false,
    git_auto_push: false,
    git_remote: 'origin',
    git_branch: '',
    entry_template: 'default'
  }
};

export function useConfigForms(args: {
  config: ConfigResponse | null;
  mutationEnabled: boolean;
  applyConfig: (config: ConfigResponse) => void;
  onConfigChanged: () => void;
}) {
  const [forms, setForms] = useState<ConfigForms>(emptyForms);
  const [savedForms, setSavedForms] = useState<ConfigForms>(emptyForms);
  const [configStatus, setConfigStatus] = useState('');
  const [configStatusKind, setConfigStatusKind] = useState<'info' | 'error' | 'ok'>('info');
  const [testResults, setTestResults] = useState<TestResults>({});
  const formsRef = useRef(forms);

  useEffect(() => {
    formsRef.current = forms;
  }, [forms]);

  useEffect(() => {
    const nextConfig = args.config;
    if (!nextConfig) return;
    const ce = nextConfig.caddy_editor;
    setForms((current) => {
      const next: ConfigForms = {
        unbound: {
          ...current.unbound,
          base_url: nextConfig.summary.unbound?.endpoint || '',
          insecure: Boolean(nextConfig.summary.unbound?.insecure)
        },
        adguard: {
          ...current.adguard,
          enabled: Boolean(nextConfig.summary.adguard?.enabled),
          base_url: nextConfig.summary.adguard?.endpoint || '',
          insecure: Boolean(nextConfig.summary.adguard?.insecure)
        },
        cloudflare: {
          ...current.cloudflare,
          enabled: Boolean(nextConfig.summary.cloudflare?.enabled),
          caddy_service_url: nextConfig.summary.cloudflare?.details?.caddy_service_url || '',
          insecure: Boolean(nextConfig.summary.cloudflare?.insecure)
        },
        caddyEditor: ce ? {
          enabled: ce.enabled,
          repo_path: ce.repo_path || '',
          caddyfile: ce.caddyfile || 'caddy/Caddyfile',
          deploy_command: ce.deploy_command || '',
          validate_command: ce.validate_command || '',
          git_auto_commit: ce.git_auto_commit,
          git_auto_push: ce.git_auto_push,
          git_remote: ce.git_remote || 'origin',
          git_branch: ce.git_branch || '',
          entry_template: ce.entry_template || 'default'
        } : current.caddyEditor
      };
      setSavedForms(next);
      return next;
    });
  }, [args.config]);

  const saveCaddyEditor = useCallback(async (nextForms = formsRef.current) => {
    if (!args.mutationEnabled) {
      setConfigStatus('Config changes are unavailable for this web session.');
      setConfigStatusKind('error');
      return;
    }
    setConfigStatus('Saving Caddy editor config...');
    setConfigStatusKind('info');
    try {
      const ce = nextForms.caddyEditor;
      const payload = {
        caddy_editor: {
          enabled: ce.enabled,
          repo_path: ce.repo_path,
          caddyfile: ce.caddyfile,
          deploy_command: ce.deploy_command,
          validate_command: ce.validate_command,
          git_auto_commit: ce.git_auto_commit,
          git_auto_push: ce.git_auto_push,
          git_remote: ce.git_remote,
          git_branch: ce.git_branch,
          entry_template: ce.entry_template
        }
      };
      const nextConfig = await api.saveConfig(payload);
      args.applyConfig(nextConfig);
      args.onConfigChanged();
      setSavedForms((s) => ({ ...s, caddyEditor: nextForms.caddyEditor }));
      setConfigStatus('Saved Caddy editor config.');
      setConfigStatusKind('ok');
    } catch (err) {
      setConfigStatus(err instanceof Error ? err.message : String(err));
      setConfigStatusKind('error');
    }
  }, [args]);

  const saveConfig = useCallback(async (service: 'unbound' | 'adguard' | 'cloudflare', nextForms = formsRef.current) => {
    if (!args.mutationEnabled) {
      setConfigStatus('Config changes are unavailable for this web session.');
      setConfigStatusKind('error');
      return;
    }
    setConfigStatus(`Saving ${service} config...`);
    setConfigStatusKind('info');
    try {
      const payload = buildConfigUpdate(service, nextForms);
      const nextConfig = await api.saveConfig(payload);
      args.applyConfig(nextConfig);
      args.onConfigChanged();
      setSavedForms((s) => ({ ...s, [service]: nextForms[service] }));
      setConfigStatus(`Saved ${service} config.`);
      setConfigStatusKind('ok');
    } catch (err) {
      setConfigStatus(err instanceof Error ? err.message : String(err));
      setConfigStatusKind('error');
    }
  }, [args]);

  const testConfig = useCallback(async (service: ServiceKey) => {
    if (!args.mutationEnabled) {
      setConfigStatus('Config tests are unavailable for this web session.');
      setConfigStatusKind('error');
      return;
    }
    setConfigStatus(`Testing ${service} config...`);
    setConfigStatusKind('info');
    setTestResults((current) => ({ ...current, [service]: { text: 'Testing connection...', kind: 'info' } }));
    try {
      const result = await api.testConfig(service);
      const detail = formatTestDetails(result.details || {});
      const text = `${result.message}${detail ? ` ${detail}` : ''}`;
      const kind = result.success ? 'ok' : 'error';
      setTestResults((current) => ({ ...current, [service]: { text, kind } }));
      setConfigStatus(result.message);
      setConfigStatusKind(kind);
    } catch (err) {
      const text = err instanceof Error ? err.message : String(err);
      setTestResults((current) => ({ ...current, [service]: { text, kind: 'error' } }));
      setConfigStatus(text);
      setConfigStatusKind('error');
    }
  }, [args.mutationEnabled]);

  return {
    forms,
    setForms,
    savedForms,
    configStatus,
    configStatusKind,
    testResults,
    saveConfig,
    saveCaddyEditor,
    testConfig
  };
}

function buildConfigUpdate(service: 'unbound' | 'adguard' | 'cloudflare', forms: ConfigForms) {
  if (service === 'unbound') {
    return {
      unbound: omitEmptySecretFields({
        base_url: forms.unbound.base_url,
        insecure: forms.unbound.insecure,
        api_key: forms.unbound.api_key,
        api_secret: forms.unbound.api_secret
      }, ['api_key', 'api_secret'])
    };
  }
  if (service === 'adguard') {
    return {
      adguard: omitEmptySecretFields({
        enabled: forms.adguard.enabled,
        base_url: forms.adguard.base_url,
        insecure: forms.adguard.insecure,
        username: forms.adguard.username,
        password: forms.adguard.password
      }, ['username', 'password'])
    };
  }
  return {
    cloudflare: omitEmptySecretFields({
      enabled: forms.cloudflare.enabled,
      caddy_service_url: forms.cloudflare.caddy_service_url,
      insecure: forms.cloudflare.insecure,
      api_token: forms.cloudflare.api_token,
      account_id: forms.cloudflare.account_id,
      zone_id: forms.cloudflare.zone_id,
      tunnel_id: forms.cloudflare.tunnel_id
    }, ['api_token', 'account_id', 'zone_id', 'tunnel_id'])
  };
}

function omitEmptySecretFields<T extends Record<string, string | boolean>>(value: T, secretFields: string[]) {
  const next = { ...value };
  for (const key of secretFields) {
    if (next[key] === '') delete next[key];
  }
  return next;
}
