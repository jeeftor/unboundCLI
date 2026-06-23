import {
  CheckCircle2,
  ChevronDown,
  Edit3,
  FileCode2,
  GitBranch,
  Loader2,
  Play,
  Plus,
  RefreshCw,
  ShieldCheck,
  Terminal,
  Trash2,
  X,
  XCircle
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../api/client';
import type { CaddyDeployEvent, CaddyEntry, CaddyEntriesResponse, CaddyValidateResult } from '../types';

// ─── Main CaddyEditor panel ───────────────────────────────────────────────

type GitRemoteStatus = { remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string };

export function CaddyEditor({ mutationEnabled }: { mutationEnabled: boolean }) {
  const [data, setData] = useState<CaddyEntriesResponse | null>(null);
  const [templates, setTemplates] = useState<string[]>([]);
  const [defaultTemplate, setDefaultTemplate] = useState('default');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [diff, setDiff] = useState('');
  const [gitStatus, setGitStatus] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<CaddyEntry | null>(null);
  const [deployOpen, setDeployOpen] = useState(false);

  // Remote-ahead state — polled every 60s and on load.
  const [remoteStatus, setRemoteStatus] = useState<GitRemoteStatus | null>(null);
  const [remoteChecking, setRemoteChecking] = useState(false);
  const [pulling, setPulling] = useState(false);
  const [pullOutput, setPullOutput] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [res, tmpl] = await Promise.all([
        api.caddyEntries(),
        api.caddyTemplates()
      ]);
      setData(res);
      setTemplates(tmpl.templates);
      setDefaultTemplate(tmpl.default);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  const loadDiff = useCallback(async () => {
    try {
      const res = await api.caddyDiff();
      setDiff(res.diff);
      setGitStatus(res.status);
    } catch {
      // ignore diff errors
    }
  }, []);

  const checkRemote = useCallback(async () => {
    setRemoteChecking(true);
    try {
      const st = await api.caddyGitStatus();
      setRemoteStatus(st);
    } catch {
      // non-fatal
    } finally {
      setRemoteChecking(false);
    }
  }, []);

  const handlePull = useCallback(async () => {
    setPulling(true);
    setPullOutput('');
    try {
      const res = await api.caddyGitPull();
      setPullOutput(res.output || 'Already up to date.');
      // Refresh everything after pull.
      await Promise.all([load(), loadDiff(), checkRemote()]);
    } catch (err) {
      setPullOutput(`Error: ${String(err)}`);
    } finally {
      setPulling(false);
    }
  }, [load, loadDiff, checkRemote]);

  useEffect(() => { void load(); void loadDiff(); }, [load, loadDiff]);

  // Check remote on mount, then every 60s.
  useEffect(() => {
    void checkRemote();
    const id = setInterval(() => { void checkRemote(); }, 60_000);
    return () => clearInterval(id);
  }, [checkRemote]);

  const handleSaved = useCallback(async () => {
    setModalOpen(false);
    setEditTarget(null);
    await load();
    await loadDiff();
  }, [load, loadDiff]);

  const handleDelete = useCallback(async (hostname: string) => {
    if (!confirm(`Remove ${hostname} from Caddyfile?`)) return;
    try {
      await api.caddyDeleteEntry(hostname);
      await load();
      await loadDiff();
    } catch (err) {
      setError(String(err));
    }
  }, [load, loadDiff]);

  if (!data?.editor.enabled && !loading) {
    return (
      <div className="caddy-editor-disabled">
        <FileCode2 size={40} />
        <h3>Caddy Editor is not enabled</h3>
        <p>
          Add a <code>caddy_editor</code> section to your <code>~/.caddy-dns-sync.json</code>{' '}
          config file with <code>"enabled": true</code> and <code>"repo_path"</code> pointing to
          your Caddy git repository.
        </p>
      </div>
    );
  }

  return (
    <div className="caddy-editor">
      <div className="caddy-editor-header">
        <div>
          <h2>Caddy Editor</h2>
          <p>Add, edit, and remove reverse-proxy entries from your Caddyfile.</p>
        </div>
        <div className="caddy-editor-actions">
          <button type="button" onClick={() => { void load(); void loadDiff(); }} disabled={loading}>
            <RefreshCw size={15} /> Refresh
          </button>
          {mutationEnabled && (
            <button type="button" className="btn-primary" onClick={() => { setEditTarget(null); setModalOpen(true); }}>
              <Plus size={15} /> Add entry
            </button>
          )}
          {mutationEnabled && (
            <button type="button" className="btn-deploy" onClick={() => setDeployOpen(true)}>
              <Play size={15} /> Deploy
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="caddy-error">
          <XCircle size={15} /> {error}
        </div>
      )}

      {/* Remote-ahead / local-ahead banner */}
      {remoteStatus && (remoteStatus.remote_ahead > 0 || remoteStatus.local_ahead > 0 || remoteStatus.fetch_error) && (
        <div className={`git-remote-banner ${remoteStatus.fetch_error ? 'error' : remoteStatus.remote_ahead > 0 ? 'warn' : 'info'}`}>
          <GitBranch size={14} />
          {remoteStatus.fetch_error ? (
            <span>Could not reach remote: {remoteStatus.fetch_error}</span>
          ) : remoteStatus.remote_ahead > 0 ? (
            <span>
              Remote <strong>{remoteStatus.remote}/{remoteStatus.branch}</strong> is{' '}
              <strong>{remoteStatus.remote_ahead}</strong> commit{remoteStatus.remote_ahead !== 1 ? 's' : ''} ahead of local
              {remoteStatus.local_ahead > 0 && ` · ${remoteStatus.local_ahead} local commit${remoteStatus.local_ahead !== 1 ? 's' : ''} not pushed`}
            </span>
          ) : (
            <span>
              <strong>{remoteStatus.local_ahead}</strong> local commit{remoteStatus.local_ahead !== 1 ? 's' : ''} not yet pushed to {remoteStatus.remote}
            </span>
          )}
          <div className="git-remote-banner-actions">
            {remoteStatus.remote_ahead > 0 && mutationEnabled && (
              <button type="button" className="btn-sm btn-warn" onClick={() => void handlePull()} disabled={pulling}>
                {pulling ? <><Loader2 size={12} className="spin" /> Pulling…</> : '⬇ Pull'}
              </button>
            )}
            <button type="button" className="btn-sm" onClick={() => void checkRemote()} disabled={remoteChecking}>
              {remoteChecking ? <Loader2 size={12} className="spin" /> : <RefreshCw size={12} />}
            </button>
          </div>
        </div>
      )}

      {pullOutput && (
        <div className="git-pull-output">
          <pre>{pullOutput}</pre>
          <button type="button" className="git-pull-dismiss" onClick={() => setPullOutput('')}><X size={13} /></button>
        </div>
      )}

      {loading ? (
        <div className="caddy-loading"><Loader2 size={20} className="spin" /> Loading entries...</div>
      ) : (
        <EntriesTable
          entries={data?.entries ?? []}
          mutationEnabled={mutationEnabled}
          onEdit={(entry) => { setEditTarget(entry); setModalOpen(true); }}
          onDelete={handleDelete}
        />
      )}

      {(diff || gitStatus) && (
        <DiffPanel diff={diff} status={gitStatus} />
      )}

      {modalOpen && (
        <EntryModal
          entry={editTarget}
          templates={templates}
          defaultTemplate={defaultTemplate}
          repoPath={data?.editor.repo_path ?? ''}
          onClose={() => { setModalOpen(false); setEditTarget(null); }}
          onSaved={handleSaved}
        />
      )}

      {deployOpen && (
        <DeployPanel
          onClose={() => { setDeployOpen(false); void loadDiff(); }}
        />
      )}
    </div>
  );
}

// ─── Entries table ────────────────────────────────────────────────────────

function EntriesTable({
  entries,
  mutationEnabled,
  onEdit,
  onDelete
}: {
  entries: CaddyEntry[];
  mutationEnabled: boolean;
  onEdit: (entry: CaddyEntry) => void;
  onDelete: (hostname: string) => void;
}) {
  if (entries.length === 0) {
    return (
      <div className="caddy-empty">
        <FileCode2 size={32} />
        <p>No entries found in the services directory.</p>
      </div>
    );
  }

  return (
    <section className="panel caddy-entries-panel">
      <table>
        <thead>
          <tr>
            <th>Hostname</th>
            <th>Upstream</th>
            <th>Directives</th>
            <th>File</th>
            {mutationEnabled && <th>Actions</th>}
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr key={entry.hostname}>
              <td><code>{entry.hostname}</code></td>
              <td><code>{entry.upstream}</code></td>
              <td className="caddy-directives">
                {entry.directives?.length > 0
                  ? entry.directives.map((d, i) => <span key={i} className="directive-tag">{d}</span>)
                  : <span className="muted">—</span>}
              </td>
              <td className="caddy-source">{entry.source_file?.split('/').slice(-2).join('/')}</td>
              {mutationEnabled && (
                <td className="caddy-row-actions">
                  <button type="button" onClick={() => onEdit(entry)} title="Edit">
                    <Edit3 size={14} />
                  </button>
                  <button type="button" className="btn-danger" onClick={() => onDelete(entry.hostname)} title="Remove">
                    <Trash2 size={14} />
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

// ─── Add / Edit modal ─────────────────────────────────────────────────────

function EntryModal({
  entry,
  templates,
  defaultTemplate,
  repoPath,
  onClose,
  onSaved
}: {
  entry: CaddyEntry | null;
  templates: string[];
  defaultTemplate: string;
  repoPath: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const isEdit = Boolean(entry);
  const [hostname, setHostname] = useState(entry?.hostname ?? '');
  const [upstream, setUpstream] = useState(entry?.upstream ?? '');
  const [template, setTemplate] = useState(defaultTemplate);
  const [preview, setPreview] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [validating, setValidating] = useState(false);
  const [validateResult, setValidateResult] = useState<CaddyValidateResult | null>(null);
  const [error, setError] = useState('');

  // Reset validation whenever inputs change.
  useEffect(() => { setValidateResult(null); }, [hostname, upstream, template]);

  const updatePreview = useCallback(async (h: string, u: string, t: string) => {
    if (!h || !u) { setPreview(''); return; }
    setPreviewLoading(true);
    try {
      const res = await api.caddyPreview(h, u, t);
      setPreview(res.content);
    } catch {
      setPreview('');
    } finally {
      setPreviewLoading(false);
    }
  }, []);

  useEffect(() => {
    void updatePreview(hostname, upstream, template);
  }, [hostname, upstream, template, updatePreview]);

  // Load initial preview for edit mode.
  useEffect(() => {
    if (entry) void updatePreview(entry.hostname, entry.upstream, template);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleValidate = async () => {
    if (!hostname.trim() || !upstream.trim()) {
      setError('Hostname and upstream are required.');
      return;
    }
    setValidating(true);
    setError('');
    setValidateResult(null);
    try {
      const res = await api.caddyValidateDraft({ hostname, upstream, template });
      setValidateResult(res);
    } catch (err) {
      setValidateResult({ ok: false, output: String(err) });
    } finally {
      setValidating(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      if (isEdit) {
        await api.caddyUpdateEntry(hostname, { upstream, template });
      } else {
        await api.caddyCreateEntry({ hostname, upstream, template });
      }
      onSaved();
    } catch (err) {
      setError(String(err));
    } finally {
      setSaving(false);
    }
  };

  const canSave = validateResult?.ok === true;

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="modal caddy-modal">
        <div className="modal-header">
          <h3>{isEdit ? 'Edit entry' : 'Add entry'}</h3>
          <button type="button" className="modal-close" onClick={onClose}><X size={18} /></button>
        </div>
        <div className="modal-body">
          <label>
            Hostname
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="myapp.example.com"
              disabled={isEdit}
            />
          </label>
          <label>
            Upstream
            <input
              type="text"
              value={upstream}
              onChange={(e) => setUpstream(e.target.value)}
              placeholder="10.0.0.100:3000"
            />
          </label>
          <label>
            Template
            <div className="select-wrapper">
              <select value={template} onChange={(e) => setTemplate(e.target.value)}>
                {templates.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
              <ChevronDown size={14} />
            </div>
          </label>

          {(preview || previewLoading) && (
            <div className="preview-block">
              <span className="preview-label">Preview</span>
              {previewLoading
                ? <div className="preview-loading"><Loader2 size={14} className="spin" /></div>
                : <pre>{preview}</pre>}
            </div>
          )}

          {validateResult && (
            <div className={`validate-result ${validateResult.ok ? 'ok' : 'error'}`}>
              {validateResult.ok
                ? <><CheckCircle2 size={14} /> Config valid — ready to write</>
                : <><XCircle size={14} /> {validateResult.output || 'Validation failed'}</>}
            </div>
          )}

          {error && <div className="caddy-error"><XCircle size={14} /> {error}</div>}
        </div>
        <div className="modal-footer">
          <button type="button" onClick={onClose}>Cancel</button>
          <button type="button" onClick={() => void handleValidate()} disabled={validating || saving}>
            {validating ? <><Loader2 size={14} className="spin" /> Validating...</> : <><ShieldCheck size={14} /> Validate</>}
          </button>
          <button type="button" className="btn-primary" onClick={() => void handleSave()} disabled={saving || !canSave}>
            {saving ? <><Loader2 size={14} className="spin" /> Saving...</> : 'Write to file'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Git diff panel ───────────────────────────────────────────────────────

function DiffPanel({ diff, status }: { diff: string; status: string }) {
  const [open, setOpen] = useState(false);
  if (!diff && !status) return null;

  const changedCount = status.split('\n').filter((l) => l.trim()).length;

  return (
    <section className="caddy-diff-panel panel">
      <button type="button" className="diff-toggle" onClick={() => setOpen(!open)}>
        <GitBranch size={14} />
        <span>Git status: {changedCount > 0 ? `${changedCount} file${changedCount !== 1 ? 's' : ''} changed` : 'clean'}</span>
        <ChevronDown size={14} className={open ? 'rotated' : ''} />
      </button>
      {open && diff && (
        <pre className="diff-content">{diff}</pre>
      )}
    </section>
  );
}

// ─── Deploy panel (SSE streamed) ──────────────────────────────────────────

function DeployPanel({ onClose }: { onClose: () => void }) {
  const [log, setLog] = useState<string[]>([]);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<'ok' | 'error' | null>(null);
  const [validateResult, setValidateResult] = useState<CaddyValidateResult | null>(null);
  const [validating, setValidating] = useState(false);
  const logRef = useRef<HTMLPreElement>(null);

  const handleValidate = async () => {
    setValidating(true);
    try {
      const res = await api.caddyValidate();
      setValidateResult(res);
    } catch (err) {
      setValidateResult({ ok: false, output: String(err) });
    } finally {
      setValidating(false);
    }
  };

  const handleDeploy = async () => {
    setRunning(true);
    setLog([]);
    setResult(null);

    const response = await fetch('/api/caddy/deploy', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-UnboundCLI-Token': window.UNBOUNDCLI_WEB_CONFIG?.applyToken ?? ''
      },
      body: JSON.stringify({})
    });

    if (!response.ok || !response.body) {
      setLog(['Error: deploy request failed']);
      setResult('error');
      setRunning(false);
      return;
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n\n');
      buffer = parts.pop() ?? '';
      for (const part of parts) {
        const dataLine = part.split('\n').find((l) => l.startsWith('data: '));
        if (!dataLine) continue;
        try {
          const event = JSON.parse(dataLine.slice(6)) as CaddyDeployEvent;
          if ('done' in event && event.done) {
            setResult(event.status);
            setRunning(false);
          } else if ('line' in event) {
            setLog((prev) => [...prev, event.line]);
          }
        } catch {
          // skip malformed events
        }
      }
    }

    setRunning(false);
  };

  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [log]);

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget && !running) onClose(); }}>
      <div className="modal caddy-modal caddy-deploy-modal">
        <div className="modal-header">
          <h3><Terminal size={16} /> Deploy</h3>
          <button type="button" className="modal-close" onClick={onClose} disabled={running}><X size={18} /></button>
        </div>
        <div className="modal-body">
          <div className="deploy-actions">
            <button type="button" onClick={() => void handleValidate()} disabled={validating || running}>
              {validating ? <Loader2 size={14} className="spin" /> : <CheckCircle2 size={14} />}
              Validate config
            </button>
            <button type="button" className="btn-deploy" onClick={() => void handleDeploy()} disabled={running}>
              {running ? <Loader2 size={14} className="spin" /> : <Play size={14} />}
              {running ? 'Deploying...' : 'Deploy'}
            </button>
          </div>

          {validateResult && (
            <div className={`validate-result ${validateResult.ok ? 'ok' : 'error'}`}>
              {validateResult.ok ? <CheckCircle2 size={14} /> : <XCircle size={14} />}
              {validateResult.ok ? ' Config valid' : ' Validation failed'}
              {validateResult.output && <pre>{validateResult.output}</pre>}
            </div>
          )}

          {log.length > 0 && (
            <pre className="deploy-log" ref={logRef}>
              {log.join('\n')}
            </pre>
          )}

          {result && (
            <div className={`deploy-result ${result}`}>
              {result === 'ok'
                ? <><CheckCircle2 size={14} /> Deployment successful</>
                : <><XCircle size={14} /> Deployment failed</>}
            </div>
          )}
        </div>
        <div className="modal-footer">
          <button type="button" onClick={onClose} disabled={running}>Close</button>
        </div>
      </div>
    </div>
  );
}
