import '../styles/ConfigModal.css';
import {
  ChevronDown,
  FileCode2,
  FileSliders
} from 'lucide-react';
import { useEffect, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { api } from '../api/client';
import type { CaddyEditorForm, ConfigForms, TestResults } from '../hooks/useConfigForms';
import {
  compactSourceKind,
  formatConfigKey,
  serviceMeta,
  sourceLabel
} from '../lib/services';
import { Field } from './InlineProgress';
import type { ConfigResponse, ConfigServiceSummary, ServiceKey } from '../types';

type ConfigTab = ServiceKey | 'caddy-editor';

const configTabLabels: Record<ConfigTab, string> = {
  caddy:        'Caddy',
  unbound:      'Unbound',
  adguard:      'AdGuard',
  dhcp:         'DHCP',
  cloudflare:   'Cloudflare',
  'caddy-editor': 'File Editor',
};

const configTabOrder: ConfigTab[] = ['caddy', 'unbound', 'adguard', 'dhcp', 'cloudflare', 'caddy-editor'];

export function ConfigModal({
  open,
  onClose,
  config,
  forms,
  setForms,
  savedForms,
  mutationEnabled,
  status,
  statusKind,
  testResults,
  onSave,
  onSaveCaddyEditor,
  onTest
}: {
  open: boolean;
  onClose: () => void;
  config: ConfigResponse | null;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  status: string;
  statusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onSaveCaddyEditor: () => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  return (
    <div id="config-panel" className={`config-modal ${open ? 'open' : ''}`} hidden={!open} role="dialog" aria-modal="true" aria-labelledby="config-modal-title">
      <div className="config-backdrop" onClick={onClose} />
      <section className="config-sheet panel">
        <header className="config-sheet-header">
          <div>
            <strong id="config-modal-title"><FileSliders size={16} /> Configuration</strong>
            <span>Runtime sources, editable config, and connection tests.</span>
          </div>
          <button id="config-close" type="button" onClick={onClose}>Close</button>
        </header>
        {config && (
          <ConfigWorkspace
            config={config}
            forms={forms}
            setForms={setForms}
            savedForms={savedForms}
            mutationEnabled={mutationEnabled}
            status={status}
            statusKind={statusKind}
            testResults={testResults}
            onSave={onSave}
            onSaveCaddyEditor={onSaveCaddyEditor}
            onTest={onTest}
          />
        )}
      </section>
    </div>
  );
}

function ConfigWorkspace(props: {
  config: ConfigResponse;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  status: string;
  statusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onSaveCaddyEditor: () => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  const [activeTab, setActiveTab] = useState<ConfigTab>('caddy');
  return (
    <div id="config-summary" className="config-workspace">
      <div id="config-status" className={`config-status ${props.statusKind}`} role="status" aria-live="polite">{props.status}</div>
      <div className="cfg-tab-bar" role="tablist">
        {configTabOrder.map((tab) => (
          <button
            key={tab}
            role="tab"
            aria-selected={activeTab === tab}
            className={`cfg-tab ${activeTab === tab ? 'active' : ''}`}
            onClick={() => setActiveTab(tab)}
          >
            {configTabLabels[tab]}
          </button>
        ))}
      </div>
      <div className="cfg-tab-content">
        {activeTab === 'caddy-editor' ? (
          <CaddyEditorSetupPanel
            form={props.forms.caddyEditor}
            savedForm={props.savedForms.caddyEditor}
            setForm={(updater) => props.setForms((current) => ({ ...current, caddyEditor: typeof updater === 'function' ? updater(current.caddyEditor) : updater }))}
            mutationEnabled={props.mutationEnabled}
            onSave={props.onSaveCaddyEditor}
          />
        ) : (
          <ConfigCard service={activeTab} {...props} />
        )}
      </div>
    </div>
  );
}

function CaddyEditorSetupPanel({
  form,
  savedForm,
  setForm,
  mutationEnabled,
  onSave
}: {
  form: CaddyEditorForm;
  savedForm: CaddyEditorForm;
  setForm: (updater: CaddyEditorForm | ((prev: CaddyEditorForm) => CaddyEditorForm)) => void;
  mutationEnabled: boolean;
  onSave: () => Promise<void>;
}) {
  const dirty = JSON.stringify(form) !== JSON.stringify(savedForm);
  const [availableTemplates, setAvailableTemplates] = useState<string[]>([]);
  const set = <K extends keyof CaddyEditorForm>(key: K, value: CaddyEditorForm[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  useEffect(() => {
    api.caddyTemplates().then((res) => setAvailableTemplates(res.templates)).catch(() => {});
  }, []);
  return (
    <section className="caddy-editor-setup">
      <header className="caddy-editor-setup-header">
        <div>
          <strong><FileCode2 size={15} /> Caddy File Editor</strong>
          <span>Configure the Caddyfile editor -- repo path, deploy command, and git settings.</span>
        </div>
      </header>
      <div className="caddy-editor-setup-body">
        <label className="checkbox-row">
          <input id="ce-enabled" type="checkbox" checked={form.enabled} onChange={(e) => set('enabled', e.target.checked)} />
          Enabled
        </label>

        {/* Paths */}
        <div className="ce-section-label">Paths</div>
        <div className="caddy-editor-setup-grid">
          <Field label="Repo path">
            <input id="ce-repo-path" type="text" value={form.repo_path} placeholder="/etc/caddy" onChange={(e) => set('repo_path', e.target.value)} />
          </Field>
          <Field label="Caddyfile (relative to repo path)">
            <input id="ce-caddyfile" type="text" value={form.caddyfile} placeholder="Caddyfile" onChange={(e) => set('caddyfile', e.target.value)} />
          </Field>
          <Field label="Entry template">
            {availableTemplates.length > 0 ? (
              <div className="select-wrapper">
                <select id="ce-entry-template" value={form.entry_template} onChange={(e) => set('entry_template', e.target.value)}>
                  {availableTemplates.map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
                <ChevronDown size={14} />
              </div>
            ) : (
              <input id="ce-entry-template" type="text" value={form.entry_template} placeholder="default" onChange={(e) => set('entry_template', e.target.value)} />
            )}
          </Field>
        </div>

        {/* Commands -- textarea so the full string is visible */}
        <div className="ce-section-label">Commands</div>
        <div className="caddy-editor-setup-grid">
          <Field label="Deploy command">
            <textarea id="ce-deploy-command" rows={2} value={form.deploy_command} placeholder="make deploy" onChange={(e) => set('deploy_command', e.target.value)} />
          </Field>
          <Field label="Validate command">
            <textarea id="ce-validate-command" rows={2} value={form.validate_command} placeholder="caddy validate --config Caddyfile" onChange={(e) => set('validate_command', e.target.value)} />
          </Field>
        </div>

        {/* Git */}
        <div className="ce-section-label">Git</div>
        <div className="caddy-editor-setup-grid ce-grid-2col">
          <Field label="Remote">
            <input id="ce-git-remote" type="text" value={form.git_remote} placeholder="origin" onChange={(e) => set('git_remote', e.target.value)} />
          </Field>
          <Field label="Branch">
            <input id="ce-git-branch" type="text" value={form.git_branch} placeholder="main" onChange={(e) => set('git_branch', e.target.value)} />
          </Field>
        </div>
        <div className="caddy-editor-setup-checks">
          <label className="checkbox-row">
            <input id="ce-git-auto-commit" type="checkbox" checked={form.git_auto_commit} onChange={(e) => set('git_auto_commit', e.target.checked)} />
            Auto-commit on save
          </label>
          <label className="checkbox-row">
            <input id="ce-git-auto-push" type="checkbox" checked={form.git_auto_push} onChange={(e) => set('git_auto_push', e.target.checked)} />
            Auto-push on save
          </label>
        </div>

        <div className="config-actions">
          <button type="button" id="ce-save" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave()}>Save Caddy Editor Config</button>
        </div>
      </div>
    </section>
  );
}

function ConfigCard({
  service,
  config,
  forms,
  setForms,
  savedForms,
  mutationEnabled,
  testResults,
  onSave,
  onTest
}: {
  service: ServiceKey;
  config: ConfigResponse;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  testResults: TestResults;
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  const summary = config.summary[service] as ConfigServiceSummary | undefined;
  if (!summary) return null;
  const meta = serviceMeta[service];
  const Icon = meta.icon;
  const fields = Object.entries(summary.fields || {});
  const details = Object.entries(summary.details || {}).filter(([, value]) => value);
  const missing = summary.missing || [];
  const tone = summary.client_ready ? 'ok' : summary.enabled ? 'warn' : 'missing';
  return (
    <article className={`config-card ${tone} ${meta.tone}`}>
      <header>
        <div>
          <strong><Icon size={15} /> {summary.label || meta.label}</strong>
          <span>{summary.client_ready ? 'Client ready' : summary.enabled ? 'Configured, incomplete' : 'Not configured'}</span>
        </div>
        <em>{compactSourceKind(summary.source)}</em>
      </header>
      <ConfigLine label="Source" value={sourceLabel(summary.source)} />
      {summary.endpoint && <ConfigLine label="Endpoint" value={summary.endpoint} />}
      {summary.insecure && <ConfigLine label="TLS" value="Insecure verification" warn />}
      {details.map(([key, value]) => <ConfigLine key={key} label={formatConfigKey(key)} value={value} />)}
      {fields.map(([key, value]) => <ConfigLine key={key} label={formatConfigKey(key)} value={value ? 'set' : 'missing'} />)}
      <div className={`missing-list ${missing.length ? '' : 'ok'}`}>{missing.length ? `Missing: ${missing.join(', ')}` : 'Required fields present'}</div>
      <ConfigEditor service={service} forms={forms} setForms={setForms} savedForms={savedForms} mutationEnabled={mutationEnabled} testResult={testResults[service]} onSave={onSave} onTest={onTest} />
    </article>
  );
}

function ConfigLine({ label, value, warn = false }: { label: string; value: string; warn?: boolean }) {
  return <div className={`config-line ${warn ? 'warn' : ''}`}><span>{label}</span><strong>{value}</strong></div>;
}

function ConfigEditor({
  service,
  forms,
  setForms,
  savedForms,
  mutationEnabled,
  testResult,
  onSave,
  onTest
}: {
  service: ServiceKey;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  mutationEnabled: boolean;
  testResult?: { text: string; kind: 'info' | 'ok' | 'error' };
  onSave: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onTest: (service: ServiceKey) => Promise<void>;
}) {
  const isDirty = (key: keyof ConfigForms) =>
    JSON.stringify(forms[key]) !== JSON.stringify(savedForms[key]);

  if (service === 'caddy') {
    return <div className="config-editor compact" data-config-editor="caddy"><ConfigTestResult service={service} result={testResult} /><button type="button" data-config-test="caddy" disabled={!mutationEnabled} onClick={() => void onTest('caddy')}>Test Caddy</button></div>;
  }
  if (service === 'dhcp') return null;
  if (service === 'unbound') {
    const dirty = isDirty('unbound');
    return (
      <div className="config-editor" data-config-editor="unbound">
        <Field label="Base URL"><input id="config-unbound-base-url" type="url" value={forms.unbound.base_url} placeholder="https://opnsense.local" onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, base_url: event.target.value } }))} /></Field>
        <Field label="API key"><input id="config-unbound-api-key" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.unbound.api_key} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, api_key: event.target.value } }))} /></Field>
        <Field label="API secret"><input id="config-unbound-api-secret" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.unbound.api_secret} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, api_secret: event.target.value } }))} /></Field>
        <label className="checkbox-row"><input id="config-unbound-insecure" type="checkbox" checked={forms.unbound.insecure} onChange={(event) => setForms((current) => ({ ...current, unbound: { ...current.unbound, insecure: event.target.checked } }))} /> Insecure TLS</label>
        <ConfigTestResult service={service} result={testResult} />
        <div className="config-actions">
          <button type="button" data-config-test="unbound" disabled={!mutationEnabled} onClick={() => void onTest('unbound')}>Test OPNSense</button>
          <button type="button" data-config-save="unbound" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave('unbound')}>Save OPNSense</button>
        </div>
      </div>
    );
  }
  if (service === 'adguard') {
    const dirty = isDirty('adguard');
    return (
      <div className="config-editor" data-config-editor="adguard">
        <label className="checkbox-row"><input id="config-adguard-enabled" type="checkbox" checked={forms.adguard.enabled} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, enabled: event.target.checked } }))} /> Enabled</label>
        <Field label="Base URL"><input id="config-adguard-base-url" type="url" value={forms.adguard.base_url} placeholder="https://adguard.local" onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, base_url: event.target.value } }))} /></Field>
        <Field label="Username"><input id="config-adguard-username" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.adguard.username} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, username: event.target.value } }))} /></Field>
        <Field label="Password"><input id="config-adguard-password" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.adguard.password} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, password: event.target.value } }))} /></Field>
        <label className="checkbox-row"><input id="config-adguard-insecure" type="checkbox" checked={forms.adguard.insecure} onChange={(event) => setForms((current) => ({ ...current, adguard: { ...current.adguard, insecure: event.target.checked } }))} /> Insecure TLS</label>
        <ConfigTestResult service={service} result={testResult} />
        <div className="config-actions">
          <button type="button" data-config-test="adguard" disabled={!mutationEnabled} onClick={() => void onTest('adguard')}>Test AdGuard</button>
          <button type="button" data-config-save="adguard" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave('adguard')}>Save AdGuard</button>
        </div>
      </div>
    );
  }
  const dirty = isDirty('cloudflare');
  return (
    <div className="config-editor" data-config-editor="cloudflare">
      <label className="checkbox-row"><input id="config-cloudflare-enabled" type="checkbox" checked={forms.cloudflare.enabled} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, enabled: event.target.checked } }))} /> Enabled</label>
      <Field label="API token"><input id="config-cloudflare-api-token" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.api_token} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, api_token: event.target.value } }))} /></Field>
      <Field label="Account ID"><input id="config-cloudflare-account-id" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.account_id} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, account_id: event.target.value } }))} /></Field>
      <Field label="Zone ID"><input id="config-cloudflare-zone-id" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.zone_id} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, zone_id: event.target.value } }))} /></Field>
      <Field label="Tunnel ID"><input id="config-cloudflare-tunnel-id" type="password" autoComplete="off" placeholder="leave unchanged" value={forms.cloudflare.tunnel_id} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, tunnel_id: event.target.value } }))} /></Field>
      <Field label="Caddy service URL"><input id="config-cloudflare-caddy-service-url" type="url" value={forms.cloudflare.caddy_service_url} placeholder="http://127.0.0.1:80" onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, caddy_service_url: event.target.value } }))} /></Field>
      <label className="checkbox-row"><input id="config-cloudflare-insecure" type="checkbox" checked={forms.cloudflare.insecure} onChange={(event) => setForms((current) => ({ ...current, cloudflare: { ...current.cloudflare, insecure: event.target.checked } }))} /> Insecure TLS</label>
      <ConfigTestResult service={service} result={testResult} />
      <div className="config-actions">
        <button type="button" data-config-test="cloudflare" disabled={!mutationEnabled} onClick={() => void onTest('cloudflare')}>Test Cloudflare</button>
        <button type="button" data-config-save="cloudflare" className={dirty ? 'btn-primary' : ''} disabled={!mutationEnabled} onClick={() => void onSave('cloudflare')}>Save Cloudflare</button>
      </div>
    </div>
  );
}

function ConfigTestResult({ service, result }: { service: ServiceKey; result?: { text: string; kind: 'info' | 'ok' | 'error' } }) {
  return <div id={`config-test-${service}`} className={`config-test-result ${result?.kind || ''}`} role="status" aria-live="polite">{result?.text || ''}</div>;
}
