import '../styles/CaddyEditor.css';
import {
  CheckCircle2,
  Edit3,
  FileCode2,
  Play,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  X,
  XCircle
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../api/client';
import type { CaddyEntry, CaddyEntriesResponse } from '../types';
import { LoadingSpinner } from './LoadingSpinner';
import { EntryModal } from './CaddyEntryModal';
import { DiffPanel } from './CaddyDiffPanel';
import { DeployPanel } from './CaddyDeployPanel';

// ─── Main CaddyEditor panel ───────────────────────────────────────────────

type GitRemoteStatus = { remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string };

export function CaddyEditor({
  mutationEnabled,
  remoteStatus,
  remoteChecking,
  pulling,
  onPull,
  onCheckRemote
}: {
  mutationEnabled: boolean;
  remoteStatus: GitRemoteStatus | null;
  remoteChecking: boolean;
  pulling: boolean;
  onPull: () => Promise<void>;
  onCheckRemote: () => Promise<void>;
}) {
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

  useEffect(() => { void load(); void loadDiff(); }, [load, loadDiff]);

  const handleSaved = useCallback(async () => {
    setModalOpen(false);
    setEditTarget(null);
    await Promise.all([load(), loadDiff(), onCheckRemote()]);
  }, [load, loadDiff, onCheckRemote]);

  const handleDelete = useCallback(async (hostname: string) => {
    if (!confirm(`Remove ${hostname} from Caddyfile?`)) return;
    try {
      await api.caddyDeleteEntry(hostname);
      await Promise.all([load(), loadDiff(), onCheckRemote()]);
    } catch (err) {
      setError(String(err));
    }
  }, [load, loadDiff, onCheckRemote]);

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
          <button type="button" onClick={() => { void load(); void loadDiff(); void onCheckRemote(); }} disabled={loading}>
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

      {pullOutput && (
        <div className="git-pull-output">
          <pre>{pullOutput}</pre>
          <button type="button" className="git-pull-dismiss" onClick={() => setPullOutput('')}><X size={13} /></button>
        </div>
      )}

      {loading ? (
        <div className="caddy-loading"><LoadingSpinner size={20} /> Loading entries...</div>
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
          onClose={() => { setDeployOpen(false); void loadDiff(); void onCheckRemote(); }}
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
  const [search, setSearch] = useState('');
  const [healthFilter, setHealthFilter] = useState<'all' | 'stale'>('all');

  const q = search.trim().toLowerCase();
  const visible = entries.filter((entry) => {
    if (healthFilter === 'stale' && entry.upstream_status !== 'stale') return false;
    return !q || (
      entry.hostname.toLowerCase().includes(q) ||
      entry.upstream.toLowerCase().includes(q) ||
      (entry.source_file ?? '').toLowerCase().includes(q)
    );
  });

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
      <div className="caddy-search-bar">
        <Search size={14} />
        <input
          type="text"
          placeholder="Search hostname, upstream, or file…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        {search && (
          <button type="button" className="caddy-search-clear" onClick={() => setSearch('')}>
            <X size={13} />
          </button>
        )}
        {(q || healthFilter !== 'all') && <span className="caddy-search-count">{visible.length} / {entries.length}</span>}
        <label className="caddy-health-filter">
          <span>Upstream</span>
          <select value={healthFilter} onChange={(event) => setHealthFilter(event.target.value as 'all' | 'stale')}>
            <option value="all">All</option>
            <option value="stale">Stale only</option>
          </select>
        </label>
      </div>
      <table>
        <thead>
          <tr>
            <th>Hostname</th>
            <th>Upstream</th>
            <th>Health</th>
            <th>Directives</th>
            <th>File</th>
            {mutationEnabled && <th>Actions</th>}
          </tr>
        </thead>
        <tbody>
          {visible.map((entry) => (
            <tr key={entry.hostname}>
              <td><code>{entry.hostname}</code></td>
              <td><code>{entry.upstream}</code></td>
              <td>
                <UpstreamHealth entry={entry} />
              </td>
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
          {q && visible.length === 0 && (
            <tr><td colSpan={mutationEnabled ? 6 : 5} className="caddy-no-results">No matching entries</td></tr>
          )}
        </tbody>
      </table>
    </section>
  );
}

function UpstreamHealth({ entry }: { entry: CaddyEntry }) {
  const status = entry.upstream_status ?? 'unknown';
  const label = status === 'reachable' ? 'Reachable' : status === 'stale' ? 'Stale' : 'Unknown';
  return (
    <span className={`caddy-health ${status}`} title={entry.upstream_error || `TCP probe: ${label.toLowerCase()}`}>
      {status === 'reachable' ? <CheckCircle2 size={13} /> : <XCircle size={13} />}
      {label}
    </span>
  );
}
