import {
  GitBranch,
  Loader2,
  RefreshCw,
  X
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import type { Dispatch, SetStateAction } from 'react';
import { api } from '../api/client';
import { AuthFlowsTab } from './AuthFlowsTab';
import { CaddyEditor } from './CaddyEditor';
import { CFRepairBanner } from './CloudflarePanel';
import { ConfigModal } from './ConfigModal';
import { DiagnosticsTab } from './DiagnosticsTab';
import { EntriesTable } from './EntriesTable';
import { EntriesToolbar } from './EntriesToolbar';
import { LogBar } from './LogBar';
import { MetricGrid } from './MetricCards';
import { OperationsHeader } from './OperationsHeader';
import { SyncModal } from './SyncModal';
import { Topbar } from './Topbar';
import type { TabId } from './TabBar';
import type { CaddyEditorForm, ConfigForms, TestResults } from '../hooks/useConfigForms';
import type { ConfigResponse, EntriesResponse, Entry, ServiceKey, SyncAction } from '../types';

export function AppShell({
  view,
  setView,
  config,
  loading,
  message,
  messageKind,
  progress,
  report,
  summary,
  statusFilter,
  setStatusFilter,
  serviceFilter,
  setServiceFilter,
  search,
  setSearch,
  entries,
  selectedEntry,
  selectedHostname,
  setSelectedHostname,
  suppressed,
  onToggleSuppress,
  mutationEnabled,
  syncService,
  setSyncService,
  syncLoading,
  syncProgress,
  syncLog,
  plannedActions,
  canSyncNow,
  onRefresh,
  onPreview,
  onDryRun,
  onSync,
  onRemoveEntry,
  onSyncAll,
  configOpen,
  setConfigOpen,
  forms,
  setForms,
  savedForms,
  configStatus,
  configStatusKind,
  testResults,
  onSaveConfig,
  onSaveCaddyEditor,
  onTestConfig
}: {
  view: TabId;
  setView: (v: TabId) => void;
  config: ConfigResponse | null;
  loading: boolean;
  message: string;
  messageKind: 'info' | 'error' | 'ok';
  progress: Record<string, import('../types').ProgressEvent>;
  report: EntriesResponse['report'];
  summary: { entries: number; inSync: number; out: number; caddyOnly: number; stale: number; cloudflare: number; issues: number };
  statusFilter: string;
  setStatusFilter: (value: string) => void;
  serviceFilter: string;
  setServiceFilter: (value: string) => void;
  search: string;
  setSearch: (value: string) => void;
  entries: Entry[];
  selectedEntry?: Entry;
  selectedHostname: string;
  setSelectedHostname: (value: string) => void;
  suppressed: Set<string>;
  onToggleSuppress: (key: string) => void;
  mutationEnabled: boolean;
  syncService: string;
  setSyncService: (value: string) => void;
  syncLoading: boolean;
  syncProgress: { title: string; detail: string };
  syncLog: string;
  plannedActions: SyncAction[];
  canSyncNow: boolean;
  onRefresh: () => void;
  onPreview: (service?: string, hostname?: string) => Promise<boolean>;
  onDryRun: () => Promise<void>;
  onSync: () => Promise<void>;
  onRemoveEntry: (hostname: string, service?: string) => Promise<void>;
  onSyncAll: () => Promise<void>;
  configOpen: boolean;
  setConfigOpen: (value: boolean) => void;
  forms: ConfigForms;
  setForms: Dispatch<SetStateAction<ConfigForms>>;
  savedForms: ConfigForms;
  configStatus: string;
  configStatusKind: 'info' | 'error' | 'ok';
  testResults: TestResults;
  onSaveConfig: (service: 'unbound' | 'adguard' | 'cloudflare') => Promise<void>;
  onSaveCaddyEditor: () => Promise<void>;
  onTestConfig: (service: ServiceKey) => Promise<void>;
}) {
  const enabledServices = config?.enabled || {};
  const [modalOpen, setModalOpen] = useState(false);
  const [modalHostname, setModalHostname] = useState('');
  const [modalAutoSync, setModalAutoSync] = useState(false);
  const [logBarOpen, setLogBarOpen] = useState(false);

  // Global remote-ahead status -- polled every 60s regardless of active tab.
  type GitRemoteStatus = { remote_ahead: number; local_ahead: number; branch: string; remote: string; fetch_error?: string };
  const [remoteStatus, setRemoteStatus] = useState<GitRemoteStatus | null>(null);
  const [remoteChecking, setRemoteChecking] = useState(false);
  const [pulling, setPulling] = useState(false);
  const [pullOutput, setPullOutput] = useState('');

  const checkRemote = useCallback(async () => {
    setRemoteChecking(true);
    try { setRemoteStatus(await api.caddyGitStatus()); } catch { /* non-fatal */ } finally { setRemoteChecking(false); }
  }, []);

  const handlePull = useCallback(async () => {
    setPulling(true); setPullOutput('');
    try {
      const res = await api.caddyGitPull();
      setPullOutput(res.output || 'Already up to date.');
      await checkRemote();
      onRefresh();
    } catch (err) { setPullOutput(`Error: ${String(err)}`); }
    finally { setPulling(false); }
  }, [checkRemote, onRefresh]);

  useEffect(() => {
    void checkRemote();
    const id = setInterval(() => { void checkRemote(); }, 60_000);
    return () => clearInterval(id);
  }, [checkRemote]);

  const modalEntry = entries.find(e => e.hostname === modalHostname);

  const openModify = (hostname: string) => {
    setModalHostname(hostname);
    setSelectedHostname(hostname);
    setModalAutoSync(false);
    setModalOpen(true);
  };

  const openQuickSync = (hostname: string) => {
    setModalHostname(hostname);
    setSelectedHostname(hostname);
    setModalAutoSync(true);
    setModalOpen(true);
  };

  const showRemoteBanner = remoteStatus && (remoteStatus.remote_ahead > 0 || remoteStatus.fetch_error);

  return (
    <div className="console-layout">
      <div className="console-main">
        <Topbar config={config} loading={loading} syncLoading={syncLoading} view={view} setView={setView} onRefresh={onRefresh} onOpenConfig={() => setConfigOpen(true)} />

        {showRemoteBanner && (
          <div className={`git-remote-banner global ${remoteStatus!.fetch_error ? 'error' : 'warn'}`}>
            <GitBranch size={14} />
            {remoteStatus!.fetch_error ? (
              <span>Caddy repo: could not reach remote — {remoteStatus!.fetch_error}</span>
            ) : (
              <span>
                Caddy repo: remote <strong>{remoteStatus!.remote}/{remoteStatus!.branch}</strong> is{' '}
                <strong>{remoteStatus!.remote_ahead}</strong> commit{remoteStatus!.remote_ahead !== 1 ? 's' : ''} ahead — pull before deploying
              </span>
            )}
            <div className="git-remote-banner-actions">
              {remoteStatus!.remote_ahead > 0 && mutationEnabled && (
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
          <div className="git-pull-output global">
            <pre>{pullOutput}</pre>
            <button type="button" className="git-pull-dismiss" onClick={() => setPullOutput('')}><X size={13} /></button>
          </div>
        )}

        <div className="console-scroll">
          {view === 'caddyfile' ? (
            <main className="dashboard-shell caddy-editor-shell">
              <CaddyEditor mutationEnabled={mutationEnabled} remoteStatus={remoteStatus} remoteChecking={remoteChecking} pulling={pulling} onPull={handlePull} onCheckRemote={checkRemote} />
            </main>
          ) : view === 'auth' ? (
            <AuthFlowsTab />
          ) : view === 'diagnostics' ? (
            <DiagnosticsTab />
          ) : (
            <main className="dashboard-shell">
              <OperationsHeader
                loading={loading}
                message={message}
                messageKind={messageKind}
                progress={progress}
                summary={summary}
              />
              <MetricGrid summary={summary} statusFilter={statusFilter} setStatusFilter={setStatusFilter} />
              <EntriesToolbar
                entriesCount={entries.length}
                statusFilter={statusFilter}
                setStatusFilter={setStatusFilter}
                serviceFilter={serviceFilter}
                setServiceFilter={setServiceFilter}
                search={search}
                setSearch={setSearch}
                enabledServices={enabledServices}
                mutationEnabled={mutationEnabled}
                onSyncAll={onSyncAll}
              />
              <CFRepairBanner
                entries={entries}
                mutationEnabled={mutationEnabled}
                cfEnabled={enabledServices.cloudflare === true}
                onRepaired={onRefresh}
              />
              <EntriesTable
                entries={entries}
                selectedHostname={selectedHostname}
                mutationEnabled={mutationEnabled}
                enabledServices={enabledServices}
                caddyServerIP={config?.caddy.server_ip || ''}
                suppressed={suppressed}
                onToggleSuppress={onToggleSuppress}
                onSelect={setSelectedHostname}
                onQuickSync={openQuickSync}
                onOpenModify={openModify}
                onRemove={onRemoveEntry}
              />
            </main>
          )}
        </div>
        <LogBar open={logBarOpen} onToggle={() => setLogBarOpen(o => !o)} />
      </div>
      <SyncModal
        open={modalOpen}
        autoSync={modalAutoSync}
        onClose={() => setModalOpen(false)}
        hostname={modalHostname}
        entry={modalEntry}
        enabledServices={enabledServices}
        caddyServerIP={config?.caddy.server_ip || ''}
        suppressed={suppressed}
        onToggleSuppress={onToggleSuppress}
        syncService={syncService}
        setSyncService={setSyncService}
        syncLoading={syncLoading}
        syncProgress={syncProgress}
        syncLog={syncLog}
        plannedActions={plannedActions}
        canSyncNow={canSyncNow}
        mutationEnabled={mutationEnabled}
        onPreviewFor={(service, hostname) => onPreview(service, hostname)}
        onDryRun={onDryRun}
        onSync={onSync}
        onRefresh={onRefresh}
        onRemoveEntry={onRemoveEntry}
      />
      <ConfigModal
        open={configOpen}
        onClose={() => setConfigOpen(false)}
        config={config}
        forms={forms}
        setForms={setForms}
        savedForms={savedForms}
        mutationEnabled={mutationEnabled}
        status={configStatus}
        statusKind={configStatusKind}
        testResults={testResults}
        onSave={onSaveConfig}
        onSaveCaddyEditor={onSaveCaddyEditor}
        onTest={onTestConfig}
      />
    </div>
  );
}
